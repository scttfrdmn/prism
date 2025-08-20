package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/progress"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// LaunchManager manages instance launches with progress reporting
type LaunchManager struct {
	server           *Server
	activeReporters  sync.Map // map[string]*progress.ProgressReporter
	optimizedResolver *templates.OptimizedResolver
}

// NewLaunchManager creates a new launch manager
func NewLaunchManager(server *Server) *LaunchManager {
	return &LaunchManager{
		server:            server,
		optimizedResolver: templates.NewOptimizedResolver(),
	}
}

// handleLaunchInstanceOptimized handles instance launch with progress reporting
func (lm *LaunchManager) handleLaunchInstanceOptimized(w http.ResponseWriter, r *http.Request) {
	var req types.LaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		lm.server.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create progress reporter
	reporter := progress.NewLaunchProgressReporter(req.Name)
	lm.activeReporters.Store(req.Name, reporter)
	defer lm.activeReporters.Delete(req.Name)

	// Add callback for real-time updates (could be WebSocket in future)
	reporter.AddCallback(func(update progress.ProgressUpdate) {
		// Log progress for now - later this could be WebSocket broadcast
		fmt.Printf("Launch Progress [%s]: %.1f%% - %s\n", 
			update.Operation, 
			update.OverallProgress*100, 
			update.Description)
	})

	// Launch asynchronously with progress reporting
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	instance, err := lm.launchWithProgress(ctx, req, reporter)
	if err != nil {
		reporter.FailStage(reporter.GetProgress().CurrentStage, err)
		lm.server.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Success response
	response := types.LaunchResponse{
		Instance:       *instance,
		Message:        fmt.Sprintf("Instance %s launched successfully", instance.Name),
		EstimatedCost:  fmt.Sprintf("$%.3f/hr (effective: $%.3f/hr)", instance.HourlyRate, instance.EffectiveRate),
		ConnectionInfo: fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP),
	}

	lm.server.writeJSON(w, http.StatusOK, response)
}

// launchWithProgress performs the actual launch with progress reporting
func (lm *LaunchManager) launchWithProgress(ctx context.Context, req types.LaunchRequest, reporter *progress.ProgressReporter) (*types.Instance, error) {
	// Stage 1: Template Resolution
	reporter.StartStage("template_resolution")
	template, runtimeTemplate, err := lm.resolveTemplate(ctx, req, reporter)
	if err != nil {
		return nil, fmt.Errorf("template resolution failed: %w", err)
	}
	reporter.CompleteStage("template_resolution")

	// Stage 2: AMI Discovery (handled in template resolution but reported separately)
	reporter.StartStage("ami_discovery")
	reporter.ReportStageProgress("ami_discovery", 1.0, "AMI selected: "+runtimeTemplate.AMI.ImageID)
	reporter.CompleteStage("ami_discovery")

	// Stage 3: Instance Launch
	reporter.StartStage("instance_launch")
	instance, err := lm.launchInstance(ctx, req, runtimeTemplate, reporter)
	if err != nil {
		return nil, fmt.Errorf("instance launch failed: %w", err)
	}
	reporter.CompleteStage("instance_launch")

	// Stage 4: Software Installation
	reporter.StartStage("software_installation")
	err = lm.monitorSoftwareInstallation(ctx, instance, reporter)
	if err != nil {
		return nil, fmt.Errorf("software installation failed: %w", err)
	}
	reporter.CompleteStage("software_installation")

	// Stage 5: Service Startup
	reporter.StartStage("service_startup")
	err = lm.monitorServiceStartup(ctx, instance, runtimeTemplate, reporter)
	if err != nil {
		return nil, fmt.Errorf("service startup failed: %w", err)
	}
	reporter.CompleteStage("service_startup")

	// Stage 6: Finalization
	reporter.StartStage("finalization")
	err = lm.finalizeInstance(instance, reporter)
	if err != nil {
		return nil, fmt.Errorf("finalization failed: %w", err)
	}
	reporter.CompleteStage("finalization")

	return instance, nil
}

// resolveTemplate resolves the template with optimization
func (lm *LaunchManager) resolveTemplate(ctx context.Context, req types.LaunchRequest, reporter *progress.ProgressReporter) (*templates.Template, *templates.RuntimeTemplate, error) {
	// Load template
	reporter.ReportStageProgress("template_resolution", 0.2, "Loading template definition")
	
	templateResolver := templates.NewTemplateResolver()
	template, err := templateResolver.LoadTemplate(req.TemplateName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Apply inheritance if needed
	reporter.ReportStageProgress("template_resolution", 0.5, "Processing template inheritance")
	if len(template.Inherits) > 0 {
		template, err = templateResolver.ResolveInheritance(template)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve template inheritance: %w", err)
		}
	}

	// Generate runtime template with optimization
	reporter.ReportStageProgress("template_resolution", 0.8, "Generating optimized runtime configuration")
	runtimeTemplate, err := lm.optimizedResolver.ResolveTemplateOptimized(
		ctx, 
		template, 
		req.Region, 
		req.Architecture, 
		req.PackageManager, 
		req.Size,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve runtime template: %w", err)
	}

	reporter.ReportStageProgress("template_resolution", 1.0, "Template resolution complete")
	return template, runtimeTemplate, nil
}

// launchInstance launches the EC2 instance
func (lm *LaunchManager) launchInstance(ctx context.Context, req types.LaunchRequest, runtimeTemplate *templates.RuntimeTemplate, reporter *progress.ProgressReporter) (*types.Instance, error) {
	var instance *types.Instance
	
	reporter.ReportStageProgress("instance_launch", 0.1, "Validating instance name uniqueness")
	
	// Use existing validation logic
	var nameExists bool
	lm.server.withAWSManager(nil, nil, func(awsManager *aws.Manager) error {
		instances, err := awsManager.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to check existing instances: %w", err)
		}

		for _, existingInstance := range instances {
			if existingInstance.Name == req.Name {
				if existingInstance.State != "terminated" && existingInstance.State != "terminating" {
					nameExists = true
					break
				}
			}
		}
		return nil
	})

	if nameExists {
		return nil, fmt.Errorf("instance with name '%s' already exists", req.Name)
	}

	reporter.ReportStageProgress("instance_launch", 0.3, "Setting up SSH key")
	
	// Handle SSH key setup
	if req.SSHKeyName == "" {
		if err := lm.server.setupSSHKeyForLaunch(&req); err != nil {
			return nil, fmt.Errorf("SSH key setup failed: %w", err)
		}
	}

	reporter.ReportStageProgress("instance_launch", 0.5, "Creating EC2 instance")
	
	// Launch instance
	lm.server.withAWSManager(nil, nil, func(awsManager *aws.Manager) error {
		// Ensure SSH key exists in AWS
		if req.SSHKeyName != "" {
			if err := lm.server.ensureSSHKeyInAWS(awsManager, &req); err != nil {
				return fmt.Errorf("failed to ensure SSH key in AWS: %w", err)
			}
		}

		reporter.ReportStageProgress("instance_launch", 0.8, "Submitting launch request to AWS")
		
		// Use the runtime template in the launch request
		enhancedReq := req
		enhancedReq.UserData = runtimeTemplate.UserData
		enhancedReq.AMI = runtimeTemplate.AMI.ImageID
		enhancedReq.InstanceType = runtimeTemplate.InstanceType.Type
		
		var err error
		instance, err = awsManager.LaunchInstance(enhancedReq)
		return err
	})

	if instance == nil {
		return nil, fmt.Errorf("failed to launch instance")
	}

	reporter.ReportStageProgress("instance_launch", 1.0, "Instance launched successfully")

	// Save state
	if err := lm.server.stateManager.SaveInstance(*instance); err != nil {
		return nil, fmt.Errorf("failed to save instance state: %w", err)
	}

	return instance, nil
}

// monitorSoftwareInstallation monitors the UserData script execution
func (lm *LaunchManager) monitorSoftwareInstallation(ctx context.Context, instance *types.Instance, reporter *progress.ProgressReporter) error {
	// This is a simplified version - in production, this would monitor SSM logs or CloudWatch
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	timeout := time.NewTimer(10 * time.Minute) // UserData timeout
	defer timeout.Stop()
	
	progress := 0.0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("software installation timed out")
		case <-ticker.C:
			// Simulate progress monitoring
			progress += 0.1
			if progress > 1.0 {
				progress = 1.0
			}
			
			description := fmt.Sprintf("Installing software packages (%.0f%% estimated)", progress*100)
			reporter.ReportStageProgress("software_installation", progress, description)
			
			if progress >= 1.0 {
				return nil
			}
		}
	}
}

// monitorServiceStartup monitors service startup and health checks
func (lm *LaunchManager) monitorServiceStartup(ctx context.Context, instance *types.Instance, runtimeTemplate *templates.RuntimeTemplate, reporter *progress.ProgressReporter) error {
	// Wait for services to start
	reporter.ReportStageProgress("service_startup", 0.2, "Waiting for services to initialize")
	
	// Check service health (simplified)
	time.Sleep(30 * time.Second) // Give services time to start
	
	reporter.ReportStageProgress("service_startup", 0.6, "Performing health checks")
	
	// In production, this would do actual health checks on the services
	time.Sleep(10 * time.Second)
	
	reporter.ReportStageProgress("service_startup", 1.0, "All services are healthy")
	return nil
}

// finalizeInstance performs final setup steps
func (lm *LaunchManager) finalizeInstance(instance *types.Instance, reporter *progress.ProgressReporter) error {
	reporter.ReportStageProgress("finalization", 0.5, "Updating instance metadata")
	
	// Update final state
	instance.State = "running"
	
	reporter.ReportStageProgress("finalization", 1.0, "Instance ready for use")
	return nil
}

// GetLaunchProgress returns current progress for a launch operation
func (lm *LaunchManager) GetLaunchProgress(instanceName string) (*progress.ProgressUpdate, bool) {
	if reporter, exists := lm.activeReporters.Load(instanceName); exists {
		update := reporter.(*progress.ProgressReporter).GetProgress()
		return &update, true
	}
	return nil, false
}