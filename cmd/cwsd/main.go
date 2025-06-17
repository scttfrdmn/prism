package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yourusername/cloudworkstation/pkg/daemon"
)

const (
	version = "0.1.0"
)

func main() {
	var (
		port    = flag.String("port", "8080", "Port to listen on")
		showVer = flag.Bool("version", false, "Show version")
		help    = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	if *showVer {
		fmt.Printf("CloudWorkstation Daemon v%s\n", version)
		return
	}

	log.Printf("CloudWorkstation Daemon v%s starting...", version)

	server, err := daemon.NewServer(*port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func printUsage() {
	fmt.Printf("CloudWorkstation Daemon v%s\n\n", version)
	fmt.Println("The CloudWorkstation daemon provides a REST API for managing cloud research environments.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [options]\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("  GET    /api/v1/ping              - Health check")
	fmt.Println("  GET    /api/v1/status            - Daemon status")
	fmt.Println("  GET    /api/v1/instances         - List instances")
	fmt.Println("  POST   /api/v1/instances         - Launch instance")
	fmt.Println("  GET    /api/v1/instances/{name}  - Get instance details")
	fmt.Println("  DELETE /api/v1/instances/{name}  - Delete instance")
	fmt.Println("  POST   /api/v1/instances/{name}/start - Start instance")
	fmt.Println("  POST   /api/v1/instances/{name}/stop  - Stop instance")
	fmt.Println("  GET    /api/v1/templates         - List templates")
	fmt.Println("  GET    /api/v1/volumes           - List EFS volumes")
	fmt.Println("  POST   /api/v1/volumes           - Create EFS volume")
	fmt.Println("  GET    /api/v1/storage           - List EBS volumes")
	fmt.Println("  POST   /api/v1/storage           - Create EBS volume")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                    # Start daemon on port 8080\n", os.Args[0])
	fmt.Printf("  %s -port 9000        # Start daemon on port 9000\n", os.Args[0])
}