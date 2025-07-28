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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/internal/cli"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

func main() {
	// Define flags
	versionFlag := flag.Bool("version", false, "Show version information")
	helpFlag := flag.Bool("help", false, "Show help information")
	
	// Parse flags but keep command and arguments separate
	flag.CommandLine.Parse(os.Args[1:])
	
	// Handle version and help flags
	if *versionFlag {
		fmt.Println(version.GetVersionInfo())
		return
	}
	
	if *helpFlag {
		printUsage()
		return
	}
	
	// Get remaining arguments after flag parsing
	remaining := flag.Args()
	if len(remaining) == 0 {
		printUsage()
		os.Exit(1)
	}
	
	// First argument is the command
	command := remaining[0]
	args := remaining[1:]
	
	// Load configuration from config.json
	configFile := "config.json"
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		// Try to find config in same directory as binary
		execPath, err := os.Executable()
		if err == nil {
			configPath := filepath.Join(filepath.Dir(execPath), "config.json")
			configData, err = ioutil.ReadFile(configPath)
		}
	}
	
	// Define config structure
	config := struct {
		AWS struct {
			Profile       string `json:"profile"`
			Region        string `json:"region"`
			VpcID         string `json:"vpc_id"`
			SubnetID      string `json:"subnet_id"`
			SecurityGroup string `json:"security_group_id"`
		} `json:"aws"`
		Daemon struct {
			Port int    `json:"port"`
			Host string `json:"host"`
		} `json:"daemon"`
	}{}
	
	// Set defaults
	config.AWS.Profile = "aws"
	config.AWS.Region = "us-west-2"
	config.AWS.SubnetID = "subnet-06a8cff8a4457b4a7" // Fixed subnet
	config.AWS.VpcID = "vpc-e7e2999f" // Fixed VPC
	config.Daemon.Host = "localhost"
	config.Daemon.Port = 8091
	
	// Parse config if found
	if configData != nil {
		if err := json.Unmarshal(configData, &config); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse config file: %v\n", err)
		}
	}

	// Set AWS environment variables if not already set
	if os.Getenv("AWS_PROFILE") == "" {
		os.Setenv("AWS_PROFILE", config.AWS.Profile)
	}
	
	if os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
		os.Setenv("AWS_REGION", config.AWS.Region)
	}
	
	// Set environment variables for subnet and VPC
	os.Setenv("CWS_SUBNET_ID", config.AWS.SubnetID)
	os.Setenv("CWS_VPC_ID", config.AWS.VpcID)
	
	// Set daemon URL if not already set
	if os.Getenv("CWSD_URL") == "" {
		os.Setenv("CWSD_URL", fmt.Sprintf("http://%s:%d", config.Daemon.Host, config.Daemon.Port))
	}
	
	// Create app
	cliApp := cli.NewApp(version.GetVersion())

	switch command {
	case "tui":
		err := cliApp.TUI(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
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
	case "hibernate":
		err := cliApp.Hibernate(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "resume":
		err := cliApp.Resume(args)
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
	case "apply":
		err := cliApp.Apply(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "diff":
		err := cliApp.Diff(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "layers":
		err := cliApp.Layers(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "rollback":
		err := cliApp.Rollback(args)
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
	case "save":
		// Route save command to ami save for now
		saveArgs := append([]string{"save"}, args...)
		err := cliApp.AMI(saveArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "migrate":
		err := cliApp.Migrate(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "profiles":
		err := cliApp.Profiles(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "idle":
		err := cliApp.Idle(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "project":
		err := cliApp.Project(args)
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
	fmt.Println("  cws [options] <command> [arguments]")
	
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --version        Show version information")
	fmt.Println("  --help           Show this help information")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  tui                                 Launch terminal UI")
	fmt.Println("  launch <template> <name> [options]  Launch a new workstation")
	fmt.Println("    --project <name>                  Associate with project")  
	fmt.Println("    --size <XS|S|M|L|XL>              Instance size")
	fmt.Println("    --spot                            Use spot instances")
	fmt.Println("    --volume <name>                   Attach EFS volume")
	fmt.Println("    --storage <size>                  Attach EBS storage")
	fmt.Println("  list [--project <name>]             List workstations (optionally by project)")
	fmt.Println("  connect <name>                      Get connection info for workstation")
	fmt.Println("  stop <name>                         Stop a workstation")
	fmt.Println("  start <name>                        Start a stopped workstation")
	fmt.Println("  delete <name>                       Delete a workstation")
	fmt.Println("  hibernate <name>                    Hibernate a workstation (preserves RAM)")
	fmt.Println("  resume <name>                       Resume a hibernated workstation")
	fmt.Println("  save <instance> <template> [opts]   Save customized instance as reusable template")
	fmt.Println("  templates [action] [args]           Template repository operations")
	fmt.Println("    list                              List available templates")
	fmt.Println("    search <query>                    Search templates across repositories")
	fmt.Println("    info <template>                   Show detailed template information")
	fmt.Println("    featured                          Show featured community templates")
	fmt.Println("    discover                          Discover templates by research category")
	fmt.Println("    install <repo:template>           Install template from repository")
	fmt.Println()
	fmt.Println("  apply <template> <instance>         Apply template to running instance")
	fmt.Println("  diff <template> <instance>          Show template differences")
	fmt.Println("  layers <instance>                   List applied template layers")  
	fmt.Println("  rollback <instance>                 Rollback template applications")
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
	fmt.Println("    save <instance> <template> [opts] Save running instance as AMI template")
	fmt.Println()
	fmt.Println("  project <action> [args]             Manage research projects")
	fmt.Println("    create <name> [options]           Create new project")
	fmt.Println("    list                              List all projects")
	fmt.Println("    info <name>                       Show project details")
	fmt.Println("    budget <name> [options]           Manage project budgets")
	fmt.Println("    instances <name>                  List project instances")
	fmt.Println("    templates <name>                  List project templates")
	fmt.Println("    members <name> [action]           Manage project members")
	fmt.Println("    delete <name>                     Delete project")
	fmt.Println()
	fmt.Println("  idle <action> [args]                Manage idle detection and hibernation policies")
	fmt.Println("    status [instance]                 Show idle detection status")
	fmt.Println("    enable                            Enable idle detection")
	fmt.Println("    disable                           Disable idle detection")
	fmt.Println("    profile list                      List idle detection profiles")
	fmt.Println("    profile create <name> [options]   Create new idle profile")
	fmt.Println("    instance <name> [options]         Set instance-specific idle settings")
	fmt.Println("    history [instance]                Show idle action history")
	fmt.Println()
	fmt.Println("  version                             Show version")
	fmt.Println("  help                                Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cws tui                                      # Launch terminal UI")
	fmt.Println("  cws launch r-research my-analysis           # Launch R environment")
	fmt.Println("  cws launch python-research ml-project --size L  # Launch Python with large instance")
	fmt.Println("  cws list                                    # List all workstations")
	fmt.Println("  cws connect my-analysis                     # Get connection details")
	fmt.Println("  cws hibernate my-analysis                   # Hibernate instance (preserve RAM)")
	fmt.Println("  cws resume my-analysis                      # Resume hibernated instance")
	fmt.Println("  cws apply python-ml my-analysis             # Add ML tools to existing instance")
	fmt.Println("  cws diff python-ml my-analysis              # Preview template changes")
	fmt.Println("  cws layers my-analysis                      # Show applied template history")
	fmt.Println("  cws rollback my-analysis                    # Undo last template application")
	fmt.Println("  cws volume create shared-data               # Create shared EFS volume")
	fmt.Println("  cws storage create fast-storage XL io2     # Create high-performance storage")
	fmt.Println("  cws ami build neuroimaging                  # Build AMI from template")
	fmt.Println("  cws save my-analysis custom-ml-env          # Save customized instance as template")
	fmt.Println("  cws project create brain-study --budget 1000  # Create project with budget")
	fmt.Println("  cws project members brain-study add user@university.edu admin  # Add team member")
	fmt.Println("  cws launch python-ml analysis --project brain-study  # Launch in project")
	fmt.Println()
	fmt.Println("Template sizes: XS, S, M, L, XL, GPU-S, GPU-M, GPU-L")
	fmt.Println("Storage sizes: XS (100GB), S (500GB), M (1TB), L (2TB), XL (4TB)")
}
