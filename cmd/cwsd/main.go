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

	"github.com/scttfrdmn/cloudworkstation/pkg/daemon"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

func main() {
	var (
		port    = flag.String("port", "8947", "Port to listen on (default: 8947 - CWS on phone keypad)")
		showVer = flag.Bool("version", false, "Show version")
		help    = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	if *showVer {
		fmt.Println(version.GetDaemonVersionInfo())
		return
	}

	log.Printf("CloudWorkstation Daemon v%s starting...", version.GetVersion())

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start the main daemon server with integrated monitoring
	server, err := daemon.NewServer(*port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
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
			log.Printf("🔄 Configuration reload requested")
			// Integrated monitoring will automatically restart if idle detection is re-enabled

		case syscall.SIGINT, syscall.SIGTERM:
			log.Printf("🛑 Graceful shutdown requested")

			// Stop main server (which includes integrated monitoring)
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
	fmt.Println("with integrated autonomous idle detection for automated cost savings.")
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
	fmt.Printf("  %s                           # Start daemon with integrated monitoring\n", os.Args[0])
	fmt.Printf("  %s -port 9000               # Custom port\n", os.Args[0])
	fmt.Println()
	fmt.Println("Integrated Autonomous Monitoring:")
	fmt.Println("  The daemon automatically provides autonomous idle detection when enabled via API:")
	fmt.Println("  • Monitor running instances for idle activity every minute")
	fmt.Println("  • Automatically hibernate or stop idle instances to save costs")
	fmt.Println("  • Leverages existing SSH connectivity for efficient monitoring")
	fmt.Println("  • No separate processes or command switches required")
	fmt.Println()
	fmt.Println("  Enable idle detection: curl -X POST http://localhost:8947/api/v1/idle/enable")
}
