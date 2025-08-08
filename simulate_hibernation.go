package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
)

// TestMetrics represents test usage metrics
type TestMetrics struct {
	InstanceID   string               `json:"instance_id"`
	InstanceName string               `json:"instance_name"`
	Metrics      idle.UsageMetrics    `json:"metrics"`
}

func main() {
	fmt.Println("=== Testing Hibernation/Stop Behavior ===")
	
	// Create test metrics that will trigger idle detection
	hibernationMetrics := TestMetrics{
		InstanceID:   "i-hibernation-test", 
		InstanceName: "idle-test-hibernation",
		Metrics: idle.UsageMetrics{
			Timestamp:   time.Now(),
			CPU:         2.0,  // Below 50% threshold
			Memory:      15.0, // Below 50% threshold
			Network:     10.0, // Below 100 KBps threshold
			Disk:        20.0, // Below 200 KBps threshold
			GPU:         nil,  // No GPU activity
			HasActivity: false, // No user activity
		},
	}

	stopMetrics := TestMetrics{
		InstanceID:   "i-stop-test",
		InstanceName: "idle-test-stop", 
		Metrics: idle.UsageMetrics{
			Timestamp:   time.Now(),
			CPU:         3.0,  // Below 50% threshold
			Memory:      20.0, // Below 50% threshold
			Network:     15.0, // Below 100 KBps threshold
			Disk:        30.0, // Below 200 KBps threshold
			HasActivity: false, // No user activity
		},
	}

	fmt.Println("1. Creating idle manager and processing metrics...")

	// Initialize idle manager directly
	idleManager, err := idle.NewManager()
	if err != nil {
		log.Fatal("Failed to initialize idle manager:", err)
	}

	// Process metrics for hibernation instance
	fmt.Println("   Processing hibernation instance metrics...")
	hibernationState, err := idleManager.ProcessMetrics(
		hibernationMetrics.InstanceID,
		hibernationMetrics.InstanceName, 
		&hibernationMetrics.Metrics,
	)
	if err != nil {
		log.Fatal("Failed to process hibernation metrics:", err)
	}

	if hibernationState != nil {
		fmt.Printf("   Hibernation state: idle=%t, profile=%s\n", hibernationState.IsIdle, hibernationState.Profile)
		if hibernationState.NextAction != nil {
			fmt.Printf("   Next action: %s at %s\n", hibernationState.NextAction.Action, hibernationState.NextAction.Time.Format(time.RFC3339))
		}
	}

	// Process metrics for stop instance  
	fmt.Println("   Processing stop instance metrics...")
	stopState, err := idleManager.ProcessMetrics(
		stopMetrics.InstanceID,
		stopMetrics.InstanceName,
		&stopMetrics.Metrics,
	)
	if err != nil {
		log.Fatal("Failed to process stop metrics:", err)
	}

	if stopState != nil {
		fmt.Printf("   Stop state: idle=%t, profile=%s\n", stopState.IsIdle, stopState.Profile)
		if stopState.NextAction != nil {
			fmt.Printf("   Next action: %s at %s\n", stopState.NextAction.Action, stopState.NextAction.Time.Format(time.RFC3339))
		}
	}

	fmt.Println("\n2. Waiting for actions to become pending...")
	
	// Wait for actions to become ready (test profiles have 1 minute idle time)
	time.Sleep(65 * time.Second)

	// Check pending actions
	fmt.Println("3. Checking for pending actions...")
	pendingActions := idleManager.CheckPendingActions()
	
	if len(pendingActions) == 0 {
		fmt.Println("   No pending actions found")
	} else {
		fmt.Printf("   Found %d pending actions:\n", len(pendingActions))
		for _, action := range pendingActions {
			fmt.Printf("   - %s (%s): %s\n", action.InstanceName, action.InstanceID, action.NextAction.Action)
		}

		// Execute pending actions via API call
		fmt.Println("\n4. Executing pending actions via API...")
		executeURL := "http://localhost:8947/api/v1/idle/execute-actions"
		
		resp, err := http.Post(executeURL, "application/json", bytes.NewBuffer([]byte("{}")))
		if err != nil {
			log.Printf("Failed to execute actions: %v", err)
		} else {
			defer resp.Body.Close()
			
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				fmt.Printf("   Execution result: %+v\n", result)
			}
		}
	}

	fmt.Println("\n5. Checking action history...")
	historyURL := "http://localhost:8947/api/v1/idle/history"
	resp, err := http.Get(historyURL)
	if err != nil {
		log.Printf("Failed to get history: %v", err)
	} else {
		defer resp.Body.Close()
		
		var history []idle.HistoryEntry
		if err := json.NewDecoder(resp.Body).Decode(&history); err == nil {
			if len(history) == 0 {
				fmt.Println("   No history entries")
			} else {
				fmt.Printf("   Found %d history entries:\n", len(history))
				for _, entry := range history {
					fmt.Printf("   - %s: %s %s (idle for %s)\n", 
						entry.Time.Format("15:04:05"),
						entry.InstanceName,
						entry.Action,
						entry.IdleDuration,
					)
				}
			}
		}
	}

	fmt.Println("\n=== Test Complete ===")
}