// CloudWorkstation Daemon (cwsd) - Background service for AWS operations.
//
// The cwsd daemon provides a REST API server for CloudWorkstation operations.
// It manages AWS resources, maintains state, and serves requests from CLI and
// GUI clients. The daemon handles all AWS authentication, resource management,
// and cost tracking while providing a simple API interface.
//
// Server Features:
//   - REST API for all CloudWorkstation operations
//   - Background AWS resource management
//   - State persistence and synchronization
//   - Cost tracking and billing integration
//   - Health monitoring and logging
//
// API Endpoints:
//
//	POST /instances                         # Launch new instances
//	GET /instances                          # List instances
//	GET /instances/{name}                   # Get instance details
//	DELETE /instances/{name}                # Terminate instance
//	POST/GET/DELETE /volumes/{name}         # EFS volume management
//	POST/GET/DELETE /storage/{name}         # EBS storage management
//
// Usage:
//
//	cwsd                                    # Start daemon on :8080
//	cwsd -port 9000                         # Start on custom port
//	cwsd -config /path/to/config.json      # Use custom config
//
// The daemon implements CloudWorkstation's core principles of reliability,
// cost transparency, and zero-surprise operations.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/daemon"
	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)


func main() {
	var (
		port       = flag.String("port", "8947", "Port to listen on (default: 8947 - CWS on phone keypad)")
		autonomous = flag.Bool("autonomous", false, "Enable autonomous idle detection and cost-saving actions")
		dryRun     = flag.Bool("dry-run", false, "Run in dry-run mode (log actions but don't execute)")
		showVer    = flag.Bool("version", false, "Show version")
		help       = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	if *showVer {
		fmt.Println(version.GetVersionInfo())
		return
	}

	log.Printf("CloudWorkstation Daemon v%s starting...", version.GetVersion())

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start the main daemon server
	server, err := daemon.NewServer(*port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start autonomous idle detection if requested
	var resilientService *idle.ResilientIdleService
	if *autonomous {
		log.Printf("🤖 Autonomous idle detection enabled")
		
		// Initialize AWS manager for autonomous service
		awsManager, err := aws.NewManager()
		if err != nil {
			log.Fatalf("Failed to initialize AWS manager for autonomous service: %v", err)
		}

		// Create autonomous config
		autonomousConfig := idle.DefaultAutonomousConfig()
		if *dryRun {
			autonomousConfig.DryRun = true
			autonomousConfig.AutoExecute = false
			log.Printf("🔍 Running in DRY-RUN mode - will log actions but not execute them")
		}

		// Get idle manager from server
		idleManager := server.GetIdleManager()
		if idleManager == nil {
			log.Fatalf("Failed to get idle manager from server")
		}

		// Create resilient autonomous service
		resilientService, err = idle.NewResilientIdleService(idleManager, awsManager, autonomousConfig)
		if err != nil {
			log.Fatalf("Failed to create autonomous idle service: %v", err)
		}

		// Start autonomous service
		if err := resilientService.Start(); err != nil {
			log.Fatalf("Failed to start autonomous idle service: %v", err)
		}

		log.Printf("✅ Autonomous idle detection started successfully")
	}

	// Start the daemon server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signals or server error
	select {
	case err := <-serverErr:
		log.Fatalf("Server failed: %v", err)
	case sig := <-sigChan:
		log.Printf("🔔 Received signal: %v", sig)
		
		switch sig {
		case syscall.SIGHUP:
			log.Printf("🔄 Reload requested - restarting autonomous service")
			if resilientService != nil {
				if err := resilientService.Stop(); err != nil {
					log.Printf("Error stopping autonomous service: %v", err)
				}
				if err := resilientService.Start(); err != nil {
					log.Printf("Error restarting autonomous service: %v", err)
				} else {
					log.Printf("✅ Autonomous service reloaded")
				}
			}
		case syscall.SIGINT, syscall.SIGTERM:
			log.Printf("🛑 Graceful shutdown requested")
			
			// Stop autonomous service first
			if resilientService != nil {
				log.Printf("Stopping autonomous idle service...")
				if err := resilientService.Stop(); err != nil {
					log.Printf("Error stopping autonomous service: %v", err)
				} else {
					log.Printf("✅ Autonomous idle service stopped")
				}
			}
			
			// Stop main server
			log.Printf("Stopping daemon server...")
			if err := server.Stop(); err != nil {
				log.Printf("Error stopping server: %v", err)
			} else {
				log.Printf("✅ Daemon server stopped")
			}
			
			log.Printf("✅ CloudWorkstation daemon shutdown complete")
			os.Exit(0)
		}
	}
}

func printUsage() {
	fmt.Printf("CloudWorkstation Daemon v%s\n\n", version.GetVersion())
	fmt.Println("The CloudWorkstation daemon provides a REST API for managing cloud research environments")
	fmt.Println("with optional autonomous idle detection for automated cost savings.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Core API Endpoints:")
	fmt.Println("  GET    /api/v1/ping              - Health check")
	fmt.Println("  GET    /api/v1/status            - Daemon status")
	fmt.Println("  POST   /api/v1/shutdown          - Shutdown daemon")
	fmt.Println("  GET    /api/v1/instances         - List instances")
	fmt.Println("  POST   /api/v1/instances         - Launch instance")
	fmt.Println("  GET    /api/v1/instances/{name}  - Get instance details")
	fmt.Println("  DELETE /api/v1/instances/{name}  - Delete instance")
	fmt.Println("  POST   /api/v1/instances/{name}/start - Start instance")
	fmt.Println("  POST   /api/v1/instances/{name}/stop  - Stop instance")
	fmt.Println("  POST   /api/v1/instances/{name}/hibernate - Hibernate instance")
	fmt.Println("  GET    /api/v1/templates         - List templates")
	fmt.Println("  GET    /api/v1/volumes           - List EFS volumes")
	fmt.Println("  POST   /api/v1/volumes           - Create EFS volume")
	fmt.Println("  GET    /api/v1/storage           - List EBS volumes")
	fmt.Println("  POST   /api/v1/storage           - Create EBS volume")
	fmt.Println()
	fmt.Println("Idle Detection API Endpoints:")
	fmt.Println("  GET    /api/v1/idle/status       - Idle detection status")
	fmt.Println("  POST   /api/v1/idle/enable       - Enable idle detection")
	fmt.Println("  POST   /api/v1/idle/disable      - Disable idle detection")
	fmt.Println("  GET    /api/v1/idle/profiles     - List idle profiles")
	fmt.Println("  GET    /api/v1/idle/history      - Show idle action history")
	fmt.Println("  GET    /api/v1/idle/pending-actions - Show pending actions")
	fmt.Println("  POST   /api/v1/idle/execute-actions - Execute pending actions")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                           # Basic daemon\n", os.Args[0])
	fmt.Printf("  %s -port 9000               # Custom port\n", os.Args[0])
	fmt.Printf("  %s -autonomous              # Enable autonomous idle detection\n", os.Args[0])
	fmt.Printf("  %s -autonomous -dry-run     # Test autonomous mode (no actions)\n", os.Args[0])
	fmt.Println()
	fmt.Println("Autonomous Mode:")
	fmt.Println("  When -autonomous is enabled, the daemon will:")
	fmt.Println("  • Monitor running instances for idle activity (CPU, memory, user input)")
	fmt.Println("  • Automatically hibernate or stop idle instances to save costs")
	fmt.Println("  • Persist state across daemon restarts and system reboots")
	fmt.Println("  • Provide comprehensive logging and history of all actions")
	fmt.Println()
	fmt.Println("  Use -dry-run to test autonomous mode without executing actions.")
}
