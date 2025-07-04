// CloudWorkstation CLI client (cws) - Launch research environments in seconds.
//
// The cws command-line tool provides a simple interface for managing cloud
// research workstations. It communicates with the CloudWorkstation daemon (cwsd)
// to launch pre-configured environments optimized for academic research.
//
// Core Commands:
//
//	cws launch template-name instance-name  # Launch new research environment
//	cws list                                # Show running instances and costs
//	cws connect instance-name               # Get connection information
//	cws stop/start instance-name            # Manage instance lifecycle
//
// Storage Commands:
//
//	cws volumes create/list/delete          # Manage EFS shared storage
//	cws storage create/list/delete          # Manage EBS high-performance storage
//
// Examples:
//
//	cws launch r-research my-analysis       # Launch R environment
//	cws launch python-ml gpu-training --size GPU-L  # Launch ML environment
//	cws list                                # Show all instances
//	cws connect my-analysis                 # Get SSH/web URLs
//
// The CLI implements CloudWorkstation's "Default to Success" principle -
// every command works out of the box with smart defaults while providing
// advanced options for power users.
package main

import (
	"fmt"
	"os"

	"github.com/scttfrdmn/cloudworkstation/internal/cli"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	cliApp := cli.NewApp(version.GetVersion())

	switch command {
	case "launch":
		err := cliApp.Launch(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		err := cliApp.List(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "connect":
		err := cliApp.Connect(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		err := cliApp.Stop(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "start":
		err := cliApp.Start(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "delete":
		err := cliApp.Delete(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "volume":
		err := cliApp.Volume(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "storage":
		err := cliApp.Storage(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "templates":
		err := cliApp.Templates(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "daemon":
		err := cliApp.Daemon(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "ami":
		err := cliApp.AMI(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println(version.GetVersionInfo())
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("CloudWorkstation CLI v%s\n\n", version.GetVersion())
	fmt.Println("Launch pre-configured cloud workstations for research in seconds.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cws <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  launch <template> <name> [options]  Launch a new workstation")
	fmt.Println("  list                                List all workstations")
	fmt.Println("  connect <name>                      Get connection info for workstation")
	fmt.Println("  stop <name>                         Stop a workstation")
	fmt.Println("  start <name>                        Start a stopped workstation")
	fmt.Println("  delete <name>                       Delete a workstation")
	fmt.Println("  templates                           List available templates")
	fmt.Println()
	fmt.Println("  volume <action> [args]              Manage EFS volumes")
	fmt.Println("    create <name> [options]           Create new EFS volume")
	fmt.Println("    list                              List EFS volumes")
	fmt.Println("    info <name>                       Show EFS volume details")
	fmt.Println("    delete <name>                     Delete EFS volume")
	fmt.Println()
	fmt.Println("  storage <action> [args]             Manage EBS volumes")
	fmt.Println("    create <name> <size> [type]       Create new EBS volume")
	fmt.Println("    list                              List EBS volumes")
	fmt.Println("    info <name>                       Show EBS volume details")
	fmt.Println("    attach <volume> <instance>        Attach EBS volume to instance")
	fmt.Println("    detach <volume>                   Detach EBS volume")
	fmt.Println("    delete <name>                     Delete EBS volume")
	fmt.Println()
	fmt.Println("  daemon <action>                     Manage daemon")
	fmt.Println("    start                             Start daemon")
	fmt.Println("    stop                              Stop daemon")
	fmt.Println("    status                            Show daemon status")
	fmt.Println("    logs                              Show daemon logs")
	fmt.Println()
	fmt.Println("  ami <action> [args]                 Manage AMIs for templates")
	fmt.Println("    build <template> [options]        Build AMI from template")
	fmt.Println("    list [template]                   List available AMIs")
	fmt.Println("    validate <template>               Validate AMI template")
	fmt.Println("    publish <template> <ami-id>       Register AMI in registry")
	fmt.Println()
	fmt.Println("  version                             Show version")
	fmt.Println("  help                                Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cws launch r-research my-analysis           # Launch R environment")
	fmt.Println("  cws launch python-research ml-project --size L  # Launch Python with large instance")
	fmt.Println("  cws list                                    # List all workstations")
	fmt.Println("  cws connect my-analysis                     # Get connection details")
	fmt.Println("  cws volume create shared-data               # Create shared EFS volume")
	fmt.Println("  cws storage create fast-storage XL io2     # Create high-performance storage")
	fmt.Println("  cws ami build neuroimaging                  # Build AMI from template")
	fmt.Println()
	fmt.Println("Template sizes: XS, S, M, L, XL, GPU-S, GPU-M, GPU-L")
	fmt.Println("Storage sizes: XS (100GB), S (500GB), M (1TB), L (2TB), XL (4TB)")
}
