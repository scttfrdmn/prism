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
	"flag"
	"fmt"
	"os"

	"github.com/scttfrdmn/cloudworkstation/internal/cli"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

func main() {
	// Define flags
	versionFlag := flag.Bool("version", false, "Show version information")
	helpFlag := flag.Bool("help", false, "Show help information")

	// Parse flags but keep command and arguments separate
	_ = flag.CommandLine.Parse(os.Args[1:])

	// Handle version and help flags
	if *versionFlag {
		fmt.Println(version.GetCLIVersionInfo())
		return
	}

	if *helpFlag {
		printUsage()
		return
	}

	// Create app
	cliApp := cli.NewApp(version.GetVersion())

	// Use the new Cobra-based system for all commands
	err := cliApp.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", cli.FormatErrorForCLI(err, "command execution"))
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
	fmt.Println("    --subnet <subnet-id>              Specify subnet for launch")
	fmt.Println("    --vpc <vpc-id>                    Specify VPC for launch")
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
	fmt.Println("  pricing <action> [args]             Manage institutional pricing discounts")
	fmt.Println("    show                              Show current pricing configuration")
	fmt.Println("    install <config-file>             Install institutional pricing config")
	fmt.Println("    validate [config-file]            Validate pricing configuration")
	fmt.Println("    example [filename]                Create example pricing config")
	fmt.Println("    calculate <type> <price> [region] Calculate discounted pricing")
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
	fmt.Println("  profiles <action> [args]            Manage CloudWorkstation profiles")
	fmt.Println("    list                              List all profiles")
	fmt.Println("    current                           Show current active profile")
	fmt.Println("    add <name> <display> [options]    Add new profile")
	fmt.Println("      --aws-profile <name>            AWS CLI profile to use")
	fmt.Println("      --region <region>               AWS region")
	fmt.Println("    switch <name>                     Switch to profile")
	fmt.Println("    remove <name>                     Remove profile")
	fmt.Println("    export                            Export profiles to file")
	fmt.Println("    import <file>                     Import profiles from file")
	fmt.Println()
	fmt.Println("  security <action> [args]            Security management and compliance")
	fmt.Println("    status                            Show security configuration")
	fmt.Println("    health                            Perform security health check")
	fmt.Println("    compliance <framework>            Compliance operations")
	fmt.Println("      validate <framework>            Validate against compliance framework")
	fmt.Println("      report <framework>              Generate compliance report")
	fmt.Println("      scp <framework>                 Generate Service Control Policies")
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
	fmt.Println("  cws pricing show                            # Show institutional pricing config")
	fmt.Println("  cws pricing install university_pricing.json  # Install institutional discounts")
	fmt.Println("  cws pricing calculate c5.large 0.096 us-west-2  # Calculate discounted pricing")
	fmt.Println("  cws profiles add personal research --aws-profile aws --region us-west-2  # Add profile")
	fmt.Println("  cws profiles switch personal                # Switch to profile")
	fmt.Println("  cws security health                         # Check security status")
	fmt.Println("  cws idle profile list                       # Show hibernation policies")
	fmt.Println()
	fmt.Println("T-shirt sizes (compute + storage):")
	fmt.Println("  XS: 1 vCPU, 2GB RAM + 100GB     S: 2 vCPU, 4GB RAM + 500GB")
	fmt.Println("  M:  2 vCPU, 8GB RAM + 1TB       L: 4 vCPU, 16GB RAM + 2TB")
	fmt.Println("  XL: 8 vCPU, 32GB RAM + 4TB")
	fmt.Println()
	fmt.Println("Smart scaling: GPU workloads → g4dn/g5g family, Memory → r5/r6g, Compute → c5/c6g")
}
