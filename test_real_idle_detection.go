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

func main() {
	fmt.Println("=== Testing Real Idle Detection System ===")
	
	baseURL := "http://localhost:8947"
	
	// First, let's restart the instances for testing
	fmt.Println("1. Starting test instances...")
	
	// Start hibernation test instance
	startURL := fmt.Sprintf("%s/api/v1/instances/idle-test-hibernation/start", baseURL)
	resp, err := http.Post(startURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Printf("Failed to start hibernation instance: %v", err)
	} else {
		resp.Body.Close()
		fmt.Println("   ✓ Started hibernation test instance")
	}
	
	// Start stop test instance
	startURL = fmt.Sprintf("%s/api/v1/instances/idle-test-stop/start", baseURL)
	resp, err = http.Post(startURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Printf("Failed to start stop instance: %v", err)
	} else {
		resp.Body.Close()
		fmt.Println("   ✓ Started stop test instance")
	}
	
	// Wait for instances to start
	fmt.Println("   Waiting 60 seconds for instances to start...")
	time.Sleep(60 * time.Second)
	
	// Create an idle manager to simulate sending metrics
	fmt.Println("2. Creating idle states with simulated metrics...")
	
	idleManager, err := idle.NewManager()
	if err != nil {
		log.Fatal("Failed to create idle manager:", err)
	}
	
	// Get actual instance IDs from the daemon
	listURL := fmt.Sprintf("%s/api/v1/instances", baseURL)
	resp, err = http.Get(listURL)
	if err != nil {
		log.Fatal("Failed to get instance list:", err)
	}
	defer resp.Body.Close()
	
	var instances []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&instances); err != nil {
		log.Fatal("Failed to decode instances:", err)
	}
	
	var hibernationInstanceID, stopInstanceID string
	for _, inst := range instances {
		name, ok := inst["name"].(string)
		if !ok {
			continue
		}
		id, ok := inst["id"].(string)
		if !ok {
			continue
		}
		
		if name == "idle-test-hibernation" {
			hibernationInstanceID = id
		} else if name == "idle-test-stop" {
			stopInstanceID = id
		}
	}
	
	if hibernationInstanceID == "" || stopInstanceID == "" {
		log.Fatal("Could not find instance IDs")
	}
	
	fmt.Printf("   Found hibernation instance: %s\n", hibernationInstanceID)
	fmt.Printf("   Found stop instance: %s\n", stopInstanceID)
	
	// Create idle metrics that will trigger the 1-minute idle policies
	fmt.Println("3. Processing idle metrics...")
	
	currentTime := time.Now()
	
	// Process hibernation instance metrics
	hibernationMetrics := &idle.UsageMetrics{
		Timestamp:   currentTime,
		CPU:         2.0,  // Below 50% threshold
		Memory:      15.0, // Below 50% threshold  
		Network:     10.0, // Below 100 KBps threshold
		Disk:        20.0, // Below 200 KBps threshold
		HasActivity: false, // No user activity
	}
	
	hibernationState, err := idleManager.ProcessMetrics(hibernationInstanceID, "idle-test-hibernation", hibernationMetrics)
	if err != nil {
		log.Printf("Failed to process hibernation metrics: %v", err)
	} else if hibernationState != nil {
		fmt.Printf("   Hibernation state: idle=%t, next_action=%s at %s\n", 
			hibernationState.IsIdle, 
			hibernationState.NextAction.Action,
			hibernationState.NextAction.Time.Format("15:04:05"))
	}
	
	// Process stop instance metrics
	stopMetrics := &idle.UsageMetrics{
		Timestamp:   currentTime,
		CPU:         3.0,  // Below 50% threshold
		Memory:      20.0, // Below 50% threshold
		Network:     15.0, // Below 100 KBps threshold
		Disk:        30.0, // Below 200 KBps threshold
		HasActivity: false, // No user activity
	}
	
	stopState, err := idleManager.ProcessMetrics(stopInstanceID, "idle-test-stop", stopMetrics)
	if err != nil {
		log.Printf("Failed to process stop metrics: %v", err)
	} else if stopState != nil {
		fmt.Printf("   Stop state: idle=%t, next_action=%s at %s\n",
			stopState.IsIdle,
			stopState.NextAction.Action,
			stopState.NextAction.Time.Format("15:04:05"))
	}
	
	// Wait for actions to become ready (1 minute + buffer)
	fmt.Println("4. Waiting for idle actions to become ready (75 seconds)...")
	time.Sleep(75 * time.Second)
	
	// Check for pending actions using the local manager
	fmt.Println("5. Checking for pending actions...")
	pendingActions := idleManager.CheckPendingActions()
	
	if len(pendingActions) == 0 {
		fmt.Println("   No pending actions found in local manager")
		
		// Also check via API
		pendingURL := fmt.Sprintf("%s/api/v1/idle/pending-actions", baseURL)
		resp, err := http.Get(pendingURL)
		if err != nil {
			log.Printf("Failed to check API pending actions: %v", err)
		} else {
			defer resp.Body.Close()
			var apiResult interface{}
			if err := json.NewDecoder(resp.Body).Decode(&apiResult); err == nil {
				fmt.Printf("   API pending actions: %+v\n", apiResult)
			}
		}
	} else {
		fmt.Printf("   Found %d pending actions in local manager:\n", len(pendingActions))
		for _, action := range pendingActions {
			fmt.Printf("   - %s (%s): %s\n", action.InstanceName, action.InstanceID, action.NextAction.Action)
		}
		
		// Try to execute actions by manually calling the hibernation/stop APIs
		fmt.Println("6. Executing actions manually...")
		for _, action := range pendingActions {
			var actionURL string
			if action.NextAction.Action == idle.Hibernate {
				actionURL = fmt.Sprintf("%s/api/v1/instances/%s/hibernate", baseURL, action.InstanceName)
			} else {
				actionURL = fmt.Sprintf("%s/api/v1/instances/%s/stop", baseURL, action.InstanceName)
			}
			
			resp, err := http.Post(actionURL, "application/json", bytes.NewBuffer([]byte("{}")))
			if err != nil {
				log.Printf("   Failed to execute %s on %s: %v", action.NextAction.Action, action.InstanceName, err)
			} else {
				resp.Body.Close()
				fmt.Printf("   ✓ Executed %s on %s\n", action.NextAction.Action, action.InstanceName)
			}
		}
		
		// Add history entries
		for _, action := range pendingActions {
			historyEntry := idle.HistoryEntry{
				InstanceID:   action.InstanceID,
				InstanceName: action.InstanceName,
				Action:       action.NextAction.Action,
				Time:         time.Now(),
				IdleDuration: time.Since(*action.IdleSince),
				Metrics:      action.LastMetrics,
			}
			
			if err := idleManager.AddHistoryEntry(historyEntry); err != nil {
				log.Printf("Failed to add history entry: %v", err)
			}
		}
	}
	
	fmt.Println("7. Final instance status check...")
	time.Sleep(10 * time.Second)
	
	// Check final instance status
	resp, err = http.Get(listURL)
	if err != nil {
		log.Printf("Failed to get final instance list: %v", err)
	} else {
		defer resp.Body.Close()
		
		var finalInstances []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&finalInstances); err == nil {
			for _, inst := range finalInstances {
				name, _ := inst["name"].(string)
				state, _ := inst["state"].(string)
				if name == "idle-test-hibernation" || name == "idle-test-stop" {
					fmt.Printf("   %s: %s\n", name, state)
				}
			}
		}
	}
	
	fmt.Println("\n=== Real Idle Detection Test Complete ===")
}