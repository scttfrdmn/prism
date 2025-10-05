// Package progress provides comprehensive tests for the progress reporting system
package progress

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

const testOperation = "test-operation"

func TestNewProgressReporter(t *testing.T) {
	stages := []Stage{
		{Name: "stage1", Weight: 0.3},
		{Name: "stage2", Weight: 0.7},
	}

	reporter := NewProgressReporter(testOperation, stages)

	if reporter.operation != testOperation {
		t.Errorf("Expected operation 'test-operation', got '%s'", reporter.operation)
	}

	if len(reporter.stages) != 2 {
		t.Errorf("Expected 2 stages, got %d", len(reporter.stages))
	}

	// Check that weights are normalized (should sum to 1.0)
	totalWeight := 0.0
	for _, stage := range reporter.stages {
		totalWeight += stage.Weight
	}

	if totalWeight < 0.99 || totalWeight > 1.01 { // Allow for floating point precision
		t.Errorf("Expected total weight ~1.0, got %f", totalWeight)
	}
}

func TestNewProgressReporterEqualWeights(t *testing.T) {
	stages := []Stage{
		{Name: "stage1"}, // No weight specified
		{Name: "stage2"}, // No weight specified
		{Name: "stage3"}, // No weight specified
	}

	reporter := NewProgressReporter(testOperation, stages)

	// Should assign equal weights (1/3 each)
	expectedWeight := 1.0 / 3.0
	for i, stage := range reporter.stages {
		if stage.Weight < expectedWeight-0.01 || stage.Weight > expectedWeight+0.01 {
			t.Errorf("Stage %d expected weight ~%f, got %f", i, expectedWeight, stage.Weight)
		}
	}
}

func TestNewLaunchProgressReporter(t *testing.T) {
	instanceName := "test-instance"
	reporter := NewLaunchProgressReporter(instanceName)

	if !contains(reporter.operation, instanceName) {
		t.Errorf("Expected operation to contain instance name '%s', got '%s'", instanceName, reporter.operation)
	}

	// Should have the predefined launch stages
	expectedStages := []string{
		"template_resolution",
		"ami_discovery",
		"instance_launch",
		"software_installation",
		"service_startup",
		"finalization",
	}

	if len(reporter.stages) != len(expectedStages) {
		t.Errorf("Expected %d stages, got %d", len(expectedStages), len(reporter.stages))
	}

	for i, expectedName := range expectedStages {
		if i >= len(reporter.stages) {
			t.Errorf("Missing stage: %s", expectedName)
			continue
		}
		if reporter.stages[i].Name != expectedName {
			t.Errorf("Expected stage %d to be '%s', got '%s'", i, expectedName, reporter.stages[i].Name)
		}
	}

	// Check metadata
	if reporter.metadata["instance_name"] != instanceName {
		t.Errorf("Expected metadata instance_name '%s', got '%v'", instanceName, reporter.metadata["instance_name"])
	}
}

func TestAddCallback(t *testing.T) {
	reporter := createTestReporter()
	callCount := 0

	callback := func(update ProgressUpdate) {
		callCount++
	}

	reporter.AddCallback(callback)

	// Trigger a callback by starting a stage
	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	// Wait for callback to be called (it's called in a goroutine)
	time.Sleep(10 * time.Millisecond)

	if callCount == 0 {
		t.Error("Callback was not called")
	}
}

func TestSetMetadata(t *testing.T) {
	reporter := createTestReporter()

	reporter.SetMetadata("test-key", "test-value")
	reporter.SetMetadata("test-number", 42)

	if reporter.metadata["test-key"] != "test-value" {
		t.Errorf("Expected metadata 'test-key' to be 'test-value', got '%v'", reporter.metadata["test-key"])
	}

	if reporter.metadata["test-number"] != 42 {
		t.Errorf("Expected metadata 'test-number' to be 42, got '%v'", reporter.metadata["test-number"])
	}
}

func TestStartStage(t *testing.T) {
	reporter := createTestReporter()

	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	stage := reporter.getStageByName("stage1")
	if stage == nil {
		t.Fatal("Stage not found")
	}

	if stage.Status != StageInProgress {
		t.Errorf("Expected stage status to be %s, got %s", StageInProgress, stage.Status)
	}

	if stage.Progress != 0.0 {
		t.Errorf("Expected stage progress to be 0.0, got %f", stage.Progress)
	}

	if stage.StartTime == nil {
		t.Error("Expected StartTime to be set")
	}

	if reporter.current != 0 {
		t.Errorf("Expected current stage index to be 0, got %d", reporter.current)
	}
}

func TestStartStageNotFound(t *testing.T) {
	reporter := createTestReporter()

	err := reporter.StartStage("nonexistent-stage")
	if err == nil {
		t.Error("Expected error when starting nonexistent stage")
	}

	if !contains(err.Error(), "stage not found") {
		t.Errorf("Expected 'stage not found' error, got: %v", err)
	}
}

func TestReportStageProgress(t *testing.T) {
	reporter := createTestReporter()

	// Start stage first
	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	// Report progress
	err = reporter.ReportStageProgress("stage1", 0.5, "Half done")
	if err != nil {
		t.Fatalf("ReportStageProgress failed: %v", err)
	}

	stage := reporter.getStageByName("stage1")
	if stage.Progress != 0.5 {
		t.Errorf("Expected stage progress to be 0.5, got %f", stage.Progress)
	}

	if stage.Description != "Half done" {
		t.Errorf("Expected description 'Half done', got '%s'", stage.Description)
	}
}

func TestReportStageProgressClamping(t *testing.T) {
	reporter := createTestReporter()
	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	// Test progress < 0
	err = reporter.ReportStageProgress("stage1", -0.5)
	if err != nil {
		t.Fatalf("ReportStageProgress failed: %v", err)
	}

	stage := reporter.getStageByName("stage1")
	if stage.Progress != 0.0 {
		t.Errorf("Expected clamped progress to be 0.0, got %f", stage.Progress)
	}

	// Test progress > 1
	err = reporter.ReportStageProgress("stage1", 1.5)
	if err != nil {
		t.Fatalf("ReportStageProgress failed: %v", err)
	}

	if stage.Progress != 1.0 {
		t.Errorf("Expected clamped progress to be 1.0, got %f", stage.Progress)
	}
}

func TestCompleteStage(t *testing.T) {
	reporter := createTestReporter()

	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	err = reporter.CompleteStage("stage1")
	if err != nil {
		t.Fatalf("CompleteStage failed: %v", err)
	}

	stage := reporter.getStageByName("stage1")
	if stage.Status != StageCompleted {
		t.Errorf("Expected stage status to be %s, got %s", StageCompleted, stage.Status)
	}

	if stage.Progress != 1.0 {
		t.Errorf("Expected stage progress to be 1.0, got %f", stage.Progress)
	}

	if stage.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestFailStage(t *testing.T) {
	reporter := createTestReporter()

	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	testError := errors.New("test error")
	err = reporter.FailStage("stage1", testError)
	if err != nil {
		t.Fatalf("FailStage failed: %v", err)
	}

	stage := reporter.getStageByName("stage1")
	if stage.Status != StageFailed {
		t.Errorf("Expected stage status to be %s, got %s", StageFailed, stage.Status)
	}

	if stage.Error != testError.Error() {
		t.Errorf("Expected error message '%s', got '%s'", testError.Error(), stage.Error)
	}

	if stage.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestGetProgress(t *testing.T) {
	reporter := createTestReporter()

	progress := reporter.GetProgress()

	if progress.Operation != testOperation {
		t.Errorf("Expected operation %s, got '%s'", testOperation, progress.Operation)
	}

	if progress.OverallProgress != 0.0 {
		t.Errorf("Expected overall progress to be 0.0, got %f", progress.OverallProgress)
	}

	if len(progress.AllStages) != 2 {
		t.Errorf("Expected 2 stages in progress, got %d", len(progress.AllStages))
	}
}

func TestCalculateOverallProgress(t *testing.T) {
	reporter := createTestReporter()

	// Complete first stage (50% weight)
	_ = reporter.StartStage("stage1")
	_ = reporter.CompleteStage("stage1")

	// Start second stage and report 50% progress
	_ = reporter.StartStage("stage2")
	_ = reporter.ReportStageProgress("stage2", 0.5)

	progress := reporter.calculateOverallProgress()

	// Should be 0.5 (stage1 complete) + 0.5 * 0.5 (stage2 half done) = 0.75
	expected := 0.75
	if progress < expected-0.01 || progress > expected+0.01 {
		t.Errorf("Expected overall progress ~%f, got %f", expected, progress)
	}
}

func TestCalculateOverallProgressWithSkippedStage(t *testing.T) {
	stages := []Stage{
		{Name: "stage1", Weight: 0.5, Status: StageCompleted},
		{Name: "stage2", Weight: 0.3, Status: StageSkipped},
		{Name: "stage3", Weight: 0.2, Status: StagePending},
	}

	reporter := NewProgressReporter("test", stages)
	progress := reporter.calculateOverallProgress()

	// Should be 0.5 (completed) + 0.3 (skipped counts as completed) + 0 (pending) = 0.8
	expected := 0.8
	if progress < expected-0.01 || progress > expected+0.01 {
		t.Errorf("Expected overall progress ~%f, got %f", expected, progress)
	}
}

func TestCalculateETAWithNoProgress(t *testing.T) {
	reporter := createTestReporter()

	eta := reporter.calculateETA(0.0)
	if eta != nil {
		t.Error("Expected ETA to be nil with no progress")
	}
}

func TestCalculateETAWithProgress(t *testing.T) {
	reporter := createTestReporter()

	// Simulate some time passing
	reporter.startTime = time.Now().Add(-10 * time.Second)

	eta := reporter.calculateETA(0.5) // 50% complete
	if eta == nil {
		t.Error("Expected ETA to be calculated with progress")
	}

	// With 50% done in 10 seconds, should estimate ~10 more seconds
	if *eta < 5*time.Second || *eta > 15*time.Second {
		t.Errorf("Expected ETA around 10 seconds, got %v", *eta)
	}
}

func TestGetStageByName(t *testing.T) {
	reporter := createTestReporter()

	stage := reporter.getStageByName("stage1")
	if stage == nil {
		t.Error("Expected to find stage1")
	}

	if stage.Name != "stage1" {
		t.Errorf("Expected stage name 'stage1', got '%s'", stage.Name)
	}

	nonExistentStage := reporter.getStageByName("nonexistent")
	if nonExistentStage != nil {
		t.Error("Expected nil for nonexistent stage")
	}
}

func TestCallbackPanicHandling(t *testing.T) {
	reporter := createTestReporter()

	// Add a callback that panics
	panicCallback := func(update ProgressUpdate) {
		panic("test panic")
	}

	reporter.AddCallback(panicCallback)

	// This should not crash the program
	err := reporter.StartStage("stage1")
	if err != nil {
		t.Fatalf("StartStage failed: %v", err)
	}

	// Give time for the callback to be called and recover
	time.Sleep(10 * time.Millisecond)

	// Reporter should still function normally
	err = reporter.ReportStageProgress("stage1", 0.5)
	if err != nil {
		t.Errorf("Progress reporting should work after callback panic: %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	reporter := createTestReporter()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent stage operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			if id == 0 {
				// One goroutine starts the stage
				_ = reporter.StartStage("stage1")
			} else {
				// Others report progress and get status
				_ = reporter.ReportStageProgress("stage1", float64(id)/float64(numGoroutines))
				_ = reporter.GetProgress()
				reporter.SetMetadata("key", id)
			}
		}(i)
	}

	wg.Wait()

	// Verify reporter is in a consistent state
	progress := reporter.GetProgress()
	if progress.Operation != testOperation {
		t.Error("Reporter state corrupted after concurrent access")
	}
}

func TestStageStatusConstants(t *testing.T) {
	// Test that stage status constants are properly defined
	if StagePending != "pending" {
		t.Errorf("Expected StagePending to be 'pending', got '%s'", StagePending)
	}
	if StageInProgress != "in_progress" {
		t.Errorf("Expected StageInProgress to be 'in_progress', got '%s'", StageInProgress)
	}
	if StageCompleted != "completed" {
		t.Errorf("Expected StageCompleted to be 'completed', got '%s'", StageCompleted)
	}
	if StageFailed != "failed" {
		t.Errorf("Expected StageFailed to be 'failed', got '%s'", StageFailed)
	}
	if StageSkipped != "skipped" {
		t.Errorf("Expected StageSkipped to be 'skipped', got '%s'", StageSkipped)
	}
}

// Helper functions for tests

func createTestReporter() *ProgressReporter {
	stages := []Stage{
		{Name: "stage1", Description: "First stage", Weight: 0.5},
		{Name: "stage2", Description: "Second stage", Weight: 0.5},
	}
	return NewProgressReporter(testOperation, stages)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Benchmark tests

func BenchmarkNewProgressReporter(b *testing.B) {
	stages := []Stage{
		{Name: "stage1", Weight: 0.5},
		{Name: "stage2", Weight: 0.5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewProgressReporter("benchmark", stages)
	}
}

func BenchmarkReportProgress(b *testing.B) {
	reporter := createTestReporter()
	_ = reporter.StartStage("stage1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reporter.ReportStageProgress("stage1", 0.5)
	}
}

func BenchmarkGetProgress(b *testing.B) {
	reporter := createTestReporter()
	_ = reporter.StartStage("stage1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reporter.GetProgress()
	}
}

func BenchmarkCalculateOverallProgress(b *testing.B) {
	reporter := createTestReporter()
	_ = reporter.StartStage("stage1")
	_ = reporter.ReportStageProgress("stage1", 0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reporter.calculateOverallProgress()
	}
}