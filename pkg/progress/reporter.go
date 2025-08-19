package progress

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProgressReporter manages and reports progress for long-running operations
type ProgressReporter struct {
	mu         sync.RWMutex
	stages     []Stage
	current    int
	startTime  time.Time
	callbacks  []ProgressCallback
	operation  string
	metadata   map[string]interface{}
}

// Stage represents a step in a multi-stage operation
type Stage struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Weight      float64      `json:"weight"`  // Relative weight for ETA calculation (0.0-1.0)
	Status      StageStatus  `json:"status"`
	Progress    float64      `json:"progress"` // Progress within this stage (0.0-1.0)
	StartTime   *time.Time   `json:"start_time,omitempty"`
	EndTime     *time.Time   `json:"end_time,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// StageStatus represents the status of a stage
type StageStatus string

const (
	StagePending    StageStatus = "pending"
	StageInProgress StageStatus = "in_progress"
	StageCompleted  StageStatus = "completed"
	StageFailed     StageStatus = "failed"
	StageSkipped    StageStatus = "skipped"
)

// ProgressUpdate contains information about current progress
type ProgressUpdate struct {
	Operation        string                 `json:"operation"`
	CurrentStage     string                 `json:"current_stage"`
	StageProgress    float64                `json:"stage_progress"`
	OverallProgress  float64                `json:"overall_progress"`
	ETA              *time.Duration         `json:"eta,omitempty"`
	ElapsedTime      time.Duration          `json:"elapsed_time"`
	Description      string                 `json:"description"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	AllStages        []Stage                `json:"all_stages"`
}

// ProgressCallback is called when progress is updated
type ProgressCallback func(update ProgressUpdate)

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(operation string, stages []Stage) *ProgressReporter {
	// Normalize weights if they don't sum to 1.0
	totalWeight := 0.0
	for _, stage := range stages {
		totalWeight += stage.Weight
	}
	
	if totalWeight > 0 {
		for i := range stages {
			stages[i].Weight = stages[i].Weight / totalWeight
		}
	} else {
		// Equal weights if not specified
		weight := 1.0 / float64(len(stages))
		for i := range stages {
			stages[i].Weight = weight
		}
	}
	
	return &ProgressReporter{
		stages:    stages,
		operation: operation,
		startTime: time.Now(),
		metadata:  make(map[string]interface{}),
	}
}

// NewLaunchProgressReporter creates a progress reporter specifically for instance launches
func NewLaunchProgressReporter(instanceName string) *ProgressReporter {
	stages := []Stage{
		{
			Name:        "template_resolution",
			Description: "Resolving template and generating configuration",
			Weight:      0.15,
			Status:      StagePending,
		},
		{
			Name:        "ami_discovery",
			Description: "Finding optimal AMI for your region",
			Weight:      0.10,
			Status:      StagePending,
		},
		{
			Name:        "instance_launch",
			Description: "Starting EC2 instance",
			Weight:      0.20,
			Status:      StagePending,
		},
		{
			Name:        "software_installation",
			Description: "Installing software and configuring environment",
			Weight:      0.40,
			Status:      StagePending,
		},
		{
			Name:        "service_startup",
			Description: "Starting services and performing health checks",
			Weight:      0.10,
			Status:      StagePending,
		},
		{
			Name:        "finalization",
			Description: "Finalizing setup and updating state",
			Weight:      0.05,
			Status:      StagePending,
		},
	}
	
	reporter := NewProgressReporter(fmt.Sprintf("launch_%s", instanceName), stages)
	reporter.SetMetadata("instance_name", instanceName)
	return reporter
}

// AddCallback adds a progress callback
func (pr *ProgressReporter) AddCallback(callback ProgressCallback) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.callbacks = append(pr.callbacks, callback)
}

// SetMetadata sets metadata for the operation
func (pr *ProgressReporter) SetMetadata(key string, value interface{}) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.metadata[key] = value
}

// StartStage marks a stage as started
func (pr *ProgressReporter) StartStage(stageName string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	stage := pr.getStageByName(stageName)
	if stage == nil {
		return fmt.Errorf("stage not found: %s", stageName)
	}
	
	now := time.Now()
	stage.Status = StageInProgress
	stage.StartTime = &now
	stage.Progress = 0.0
	
	// Update current stage index
	for i, s := range pr.stages {
		if s.Name == stageName {
			pr.current = i
			break
		}
	}
	
	pr.notifyCallbacks()
	return nil
}

// ReportStageProgress updates progress for the current stage
func (pr *ProgressReporter) ReportStageProgress(stageName string, progress float64, description ...string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	stage := pr.getStageByName(stageName)
	if stage == nil {
		return fmt.Errorf("stage not found: %s", stageName)
	}
	
	// Clamp progress to [0.0, 1.0]
	if progress < 0.0 {
		progress = 0.0
	} else if progress > 1.0 {
		progress = 1.0
	}
	
	stage.Progress = progress
	
	// Update description if provided
	if len(description) > 0 && description[0] != "" {
		stage.Description = description[0]
	}
	
	pr.notifyCallbacks()
	return nil
}

// CompleteStage marks a stage as completed
func (pr *ProgressReporter) CompleteStage(stageName string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	stage := pr.getStageByName(stageName)
	if stage == nil {
		return fmt.Errorf("stage not found: %s", stageName)
	}
	
	now := time.Now()
	stage.Status = StageCompleted
	stage.Progress = 1.0
	stage.EndTime = &now
	
	pr.notifyCallbacks()
	return nil
}

// FailStage marks a stage as failed
func (pr *ProgressReporter) FailStage(stageName string, err error) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	
	stage := pr.getStageByName(stageName)
	if stage == nil {
		return fmt.Errorf("stage not found: %s", stageName)
	}
	
	now := time.Now()
	stage.Status = StageFailed
	stage.EndTime = &now
	stage.Error = err.Error()
	
	pr.notifyCallbacks()
	return nil
}

// GetProgress returns the current progress state
func (pr *ProgressReporter) GetProgress() ProgressUpdate {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	
	overallProgress := pr.calculateOverallProgress()
	eta := pr.calculateETA(overallProgress)
	
	currentStageName := ""
	stageProgress := 0.0
	description := ""
	
	if pr.current < len(pr.stages) {
		currentStage := &pr.stages[pr.current]
		currentStageName = currentStage.Name
		stageProgress = currentStage.Progress
		description = currentStage.Description
	}
	
	return ProgressUpdate{
		Operation:       pr.operation,
		CurrentStage:    currentStageName,
		StageProgress:   stageProgress,
		OverallProgress: overallProgress,
		ETA:             eta,
		ElapsedTime:     time.Since(pr.startTime),
		Description:     description,
		Metadata:        pr.metadata,
		AllStages:       pr.stages,
	}
}

// calculateOverallProgress computes the overall progress across all stages
func (pr *ProgressReporter) calculateOverallProgress() float64 {
	totalProgress := 0.0
	
	for _, stage := range pr.stages {
		switch stage.Status {
		case StageCompleted:
			totalProgress += stage.Weight
		case StageInProgress:
			totalProgress += stage.Weight * stage.Progress
		case StageFailed:
			// Failed stages don't contribute to progress
		case StageSkipped:
			totalProgress += stage.Weight // Skipped stages count as completed
		}
	}
	
	return totalProgress
}

// calculateETA estimates time to completion based on current progress
func (pr *ProgressReporter) calculateETA(overallProgress float64) *time.Duration {
	if overallProgress <= 0.0 {
		return nil
	}
	
	elapsed := time.Since(pr.startTime)
	if elapsed < time.Second {
		return nil // Not enough data for accurate ETA
	}
	
	// Simple linear projection
	totalEstimated := time.Duration(float64(elapsed) / overallProgress)
	remaining := totalEstimated - elapsed
	
	if remaining < 0 {
		remaining = 0
	}
	
	return &remaining
}

// getStageByName finds a stage by name
func (pr *ProgressReporter) getStageByName(name string) *Stage {
	for i := range pr.stages {
		if pr.stages[i].Name == name {
			return &pr.stages[i]
		}
	}
	return nil
}

// notifyCallbacks notifies all registered callbacks
func (pr *ProgressReporter) notifyCallbacks() {
	update := pr.calculateProgressUpdate()
	for _, callback := range pr.callbacks {
		go func(cb ProgressCallback, u ProgressUpdate) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but don't crash the reporter
					fmt.Printf("Progress callback panic: %v\n", r)
				}
			}()
			cb(u)
		}(callback, update)
	}
}

// calculateProgressUpdate creates a progress update (internal helper)
func (pr *ProgressReporter) calculateProgressUpdate() ProgressUpdate {
	overallProgress := pr.calculateOverallProgress()
	eta := pr.calculateETA(overallProgress)
	
	currentStageName := ""
	stageProgress := 0.0
	description := ""
	
	if pr.current < len(pr.stages) {
		currentStage := &pr.stages[pr.current]
		currentStageName = currentStage.Name
		stageProgress = currentStage.Progress
		description = currentStage.Description
	}
	
	// Copy stages to avoid race conditions
	stagesCopy := make([]Stage, len(pr.stages))
	copy(stagesCopy, pr.stages)
	
	return ProgressUpdate{
		Operation:       pr.operation,
		CurrentStage:    currentStageName,
		StageProgress:   stageProgress,
		OverallProgress: overallProgress,
		ETA:             eta,
		ElapsedTime:     time.Since(pr.startTime),
		Description:     description,
		Metadata:        pr.metadata,
		AllStages:       stagesCopy,
	}
}