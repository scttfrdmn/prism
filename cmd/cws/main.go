// CloudWorkstation CLI client (cws) - Launch research computing environments.
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
	// Create app
	cliApp := cli.NewApp(version.GetVersion())

	// Use the Cobra-based system for all commands
	err := cliApp.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", cli.FormatErrorForCLI(err, "command execution"))
		os.Exit(1)
	}
}
