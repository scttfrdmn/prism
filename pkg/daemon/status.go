package daemon

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// StatusTracker tracks daemon operational metrics
type StatusTracker struct {
	// startTime is when the daemon was started
	startTime time.Time

	// activeOperations is the number of currently active operations
	activeOperations int32

	// totalRequests is the total number of requests processed
	totalRequests int64

	// requestTimes stores timestamps of recent requests for rate calculation
	requestTimes []time.Time

	// requestTimesLock protects requestTimes
	requestTimesLock sync.Mutex

	// requestTimeWindow is the time window for rate calculation (default: 1 minute)
	requestTimeWindow time.Duration

	// operationCounter is used to generate unique operation IDs
	operationCounter int64

	// activeOperationTypes tracks types of active operations (for debugging)
	activeOperationTypes map[string]int32

	// operationTypesLock protects activeOperationTypes
	operationTypesLock sync.Mutex

	// instanceActivity tracks recent activity per instance for smart idle detection
	instanceActivity map[string]time.Time

	// instanceActivityLock protects instanceActivity
	instanceActivityLock sync.Mutex
}

// NewStatusTracker creates a new status tracker
func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		startTime:            time.Now(),
		requestTimes:         make([]time.Time, 0, 1000), // Pre-allocate for efficiency
		requestTimeWindow:    1 * time.Minute,            // 1 minute sliding window
		activeOperationTypes: make(map[string]int32),
		instanceActivity:     make(map[string]time.Time),
	}
}

// StartOperation increments the active operations counter
// Returns an operation ID that can be used for tracking
func (s *StatusTracker) StartOperation() int64 {
	// Generate a unique operation ID
	opID := atomic.AddInt64(&s.operationCounter, 1)

	// Increment active operation counter
	atomic.AddInt32(&s.activeOperations, 1)

	return opID
}

// StartOperationWithType increments the active operations counter for a specific type
// Returns an operation ID that can be used for tracking
func (s *StatusTracker) StartOperationWithType(opType string) int64 {
	// Generate a unique operation ID
	opID := atomic.AddInt64(&s.operationCounter, 1)

	// Increment active operation counter
	atomic.AddInt32(&s.activeOperations, 1)

	// Track operation type
	if opType != "" {
		s.operationTypesLock.Lock()
		s.activeOperationTypes[opType]++
		s.operationTypesLock.Unlock()
	}

	return opID
}

// EndOperation decrements the active operations counter
func (s *StatusTracker) EndOperation() {
	atomic.AddInt32(&s.activeOperations, -1)
}

// EndOperationWithType decrements the active operations counter for a specific type
func (s *StatusTracker) EndOperationWithType(opType string) {
	// Decrement active operation counter
	atomic.AddInt32(&s.activeOperations, -1)

	// Untrack operation type
	if opType != "" {
		s.operationTypesLock.Lock()
		if s.activeOperationTypes[opType] > 0 {
			s.activeOperationTypes[opType]--
		}
		s.operationTypesLock.Unlock()
	}
}

// RecordRequest records a request and updates metrics
func (s *StatusTracker) RecordRequest() {
	// Increment request counter
	atomic.AddInt64(&s.totalRequests, 1)

	// Add timestamp to the sliding window
	now := time.Now()

	s.requestTimesLock.Lock()
	defer s.requestTimesLock.Unlock()

	// Add current timestamp
	s.requestTimes = append(s.requestTimes, now)

	// Remove timestamps older than the window
	cutoff := now.Add(-s.requestTimeWindow)
	newStart := 0

	// Find first timestamp that's within the window
	for i, t := range s.requestTimes {
		if t.After(cutoff) {
			newStart = i
			break
		}
	}

	// If all timestamps are within window, don't bother slicing
	if newStart > 0 {
		s.requestTimes = s.requestTimes[newStart:]
	}
}

// GetRequestRate calculates the current request rate per minute
func (s *StatusTracker) GetRequestRate() float64 {
	s.requestTimesLock.Lock()
	defer s.requestTimesLock.Unlock()

	// If no requests or just one, return 0 or low rate
	if len(s.requestTimes) <= 1 {
		return 0
	}

	// Calculate time span of the window
	oldest := s.requestTimes[0]
	newest := s.requestTimes[len(s.requestTimes)-1]
	duration := newest.Sub(oldest)

	// Avoid division by zero
	if duration == 0 {
		return 0
	}

	// Calculate rate per minute
	count := float64(len(s.requestTimes))
	rate := count / duration.Minutes()

	return rate
}

// GetStatus returns the current daemon status
func (s *StatusTracker) GetStatus(version string, region string, awsProfile string) types.DaemonStatus {
	// Get active operations count
	activeOps := int(atomic.LoadInt32(&s.activeOperations))

	// Get operation type breakdown for detailed monitoring
	var operationDetails map[string]int
	s.operationTypesLock.Lock()
	if len(s.activeOperationTypes) > 0 {
		operationDetails = make(map[string]int)
		for opType, count := range s.activeOperationTypes {
			if count > 0 {
				operationDetails[opType] = int(count)
			}
		}
	}
	s.operationTypesLock.Unlock()

	return types.DaemonStatus{
		Version:           version,
		Status:            "running",
		StartTime:         s.startTime,
		Uptime:            time.Since(s.startTime).Round(time.Second).String(),
		ActiveOps:         activeOps,
		TotalRequests:     atomic.LoadInt64(&s.totalRequests),
		RequestsPerMinute: s.GetRequestRate(),
		AWSRegion:         region,
		AWSProfile:        awsProfile,
	}
}

// GetActiveOperationCount returns the current number of active operations
func (s *StatusTracker) GetActiveOperationCount() int {
	return int(atomic.LoadInt32(&s.activeOperations))
}

// RecordInstanceActivity records that an instance had recent activity
func (s *StatusTracker) RecordInstanceActivity(instanceName string) {
	s.instanceActivityLock.Lock()
	defer s.instanceActivityLock.Unlock()
	s.instanceActivity[instanceName] = time.Now()
}

// GetRecentlyActiveInstances returns instances that had activity within the specified duration
func (s *StatusTracker) GetRecentlyActiveInstances(within time.Duration) []string {
	s.instanceActivityLock.Lock()
	defer s.instanceActivityLock.Unlock()

	cutoff := time.Now().Add(-within)
	var activeInstances []string

	for instanceName, lastActivity := range s.instanceActivity {
		if lastActivity.After(cutoff) {
			activeInstances = append(activeInstances, instanceName)
		}
	}

	// Clean up old entries while we're here
	for instanceName, lastActivity := range s.instanceActivity {
		if lastActivity.Before(cutoff.Add(-24 * time.Hour)) { // Keep 24 hours of history
			delete(s.instanceActivity, instanceName)
		}
	}

	return activeInstances
}
