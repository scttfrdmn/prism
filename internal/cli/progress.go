// Package cli provides enhanced progress reporting for Prism launch operations.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// ProgressReporter provides enhanced real-time progress reporting for Prism operations
type ProgressReporter struct {
	instanceName   string
	templateName   string
	templateType   string
	startTime      time.Time
	currentStage   int
	totalStages    int
	stageStartTime time.Time
	estimatedTotal time.Duration
}

// ProgressStage represents a stage in the launch process
type ProgressStage struct {
	Name        string
	Description string
	Icon        string
	MinDuration time.Duration
	MaxDuration time.Duration
}

// NewProgressReporter creates a new enhanced progress reporter
func NewProgressReporter(instanceName, templateName string, template *types.Template) *ProgressReporter {
	templateType := "package"

	// Determine template type based on template properties
	// Priority: UserData (package-based) > AMI map (AMI-based) > template name
	if template != nil {
		// If template has UserData, it's package-based (will run installation scripts)
		// If template has AMI map populated, it's AMI-based (pre-built image)
		if len(template.UserData) > 0 {
			templateType = "package"
		} else if len(template.AMI) > 0 {
			// Check if AMI map actually has entries (not just empty nested maps)
			hasAMI := false
			for _, archMap := range template.AMI {
				if len(archMap) > 0 {
					hasAMI = true
					break
				}
			}
			if hasAMI {
				templateType = "ami"
			}
		}
	}

	// Fallback: check template name for AMI indicators
	if templateType == "package" && strings.Contains(strings.ToLower(templateName), "ami") {
		templateType = "ami"
	}

	now := time.Now()
	reporter := &ProgressReporter{
		instanceName:   instanceName,
		templateName:   templateName,
		templateType:   templateType,
		startTime:      now,
		stageStartTime: now,
		currentStage:   0,
	}

	// Set total stages based on template type
	if templateType == "ami" {
		reporter.totalStages = 3 // Initialize, Start, Ready
		reporter.estimatedTotal = 3 * time.Minute
	} else {
		reporter.totalStages = 6 // Initialize, Start, Setup, Packages, Services, Ready
		reporter.estimatedTotal = 8 * time.Minute

		// Adjust estimate based on template characteristics
		// For package-based templates, use heuristics based on name
		templateLower := strings.ToLower(templateName)
		switch {
		case strings.Contains(templateLower, "conda"):
			reporter.estimatedTotal = 12 * time.Minute
		case strings.Contains(templateLower, "ml") || strings.Contains(templateLower, "deep"):
			reporter.estimatedTotal = 10 * time.Minute
		case strings.Contains(templateLower, "simple"):
			reporter.estimatedTotal = 4 * time.Minute
		default:
			reporter.estimatedTotal = 6 * time.Minute
		}
	}

	return reporter
}

// GetProgressStages returns the stages for the current template type
func (pr *ProgressReporter) GetProgressStages() []ProgressStage {
	if pr.templateType == "ami" {
		return []ProgressStage{
			{"initialize", "Initializing instance", "⏳", 10 * time.Second, 30 * time.Second},
			{"starting", "Starting instance", "🔄", 30 * time.Second, 2 * time.Minute},
			{"ready", "Instance ready", "✅", 0, 30 * time.Second},
		}
	} else {
		return []ProgressStage{
			{"initialize", "Initializing instance", "⏳", 10 * time.Second, 30 * time.Second},
			{"starting", "Starting instance", "🔄", 30 * time.Second, 2 * time.Minute},
			{"setup", "Beginning setup", "🔧", 30 * time.Second, 1 * time.Minute},
			{"packages", "Installing packages", "📥", 1 * time.Minute, 8 * time.Minute},
			{"services", "Configuring services", "⚙️", 30 * time.Second, 2 * time.Minute},
			{"ready", "Instance ready", "✅", 0, 30 * time.Second},
		}
	}
}

// ShowHeader displays the enhanced progress header
func (pr *ProgressReporter) ShowHeader() {
	fmt.Printf("🚀 Launching '%s' using template '%s'\n", pr.instanceName, pr.templateName)
	fmt.Printf("📋 Template type: %s-based (%s estimated)\n",
		strings.ToUpper(pr.templateType),
		pr.formatDuration(pr.estimatedTotal))
	fmt.Printf("⏱️  Started: %s\n\n", pr.startTime.Format("15:04:05"))
}

// UpdateProgress updates and displays current progress
func (pr *ProgressReporter) UpdateProgress(instance *types.Instance, elapsed time.Duration) {
	stages := pr.GetProgressStages()

	// Determine current stage based on instance state
	stageIndex := pr.getStageIndexFromState(instance.State)

	// Update stage if changed
	if stageIndex != pr.currentStage && stageIndex >= 0 {
		pr.currentStage = stageIndex
		pr.stageStartTime = time.Now()
	}

	// Calculate progress percentage
	progressPercent := float64(pr.currentStage) / float64(pr.totalStages) * 100
	if pr.currentStage >= pr.totalStages {
		progressPercent = 100
	}

	// Show progress bar
	pr.showProgressBar(progressPercent)

	// Show current stage
	if pr.currentStage < len(stages) {
		stage := stages[pr.currentStage]
		stageElapsed := time.Since(pr.stageStartTime)
		fmt.Printf("%s %s (%s)\n",
			stage.Icon,
			stage.Description,
			pr.formatDuration(stageElapsed))
	}

	// Show time information
	pr.showTimeInfo(elapsed)

	// Show cost information if available
	if instance.InstanceType != "" {
		pr.showCostInfo(instance, elapsed)
	}

	fmt.Println() // Add spacing
}

// showProgressBar displays a visual progress bar
func (pr *ProgressReporter) showProgressBar(percent float64) {
	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled && percent < 100 {
			bar += "▌"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %.1f%%", percent)

	fmt.Printf("📊 Progress: %s\n", bar)
}

// showTimeInfo displays timing information
func (pr *ProgressReporter) showTimeInfo(elapsed time.Duration) {
	remaining := pr.estimatedTotal - elapsed
	if remaining < 0 {
		remaining = 0
	}

	fmt.Printf("⏱️  Elapsed: %s | Remaining: ~%s\n",
		pr.formatDuration(elapsed),
		pr.formatDuration(remaining))
}

// showCostInfo displays cost information
func (pr *ProgressReporter) showCostInfo(instance *types.Instance, elapsed time.Duration) {
	// Simple cost estimation - in real implementation this would use pricing calculator
	hourlyCost := 0.10 // Default estimate

	switch {
	case strings.Contains(instance.InstanceType, "t3.micro"):
		hourlyCost = 0.0104
	case strings.Contains(instance.InstanceType, "t3.small"):
		hourlyCost = 0.0208
	case strings.Contains(instance.InstanceType, "t3.medium"):
		hourlyCost = 0.0416
	case strings.Contains(instance.InstanceType, "t3.large"):
		hourlyCost = 0.0832
	}

	if instance.InstanceLifecycle == "spot" {
		hourlyCost *= 0.3 // Spot discount
	}

	currentCost := hourlyCost * elapsed.Hours()

	fmt.Printf("💰 Instance: %s (%s) | Cost so far: $%.4f\n",
		instance.InstanceType,
		instance.InstanceLifecycle,
		currentCost)
}

// ShowCompletion displays completion information
func (pr *ProgressReporter) ShowCompletion(instance *types.Instance) {
	totalTime := time.Since(pr.startTime)

	fmt.Printf("🎉 Launch Complete!\n")
	fmt.Printf("✅ Instance '%s' is ready for use\n", pr.instanceName)
	fmt.Printf("⏱️  Total time: %s\n", pr.formatDuration(totalTime))

	if instance.PublicIP != "" {
		fmt.Printf("🌐 Public IP: %s\n", instance.PublicIP)
	}

	fmt.Printf("🔗 Connect: cws connect %s\n", pr.instanceName)

	// Show setup summary
	if pr.templateType == "package" {
		fmt.Printf("📦 Template setup completed successfully\n")
	} else {
		fmt.Printf("📦 AMI instance launched and ready\n")
	}
}

// ShowError displays enhanced error information
func (pr *ProgressReporter) ShowError(err error, instance *types.Instance) {
	totalTime := time.Since(pr.startTime)

	fmt.Printf("❌ Launch Failed\n")
	fmt.Printf("⏱️  Failed after: %s\n", pr.formatDuration(totalTime))

	if instance != nil {
		fmt.Printf("📊 Final state: %s\n", instance.State)
		if instance.PublicIP != "" {
			fmt.Printf("🌐 Instance IP: %s\n", instance.PublicIP)
		}
	}

	fmt.Printf("💡 Troubleshooting:\n")
	fmt.Printf("   • Check logs: cws daemon logs\n")
	fmt.Printf("   • Retry with: cws launch %s %s\n", pr.templateName, pr.instanceName)
	fmt.Printf("   • Try different region: --region us-west-2\n")
	fmt.Printf("   • Try smaller size: --size S\n")
}

// getStageIndexFromState maps instance state to progress stage
func (pr *ProgressReporter) getStageIndexFromState(state string) int {
	if pr.templateType == "ami" {
		switch state {
		case "pending", "initializing":
			return 0 // initialize
		case "starting":
			return 1 // starting
		case "running":
			return 2 // ready
		default:
			return -1 // unknown state
		}
	} else {
		switch state {
		case "pending", "initializing":
			return 0 // initialize
		case "starting":
			return 1 // starting
		case "running":
			// For package-based, we need to determine sub-stage
			// This would ideally check setup completion status
			return 2 // setup (could be 2, 3, 4, or 5 depending on actual progress)
		default:
			return -1 // unknown state
		}
	}
}

// FormatDuration formats a duration in a human-readable way (exported for access from app.go)
func (pr *ProgressReporter) FormatDuration(d time.Duration) string {
	return pr.formatDuration(d)
}

// formatDuration formats a duration in a human-readable way
func (pr *ProgressReporter) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// Spinner provides an animated terminal spinner for long-running operations
type Spinner struct {
	frames   []string
	message  string
	delay    time.Duration
	writer   io.Writer
	stopChan chan struct{}
	wg       sync.WaitGroup
	active   bool
	mu       sync.Mutex
}

// Default spinner frames (various styles available)
var (
	// DotsSpinner is a simple dots animation
	DotsSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// LineSpinner is a simple line rotation
	LineSpinner = []string{"|", "/", "-", "\\"}
	// ArrowSpinner is a rotating arrow
	ArrowSpinner = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	// CircleSpinner is a circle animation
	CircleSpinner = []string{"◐", "◓", "◑", "◒"}
	// BoxSpinner is a box bouncing animation
	BoxSpinner = []string{"◰", "◳", "◲", "◱"}
	// EarthSpinner is a rotating earth (fun option!)
	EarthSpinner = []string{"🌍", "🌎", "🌏"}
)

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   DotsSpinner, // Default to dots
		message:  message,
		delay:    80 * time.Millisecond,
		writer:   os.Stdout,
		stopChan: make(chan struct{}),
	}
}

// WithFrames sets custom spinner frames
func (s *Spinner) WithFrames(frames []string) *Spinner {
	s.frames = frames
	return s
}

// WithDelay sets the animation delay
func (s *Spinner) WithDelay(delay time.Duration) *Spinner {
	s.delay = delay
	return s
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return // Already running
	}
	s.active = true
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		frameIndex := 0
		ticker := time.NewTicker(s.delay)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				// Clear the spinner line
				fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+5))
				return
			case <-ticker.C:
				// Print current frame
				frame := s.frames[frameIndex%len(s.frames)]
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				fmt.Fprintf(s.writer, "\r%s %s", frame, msg)
				frameIndex++
			}
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return // Not running
	}
	s.active = false
	s.mu.Unlock()

	close(s.stopChan)
	s.wg.Wait()
}

// UpdateMessage updates the spinner message while it's running
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// StopWithMessage stops the spinner and prints a final message
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Fprintln(s.writer, message)
}
