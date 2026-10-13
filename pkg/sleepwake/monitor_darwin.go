//go:build darwin
// +build darwin

package sleepwake

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>
#include <IOKit/pwr_mgt/IOPMLib.h>
#include <IOKit/IOMessage.h>

// TECHNICAL DEBT: kIOMasterPortDefault is deprecated in macOS 12.0+
// See docs/archive/roadmap/TECHNICAL_DEBT_BACKLOG.md #12
// TODO(v0.6.0): Replace with kIOMainPortDefault when minimum macOS version is raised
// Alternative: Use IOMainPort(kNilOptions, &mainPort) for dynamic acquisition
// Current implementation works correctly on all macOS versions but produces compiler warning

// Forward declaration
extern void goSleepWakeCallback(uintptr_t, int, long);

// C callback that forwards to Go
static inline void sleepWakeCallbackC(void *refcon, io_service_t service, natural_t messageType, void *messageArgument) {
    goSleepWakeCallback((uintptr_t)refcon, (int)messageType, (long)messageArgument);
}

// Register for sleep/wake notifications
static inline IOReturn registerSleepWake(io_connect_t *connection, IONotificationPortRef *notifyPort, io_object_t *notifier, uintptr_t refcon) {
    *notifyPort = IONotificationPortCreate(kIOMainPortDefault);
    if (*notifyPort == NULL) {
        return kIOReturnError;
    }

    // Get the run loop source
    CFRunLoopSourceRef runLoopSource = IONotificationPortGetRunLoopSource(*notifyPort);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), runLoopSource, kCFRunLoopDefaultMode);

    // Register for sleep/wake notifications
    *connection = IORegisterForSystemPower((void*)refcon, notifyPort, sleepWakeCallbackC, notifier);
    if (*connection == 0) {
        return kIOReturnError;
    }

    return kIOReturnSuccess;
}

// Acknowledge sleep notification (required to actually sleep)
static inline void acknowledgeSleep(io_connect_t connection, long notificationID) {
    IOAllowPowerChange(connection, (intptr_t)notificationID);
}
*/
import "C"

import (
	"log"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// darwinMonitor implements sleep/wake monitoring for macOS using IOKit
type darwinMonitor struct {
	connection C.io_connect_t
	notifyPort C.IONotificationPortRef
	notifier   C.io_object_t

	eventCh chan<- Event
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// Global map to track monitor instances (needed for CGo callback)
var (
	monitorsMu sync.RWMutex
	monitors   = make(map[uintptr]*darwinMonitor)
)

// newDarwinMonitor creates a new macOS sleep/wake monitor
func newDarwinMonitor() (platformMonitor, error) {
	return &darwinMonitor{
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}, nil
}

// newPlatformMonitor creates a platform-specific monitor (macOS implementation)
func newPlatformMonitor() (platformMonitor, error) {
	return newDarwinMonitor()
}

// Start begins monitoring for sleep/wake events
func (m *darwinMonitor) Start(eventCh chan<- Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.eventCh = eventCh

	// Register this monitor globally
	refcon := uintptr(unsafe.Pointer(m))
	monitorsMu.Lock()
	monitors[refcon] = m
	monitorsMu.Unlock()

	// Register for sleep/wake notifications
	// Pass uintptr as void* - safe because we're not dereferencing it in C
	result := C.registerSleepWake(
		&m.connection,
		&m.notifyPort,
		&m.notifier,
		C.uintptr_t(refcon), //nolint:gocritic // dupSubExpr: false positive in cgo-generated call wrapper; this source has no == expression
	)

	if result != C.kIOReturnSuccess {
		monitorsMu.Lock()
		delete(monitors, refcon)
		monitorsMu.Unlock()
		return ErrPlatformNotSupported
	}

	m.running = true

	// Start run loop in separate goroutine
	go m.runLoop()

	log.Println("macOS IOKit sleep/wake monitoring started")
	return nil
}

// Stop stops monitoring
func (m *darwinMonitor) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	// Signal stop
	close(m.stopCh)

	// Unregister from global map
	refcon := uintptr(unsafe.Pointer(m))
	monitorsMu.Lock()
	delete(monitors, refcon)
	monitorsMu.Unlock()

	// Clean up IOKit resources
	if m.connection != 0 {
		C.IODeregisterForSystemPower(&m.notifier)
		C.IOServiceClose(m.connection)
		m.connection = 0
	}

	if m.notifyPort != nil {
		C.IONotificationPortDestroy(m.notifyPort)
		m.notifyPort = nil
	}

	// Wait for run loop to finish
	<-m.doneCh

	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	log.Println("macOS IOKit sleep/wake monitoring stopped")
	return nil
}

// Platform returns the platform name
func (m *darwinMonitor) Platform() string {
	return "darwin"
}

// runLoop runs the CoreFoundation run loop for event processing
func (m *darwinMonitor) runLoop() {
	defer close(m.doneCh)

	// Lock OS thread for CoreFoundation run loop
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Run the CF run loop
	// We use a timer to check stopCh periodically
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-timer.C:
			// Let the run loop process events
			C.CFRunLoopRunInMode(C.kCFRunLoopDefaultMode, 0.1, C.true)
		}
	}
}

// handleSleepWakeMessage handles IOKit sleep/wake messages
func (m *darwinMonitor) handleSleepWakeMessage(messageType int, notificationID int64) {
	// IOKit power management message types
	const (
		kIOMessageCanSystemSleep     = 0xE0000270
		kIOMessageSystemWillSleep    = 0xE0000280
		kIOMessageSystemHasPoweredOn = 0xE0000300
	)

	switch uint32(messageType) {
	case kIOMessageCanSystemSleep:
		// System can sleep - acknowledge immediately
		log.Println("IOKit: System can sleep (acknowledging)")
		C.acknowledgeSleep(m.connection, C.long(notificationID))

	case kIOMessageSystemWillSleep:
		// System is about to sleep
		log.Println("IOKit: System will sleep")

		// Send sleep event
		event := Event{
			Type:      EventSleep,
			Timestamp: time.Now(),
			Source:    "iokit",
		}

		select {
		case m.eventCh <- event:
		default:
			log.Println("Warning: Event channel full, dropping sleep event")
		}

		// Must acknowledge to allow sleep
		C.acknowledgeSleep(m.connection, C.long(notificationID))

	case kIOMessageSystemHasPoweredOn:
		// System has woken up
		log.Println("IOKit: System has powered on")

		// Send wake event
		event := Event{
			Type:      EventWake,
			Timestamp: time.Now(),
			Source:    "iokit",
		}

		select {
		case m.eventCh <- event:
		default:
			log.Println("Warning: Event channel full, dropping wake event")
		}
	}
}

//export goSleepWakeCallback
func goSleepWakeCallback(refcon C.uintptr_t, messageType C.int, notificationID C.long) {
	// Lookup monitor from global map
	monitorsMu.RLock()
	monitor, ok := monitors[uintptr(refcon)]
	monitorsMu.RUnlock()

	if !ok {
		log.Println("Warning: Received callback for unknown monitor")
		return
	}

	monitor.handleSleepWakeMessage(int(messageType), int64(notificationID))
}
