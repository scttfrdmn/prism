// Package idle provides advanced idle detection and policy management
package idle

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// AWSInstanceManager defines the interface for AWS instance operations needed by the scheduler
type AWSInstanceManager interface {
	HibernateInstance(name string) error
	ResumeInstance(name string) error
	StopInstance(name string) error
	StartInstance(name string) error
	GetInstanceNames() ([]string, error)
	GetInstanceID(name string) (string, error) // Get AWS instance ID from instance name
}

// ScheduleType defines the type of hibernation schedule
type ScheduleType string

const (
	ScheduleTypeDaily     ScheduleType = "daily"
	ScheduleTypeWeekly    ScheduleType = "weekly"
	ScheduleTypeWorkHours ScheduleType = "work_hours"
	ScheduleTypeCustom    ScheduleType = "custom"
	ScheduleTypeIdle      ScheduleType = "idle"
)

// DayOfWeek represents a day of the week
type DayOfWeek string

const (
	Monday    DayOfWeek = "monday"
	Tuesday   DayOfWeek = "tuesday"
	Wednesday DayOfWeek = "wednesday"
	Thursday  DayOfWeek = "thursday"
	Friday    DayOfWeek = "friday"
	Saturday  DayOfWeek = "saturday"
	Sunday    DayOfWeek = "sunday"
)

// Schedule represents a hibernation schedule
type Schedule struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        ScheduleType `json:"type"`
	Enabled     bool         `json:"enabled"`

	// Target instances
	TargetInstances []string `json:"target_instances,omitempty"` // Specific instance names, empty means all

	// Time-based scheduling
	StartTime  string      `json:"start_time,omitempty"` // HH:MM format
	EndTime    string      `json:"end_time,omitempty"`   // HH:MM format
	DaysOfWeek []DayOfWeek `json:"days_of_week,omitempty"`
	Timezone   string      `json:"timezone,omitempty"`

	// Idle-based scheduling
	IdleMinutes      int     `json:"idle_minutes,omitempty"`
	CPUThreshold     float64 `json:"cpu_threshold,omitempty"`
	MemoryThreshold  float64 `json:"memory_threshold,omitempty"`
	NetworkThreshold float64 `json:"network_threshold,omitempty"`

	// Actions
	HibernateAction string `json:"hibernate_action"` // hibernate, stop, terminate
	WakeAction      string `json:"wake_action"`      // resume, start, none

	// Advanced options
	GracePeriodMinutes int      `json:"grace_period_minutes"`
	IgnoreTags         []string `json:"ignore_tags"`
	RequireTags        []string `json:"require_tags"`

	// Cost tracking
	EstimatedMonthlySavings float64   `json:"estimated_monthly_savings"`
	LastExecuted            time.Time `json:"last_executed"`
	TotalSavings            float64   `json:"total_savings"`
}

// Scheduler manages hibernation schedules
type Scheduler struct {
	mu                sync.RWMutex
	schedules         map[string]*Schedule
	active            map[string]*ScheduleExecution
	instanceSchedules map[string][]string // instance name -> schedule IDs
	ticker            *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
	awsManager        AWSInstanceManager
	metricsCollector  *MetricsCollector
}

// ScheduleExecution tracks active schedule execution
type ScheduleExecution struct {
	ScheduleID string
	StartTime  time.Time
	NextRun    time.Time
	IsActive   bool
}

// NewScheduler creates a new hibernation scheduler
func NewScheduler(awsManager AWSInstanceManager, metricsCollector *MetricsCollector) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		schedules:         make(map[string]*Schedule),
		active:            make(map[string]*ScheduleExecution),
		instanceSchedules: make(map[string][]string),
		ctx:               ctx,
		cancel:            cancel,
		awsManager:        awsManager,
		metricsCollector:  metricsCollector,
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	s.ticker = time.NewTicker(1 * time.Minute)
	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.cancel()
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			s.checkSchedules()
		}
	}
}

// checkSchedules evaluates all schedules
func (s *Scheduler) checkSchedules() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}

		if s.shouldExecute(schedule, now) {
			go s.executeSchedule(schedule)
		}
	}
}

// shouldExecute determines if a schedule should run
func (s *Scheduler) shouldExecute(schedule *Schedule, now time.Time) bool {
	switch schedule.Type {
	case ScheduleTypeDaily:
		return s.shouldExecuteDaily(schedule, now)
	case ScheduleTypeWeekly:
		return s.shouldExecuteWeekly(schedule, now)
	case ScheduleTypeWorkHours:
		return s.shouldExecuteWorkHours(schedule, now)
	case ScheduleTypeIdle:
		return s.shouldExecuteIdle(schedule)
	case ScheduleTypeCustom:
		return s.shouldExecuteCustom(schedule, now)
	default:
		return false
	}
}

// shouldExecuteDaily checks daily schedule
func (s *Scheduler) shouldExecuteDaily(schedule *Schedule, now time.Time) bool {
	// Parse start and end times
	currentTime := now.Format("15:04")

	// Check if we're in the hibernation window
	if schedule.StartTime <= currentTime && currentTime < schedule.EndTime {
		// Check if already executing
		if exec, exists := s.active[schedule.ID]; exists && exec.IsActive {
			return false
		}
		return true
	}

	return false
}

// shouldExecuteWeekly checks weekly schedule
func (s *Scheduler) shouldExecuteWeekly(schedule *Schedule, now time.Time) bool {
	// Get current day of week
	currentDay := strings.ToLower(now.Weekday().String())

	// Check if today is in the schedule
	for _, day := range schedule.DaysOfWeek {
		if string(day) == currentDay {
			return s.shouldExecuteDaily(schedule, now)
		}
	}

	return false
}

// shouldExecuteWorkHours checks work hours schedule
func (s *Scheduler) shouldExecuteWorkHours(schedule *Schedule, now time.Time) bool {
	// Work hours: Monday-Friday, 9 AM - 6 PM
	weekday := now.Weekday()
	hour := now.Hour()

	// Outside work hours or weekend
	if weekday == time.Saturday || weekday == time.Sunday {
		return true // Hibernate on weekends
	}

	if hour < 9 || hour >= 18 {
		return true // Hibernate outside work hours
	}

	return false
}

// shouldExecuteIdle checks idle-based schedule
// This checks if instances have been idle for the specified duration
func (s *Scheduler) shouldExecuteIdle(schedule *Schedule) bool {
	// If IdleMinutes is not set, can't evaluate
	if schedule.IdleMinutes <= 0 {
		return false
	}

	// If no metrics collector available, can't check idle status
	if s.metricsCollector == nil {
		log.Printf("Warning: No metrics collector available for idle detection on schedule %s", schedule.Name)
		return false
	}

	// Get target instances
	instances := schedule.TargetInstances
	if len(instances) == 0 {
		// If no specific targets, get all instances
		allInstances, err := s.awsManager.GetInstanceNames()
		if err != nil {
			log.Printf("Failed to get instance names for idle detection: %v", err)
			return false
		}
		instances = allInstances
	}

	// Check if any instance should be hibernated due to idle
	// We return true if ANY instance is idle (schedule will execute on all targets)
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	for _, instanceName := range instances {
		// Get AWS instance ID for CloudWatch metrics
		instanceID, err := s.awsManager.GetInstanceID(instanceName)
		if err != nil {
			log.Printf("Failed to get instance ID for %s: %v", instanceName, err)
			continue
		}

		// Check if instance is idle using CloudWatch metrics
		isIdle, err := s.metricsCollector.IsInstanceIdle(ctx, instanceID, schedule)
		if err != nil {
			log.Printf("Failed to check idle status for instance %s (ID: %s): %v", instanceName, instanceID, err)
			continue
		}

		if isIdle {
			log.Printf("Instance %s (ID: %s) detected as idle (CPU/network below thresholds for %d minutes)",
				instanceName, instanceID, schedule.IdleMinutes)
			return true
		}
	}

	return false
}

// shouldExecuteCustom checks custom schedule
func (s *Scheduler) shouldExecuteCustom(schedule *Schedule, now time.Time) bool {
	// Custom logic based on schedule configuration
	return false
}

// executeSchedule executes a hibernation schedule
func (s *Scheduler) executeSchedule(schedule *Schedule) {
	s.mu.Lock()
	s.active[schedule.ID] = &ScheduleExecution{
		ScheduleID: schedule.ID,
		StartTime:  time.Now(),
		IsActive:   true,
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if exec, exists := s.active[schedule.ID]; exists {
			exec.IsActive = false
		}
		s.mu.Unlock()
	}()

	log.Printf("Executing hibernation schedule: %s (ID: %s)", schedule.Name, schedule.ID)

	// Determine which instances to target
	var targetInstances []string
	if len(schedule.TargetInstances) > 0 {
		// Specific instances
		targetInstances = schedule.TargetInstances
	} else {
		// All instances - query AWS if manager available
		if s.awsManager != nil {
			instances, err := s.awsManager.GetInstanceNames()
			if err != nil {
				log.Printf("Failed to list instances for schedule %s: %v", schedule.Name, err)
				return
			}
			targetInstances = instances
		} else {
			log.Printf("No AWS manager available for schedule %s", schedule.Name)
			return
		}
	}

	// Execute hibernation action on each target instance
	successCount := 0
	failureCount := 0
	for _, instanceName := range targetInstances {
		if err := s.executeAction(schedule, instanceName); err != nil {
			log.Printf("Failed to execute action for instance %s in schedule %s: %v",
				instanceName, schedule.Name, err)
			failureCount++
		} else {
			successCount++
		}
	}

	// Update last executed time
	schedule.LastExecuted = time.Now()

	log.Printf("Schedule %s execution complete: %d succeeded, %d failed",
		schedule.Name, successCount, failureCount)
}

// executeAction executes the hibernation action on a single instance
func (s *Scheduler) executeAction(schedule *Schedule, instanceName string) error {
	if s.awsManager == nil {
		return fmt.Errorf("no AWS manager available")
	}

	switch schedule.HibernateAction {
	case "hibernate":
		return s.awsManager.HibernateInstance(instanceName)
	case "stop":
		return s.awsManager.StopInstance(instanceName)
	case "terminate":
		// Terminate is intentionally not implemented for safety
		return fmt.Errorf("terminate action not supported by scheduler (use CLI directly)")
	default:
		return fmt.Errorf("unknown hibernate action: %s", schedule.HibernateAction)
	}
}

// AddSchedule adds a new schedule
func (s *Scheduler) AddSchedule(schedule *Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if schedule.ID == "" {
		schedule.ID = generateScheduleID()
	}

	// Validate schedule
	if err := s.validateSchedule(schedule); err != nil {
		return err
	}

	// Calculate estimated savings
	schedule.EstimatedMonthlySavings = s.calculateEstimatedSavings(schedule)

	s.schedules[schedule.ID] = schedule
	return nil
}

// UpdateSchedule updates an existing schedule
func (s *Scheduler) UpdateSchedule(id string, updates *Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.schedules[id]
	if !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	// Merge updates
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.Type != "" {
		existing.Type = updates.Type
	}
	existing.Enabled = updates.Enabled

	// Update time settings
	if updates.StartTime != "" {
		existing.StartTime = updates.StartTime
	}
	if updates.EndTime != "" {
		existing.EndTime = updates.EndTime
	}
	if len(updates.DaysOfWeek) > 0 {
		existing.DaysOfWeek = updates.DaysOfWeek
	}

	// Recalculate savings
	existing.EstimatedMonthlySavings = s.calculateEstimatedSavings(existing)

	return nil
}

// DeleteSchedule removes a schedule
func (s *Scheduler) DeleteSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[id]; !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	delete(s.schedules, id)
	delete(s.active, id)

	return nil
}

// GetSchedule retrieves a schedule
func (s *Scheduler) GetSchedule(id string) (*Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[id]
	if !exists {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}

	return schedule, nil
}

// ListSchedules returns all schedules
func (s *Scheduler) ListSchedules() []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make([]*Schedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules
}

// validateSchedule validates a schedule configuration
func (s *Scheduler) validateSchedule(schedule *Schedule) error {
	if schedule.Name == "" {
		return fmt.Errorf("schedule name is required")
	}

	switch schedule.Type {
	case ScheduleTypeDaily, ScheduleTypeWeekly:
		if schedule.StartTime == "" || schedule.EndTime == "" {
			return fmt.Errorf("start and end times are required for %s schedule", schedule.Type)
		}
	case ScheduleTypeIdle:
		if schedule.IdleMinutes <= 0 {
			return fmt.Errorf("idle minutes must be positive for idle schedule")
		}
	}

	return nil
}

// calculateEstimatedSavings estimates monthly cost savings
func (s *Scheduler) calculateEstimatedSavings(schedule *Schedule) float64 {
	// Base calculation on schedule type and frequency
	hoursPerDay := 0.0
	daysPerMonth := 30.0

	switch schedule.Type {
	case ScheduleTypeDaily:
		// Calculate hours between start and end time
		hoursPerDay = calculateHoursBetween(schedule.StartTime, schedule.EndTime)
	case ScheduleTypeWeekly:
		hoursPerDay = calculateHoursBetween(schedule.StartTime, schedule.EndTime)
		daysPerMonth = float64(len(schedule.DaysOfWeek)) * 4 // Roughly 4 weeks per month
	case ScheduleTypeWorkHours:
		hoursPerDay = 15.0 // 6 PM to 9 AM next day + weekends
		daysPerMonth = 30.0
	case ScheduleTypeIdle:
		// Estimate based on idle threshold
		hoursPerDay = float64(schedule.IdleMinutes) / 60.0 * 8 // Assume 8 idle periods per day
	default:
		hoursPerDay = 8.0 // Default estimate
	}

	// Assume average instance cost of $0.10 per hour
	// This would be calculated based on actual instance types
	avgHourlyCost := 0.10
	monthlySavings := hoursPerDay * daysPerMonth * avgHourlyCost

	return monthlySavings
}

// Helper functions

func generateScheduleID() string {
	return fmt.Sprintf("sched-%d", time.Now().Unix())
}

func calculateHoursBetween(start, end string) float64 {
	// Parse HH:MM format and calculate hours
	// Simplified implementation
	return 8.0 // Default to 8 hours
}

// AssignScheduleToInstance assigns a schedule to a specific instance
func (s *Scheduler) AssignScheduleToInstance(scheduleID, instanceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify schedule exists
	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Add instance to schedule's target list if not already there
	found := false
	for _, target := range schedule.TargetInstances {
		if target == instanceName {
			found = true
			break
		}
	}
	if !found {
		schedule.TargetInstances = append(schedule.TargetInstances, instanceName)
	}

	// Track instance -> schedule mapping
	if scheduleIDs, exists := s.instanceSchedules[instanceName]; exists {
		// Check if already assigned
		for _, sid := range scheduleIDs {
			if sid == scheduleID {
				return nil // Already assigned
			}
		}
		s.instanceSchedules[instanceName] = append(scheduleIDs, scheduleID)
	} else {
		s.instanceSchedules[instanceName] = []string{scheduleID}
	}

	log.Printf("Assigned schedule %s (%s) to instance %s", scheduleID, schedule.Name, instanceName)
	return nil
}

// RemoveScheduleFromInstance removes a schedule from a specific instance
func (s *Scheduler) RemoveScheduleFromInstance(scheduleID, instanceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify schedule exists
	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Remove instance from schedule's target list
	newTargets := make([]string, 0)
	for _, target := range schedule.TargetInstances {
		if target != instanceName {
			newTargets = append(newTargets, target)
		}
	}
	schedule.TargetInstances = newTargets

	// Remove from instance -> schedule mapping
	if scheduleIDs, exists := s.instanceSchedules[instanceName]; exists {
		newScheduleIDs := make([]string, 0)
		for _, sid := range scheduleIDs {
			if sid != scheduleID {
				newScheduleIDs = append(newScheduleIDs, sid)
			}
		}
		if len(newScheduleIDs) > 0 {
			s.instanceSchedules[instanceName] = newScheduleIDs
		} else {
			delete(s.instanceSchedules, instanceName)
		}
	}

	log.Printf("Removed schedule %s (%s) from instance %s", scheduleID, schedule.Name, instanceName)
	return nil
}

// GetInstanceSchedules returns all schedules assigned to an instance
func (s *Scheduler) GetInstanceSchedules(instanceName string) []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheduleIDs, exists := s.instanceSchedules[instanceName]
	if !exists {
		return []*Schedule{}
	}

	schedules := make([]*Schedule, 0, len(scheduleIDs))
	for _, scheduleID := range scheduleIDs {
		if schedule, exists := s.schedules[scheduleID]; exists {
			schedules = append(schedules, schedule)
		}
	}

	return schedules
}

// AWSManagerAdapter adapts an AWS Manager to the AWSInstanceManager interface
// This allows the scheduler to work with the real AWS manager without direct dependencies
type AWSManagerAdapter struct {
	hibernateFn        func(string) error
	resumeFn           func(string) error
	stopFn             func(string) error
	startFn            func(string) error
	getInstanceNamesFn func() ([]string, error)
	getInstanceIDFn    func(string) (string, error)
}

// NewAWSManagerAdapter creates an adapter for an AWS manager
func NewAWSManagerAdapter(
	hibernateFn func(string) error,
	resumeFn func(string) error,
	stopFn func(string) error,
	startFn func(string) error,
	getInstanceNamesFn func() ([]string, error),
	getInstanceIDFn func(string) (string, error),
) *AWSManagerAdapter {
	return &AWSManagerAdapter{
		hibernateFn:        hibernateFn,
		resumeFn:           resumeFn,
		stopFn:             stopFn,
		startFn:            startFn,
		getInstanceNamesFn: getInstanceNamesFn,
		getInstanceIDFn:    getInstanceIDFn,
	}
}

func (a *AWSManagerAdapter) HibernateInstance(name string) error {
	return a.hibernateFn(name)
}

func (a *AWSManagerAdapter) ResumeInstance(name string) error {
	return a.resumeFn(name)
}

func (a *AWSManagerAdapter) StopInstance(name string) error {
	return a.stopFn(name)
}

func (a *AWSManagerAdapter) StartInstance(name string) error {
	return a.startFn(name)
}

func (a *AWSManagerAdapter) GetInstanceNames() ([]string, error) {
	return a.getInstanceNamesFn()
}

func (a *AWSManagerAdapter) GetInstanceID(name string) (string, error) {
	return a.getInstanceIDFn(name)
}
