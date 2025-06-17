package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Template defines a cloud workstation template
type Template struct {
	Name         string
	Description  string
	AMI          map[string]map[string]string // region -> arch -> AMI ID
	InstanceType map[string]string            // arch -> instance type
	UserData     string
	Ports        []int
	EstimatedCostPerHour map[string]float64 // arch -> cost per hour
}

// Config manages application configuration
type Config struct {
	DefaultProfile string `json:"default_profile"`
	DefaultRegion  string `json:"default_region"`
}

// Instance represents a running cloud workstation
type Instance struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Template       string    `json:"template"`
	PublicIP       string    `json:"public_ip"`
	PrivateIP      string    `json:"private_ip"`
	State          string    `json:"state"`
	LaunchTime     time.Time `json:"launch_time"`
	EstimatedDailyCost float64 `json:"estimated_daily_cost"`
}

// State manages the application state
type State struct {
	Instances map[string]Instance `json:"instances"`
	Config    Config              `json:"config"`
}

// Hard-coded templates for MVP
var templates = map[string]Template{
	"r-research": {
		Name:        "R Research Environment",
		Description: "R + RStudio Server + tidyverse packages",
		AMI: map[string]map[string]string{
			"us-east-1": {
				"x86_64": "ami-02029c87fa31fb148",
				"arm64":  "ami-050499786ebf55a6a",
			},
			"us-east-2": {
				"x86_64": "ami-0b05d988257befbbe",
				"arm64":  "ami-010755a3881216bba",
			},
			"us-west-1": {
				"x86_64": "ami-043b59f1d11f8f189",
				"arm64":  "ami-0d3e8bea392f79ebb",
			},
			"us-west-2": {
				"x86_64": "ami-016d360a89daa11ba",
				"arm64":  "ami-09f6c9efbf93542be",
			},
		},
		InstanceType: map[string]string{
			"x86_64": "t3.medium",
			"arm64":  "t4g.medium", // Graviton2 ARM-based
		},
		UserData: `#!/bin/bash
apt update -y
apt install -y r-base r-base-dev

# Detect architecture and install appropriate RStudio Server
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    wget https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2023.06.1-524-amd64.deb
    dpkg -i rstudio-server-2023.06.1-524-amd64.deb || true
elif [ "$ARCH" = "aarch64" ]; then
    wget https://download2.rstudio.org/server/jammy/arm64/rstudio-server-2023.06.1-524-arm64.deb
    dpkg -i rstudio-server-2023.06.1-524-arm64.deb || true
fi
apt-get install -f -y

# Install common R packages
R -e "install.packages(c('tidyverse','ggplot2','dplyr','readr'), repos='http://cran.rstudio.com/')"
# Configure RStudio
echo "www-port=8787" >> /etc/rstudio/rserver.conf
systemctl restart rstudio-server
# Create ubuntu user for RStudio
echo "ubuntu:password123" | chpasswd
echo "Setup complete" > /var/log/cws-setup.log
`,
		Ports: []int{22, 8787},
		EstimatedCostPerHour: map[string]float64{
			"x86_64": 0.0464,
			"arm64":  0.0368, // Graviton2 is typically 20% cheaper
		},
	},
	"python-research": {
		Name:        "Python Research Environment",
		Description: "Python + Jupyter + data science packages",
		AMI: map[string]map[string]string{
			"us-east-1": {
				"x86_64": "ami-02029c87fa31fb148",
				"arm64":  "ami-050499786ebf55a6a",
			},
			"us-east-2": {
				"x86_64": "ami-0b05d988257befbbe",
				"arm64":  "ami-010755a3881216bba",
			},
			"us-west-1": {
				"x86_64": "ami-043b59f1d11f8f189",
				"arm64":  "ami-0d3e8bea392f79ebb",
			},
			"us-west-2": {
				"x86_64": "ami-016d360a89daa11ba",
				"arm64":  "ami-09f6c9efbf93542be",
			},
		},
		InstanceType: map[string]string{
			"x86_64": "t3.medium",
			"arm64":  "t4g.medium",
		},
		UserData: `#!/bin/bash
apt update -y
apt install -y python3 python3-pip
pip3 install jupyter pandas numpy matplotlib seaborn scikit-learn
# Configure Jupyter
mkdir -p /home/ubuntu/.jupyter
cat > /home/ubuntu/.jupyter/jupyter_notebook_config.py << 'JUPYTER_EOF'
c.NotebookApp.ip = '0.0.0.0'
c.NotebookApp.port = 8888
c.NotebookApp.open_browser = False
c.NotebookApp.token = ''
c.NotebookApp.password = ''
JUPYTER_EOF
chown -R ubuntu:ubuntu /home/ubuntu/.jupyter
# Start Jupyter as service
sudo -u ubuntu nohup jupyter notebook --config=/home/ubuntu/.jupyter/jupyter_notebook_config.py > /var/log/jupyter.log 2>&1 &
echo "Setup complete" > /var/log/cws-setup.log
`,
		Ports: []int{22, 8888},
		EstimatedCostPerHour: map[string]float64{
			"x86_64": 0.0464,
			"arm64":  0.0368,
		},
	},
	"basic-ubuntu": {
		Name:        "Basic Ubuntu",
		Description: "Plain Ubuntu 22.04 for general use",
		AMI: map[string]map[string]string{
			"us-east-1": {
				"x86_64": "ami-02029c87fa31fb148",
				"arm64":  "ami-050499786ebf55a6a",
			},
			"us-east-2": {
				"x86_64": "ami-0b05d988257befbbe",
				"arm64":  "ami-010755a3881216bba",
			},
			"us-west-1": {
				"x86_64": "ami-043b59f1d11f8f189",
				"arm64":  "ami-0d3e8bea392f79ebb",
			},
			"us-west-2": {
				"x86_64": "ami-016d360a89daa11ba",
				"arm64":  "ami-09f6c9efbf93542be",
			},
		},
		InstanceType: map[string]string{
			"x86_64": "t3.small",
			"arm64":  "t4g.small",
		},
		UserData: `#!/bin/bash
apt update -y
apt install -y curl wget git vim
echo "Setup complete" > /var/log/cws-setup.log
`,
		Ports: []int{22},
		EstimatedCostPerHour: map[string]float64{
			"x86_64": 0.0232,
			"arm64":  0.0184,
		},
	},
}

var ec2Client *ec2.Client

// getLocalArchitecture detects the local system architecture and maps it to AWS instance architectures
func getLocalArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		// Default to x86_64 for other architectures
		return "x86_64"
	}
}

// getTemplateForArchitecture returns template values for the detected architecture and region
func getTemplateForArchitecture(template Template, arch, region string) (string, string, float64, error) {
	regionAMIs, regionExists := template.AMI[region]
	if !regionExists {
		return "", "", 0, fmt.Errorf("template does not support region %s", region)
	}
	
	ami, archExists := regionAMIs[arch]
	if !archExists {
		return "", "", 0, fmt.Errorf("template does not support architecture %s in region %s", arch, region)
	}
	
	instanceType, exists := template.InstanceType[arch]
	if !exists {
		return "", "", 0, fmt.Errorf("template does not have instance type for architecture %s", arch)
	}
	
	cost, exists := template.EstimatedCostPerHour[arch]
	if !exists {
		return "", "", 0, fmt.Errorf("template does not have cost info for architecture %s", arch)
	}
	
	return ami, instanceType, cost, nil
}

// getCurrentRegion returns the currently configured AWS region
func getCurrentRegion() string {
	state := loadState()
	
	// Check environment variables first
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	
	// Fall back to application config
	if region == "" && state.Config.DefaultRegion != "" {
		region = state.Config.DefaultRegion
	}
	
	return region
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	
	// Commands that don't need AWS client
	switch command {
	case "config":
		handleConfig()
		return
	case "arch":
		handleArch()
		return
	case "templates":
		handleTemplates()
		return
	}

	// Initialize AWS client for commands that need it
	initializeAWSClient()

	switch command {
	case "launch":
		handleLaunch()
	case "list":
		handleList()
	case "connect":
		handleConnect()
	case "stop":
		handleStop()
	case "start":
		handleStart()
	case "delete":
		handleDelete()
	case "test":
		handleTest()
	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func initializeAWSClient() {
	state := loadState()
	
	var configOptions []func(*config.LoadOptions) error
	
	// Check for AWS profile from environment or state
	profileName := os.Getenv("AWS_PROFILE")
	if profileName == "" && state.Config.DefaultProfile != "" {
		profileName = state.Config.DefaultProfile
	}
	
	if profileName != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(profileName))
		fmt.Printf("🔧 Using AWS profile: %s\n", profileName)
	}
	
	// Check for AWS region from environment or state
	regionName := os.Getenv("AWS_REGION")
	if regionName == "" {
		regionName = os.Getenv("AWS_DEFAULT_REGION")
	}
	if regionName == "" && state.Config.DefaultRegion != "" {
		regionName = state.Config.DefaultRegion
	}
	
	if regionName != "" {
		configOptions = append(configOptions, config.WithRegion(regionName))
		fmt.Printf("🌍 Using AWS region: %s\n", regionName)
	}
	
	cfg, err := config.LoadDefaultConfig(context.TODO(), configOptions...)
	if err != nil {
		fmt.Printf("❌ Failed to load AWS config: %v\n", err)
		if profileName != "" {
			fmt.Printf("💡 Check if profile '%s' exists: aws configure list-profiles\n", profileName)
		} else {
			fmt.Println("💡 Make sure AWS CLI is configured: aws configure")
		}
		if regionName == "" {
			fmt.Println("💡 Set a region: cws config region us-east-1")
		}
		os.Exit(1)
	}
	
	// Validate that we have a region
	if cfg.Region == "" {
		fmt.Println("❌ No AWS region configured")
		fmt.Println("💡 Set a region with: cws config region <region>")
		fmt.Println("💡 Or set AWS_REGION environment variable")
		fmt.Println("💡 Or configure it in your AWS profile")
		os.Exit(1)
	}
	
	ec2Client = ec2.NewFromConfig(cfg)
}

func printUsage() {
	fmt.Println("Cloud Workstation Platform - Launch research environments in seconds")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cws launch <template> <name> [--dry-run]  Launch a new workstation")
	fmt.Println("  cws list                                  List all workstations")
	fmt.Println("  cws connect <name>              Connect to workstation")
	fmt.Println("  cws stop <name>                 Stop workstation")
	fmt.Println("  cws start <name>                Start stopped workstation") 
	fmt.Println("  cws delete <name>               Delete workstation")
	fmt.Println("  cws templates                   List available templates")
	fmt.Println("  cws config profile <name>       Set default AWS profile")
	fmt.Println("  cws config region <region>      Set default AWS region")
	fmt.Println("  cws config show                 Show current configuration")
	fmt.Println("  cws test                        Test AWS connectivity and permissions")
	fmt.Println("  cws arch                        Show detected architecture")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cws launch r-research my-analysis")
	fmt.Println("  cws launch python-research ml-project --dry-run")
	fmt.Println("  cws connect my-analysis")
	fmt.Println("  cws list")
	fmt.Println("  cws config profile research")
	fmt.Println("  cws config region us-east-1")
}

func handleLaunch() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws launch <template> <name> [--dry-run]")
		fmt.Println("  --dry-run    Validate configuration without launching instance")
		handleTemplates()
		return
	}

	templateName := os.Args[2]
	instanceName := os.Args[3]
	
	// Check for dry-run flag
	dryRun := false
	if len(os.Args) > 4 && os.Args[4] == "--dry-run" {
		dryRun = true
	}

	template, exists := templates[templateName]
	if !exists {
		fmt.Printf("❌ Template '%s' not found\n", templateName)
		handleTemplates()
		return
	}

	// Detect local architecture and get appropriate template values
	arch := getLocalArchitecture()
	region := getCurrentRegion()
	if region == "" {
		fmt.Printf("❌ No AWS region configured\n")
		fmt.Println("💡 Set a region with: cws config region <region>")
		return
	}
	
	ami, instanceType, costPerHour, err := getTemplateForArchitecture(template, arch, region)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		if strings.Contains(err.Error(), "does not support region") {
			fmt.Printf("💡 Supported regions: us-east-1, us-east-2, us-west-1, us-west-2\n")
		}
		return
	}

	if dryRun {
		fmt.Printf("🧪 DRY RUN: Validating %s workstation '%s' (%s)...\n", template.Name, instanceName, arch)
	} else {
		fmt.Printf("🚀 Launching %s workstation '%s' (%s)...\n", template.Name, instanceName, arch)
	}

	// Check if instance name already exists
	state := loadState()
	if _, exists := state.Instances[instanceName]; exists {
		fmt.Printf("❌ Instance '%s' already exists\n", instanceName)
		return
	}

	// Show configuration that will be used
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Template: %s\n", template.Name)
	fmt.Printf("   Instance Name: %s\n", instanceName)
	fmt.Printf("   Region: %s\n", region)
	fmt.Printf("   Architecture: %s\n", arch)
	fmt.Printf("   AMI: %s\n", ami)
	fmt.Printf("   Instance Type: %s\n", instanceType)
	fmt.Printf("   Estimated Cost: $%.2f/day\n", costPerHour*24)
	fmt.Printf("   Ports: %v\n", template.Ports)
	
	if dryRun {
		fmt.Println()
		fmt.Println("✅ Dry run complete! Configuration validated successfully.")
		fmt.Printf("💡 To actually launch: cws launch %s %s\n", templateName, instanceName)
		return
	}
	fmt.Println()

	// Launch EC2 instance
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		UserData:     aws.String(base64.StdEncoding.EncodeToString([]byte(template.UserData))),
		SecurityGroups: []string{"default"},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(instanceName)},
					{Key: aws.String("CreatedBy"), Value: aws.String("cloudworkstation")},
					{Key: aws.String("Template"), Value: aws.String(templateName)},
				},
			},
		},
	}

	result, err := ec2Client.RunInstances(context.TODO(), input)
	if err != nil {
		fmt.Printf("❌ Failed to launch instance: %v\n", err)
		return
	}

	instanceID := *result.Instances[0].InstanceId
	fmt.Printf("🎯 Instance ID: %s\n", instanceID)
	
	// Wait for instance to start with detailed progress
	fmt.Println("⏳ Monitoring instance startup progress...")
	var publicIP string
	var privateIP string
	var lastState string
	startTime := time.Now()
	
	for i := 0; i < 36; i++ { // Wait up to 6 minutes (36 * 10 seconds)
		elapsed := time.Since(startTime).Round(time.Second)
		
		describeInput := &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}
		
		describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)
		if err != nil {
			fmt.Printf("   [%s] ⚠️  Error checking status: %v\n", elapsed, err)
			time.Sleep(10 * time.Second)
			continue
		}
		
		if len(describeResult.Reservations) > 0 && len(describeResult.Reservations[0].Instances) > 0 {
			instance := describeResult.Reservations[0].Instances[0]
			currentState := string(instance.State.Name)
			
			// Show state changes
			if currentState != lastState {
				switch currentState {
				case "pending":
					fmt.Printf("   [%s] 🟡 Instance starting up...\n", elapsed)
				case "running":
					fmt.Printf("   [%s] 🟢 Instance is running!\n", elapsed)
				case "stopping":
					fmt.Printf("   [%s] 🟠 Instance stopping...\n", elapsed)
				case "stopped":
					fmt.Printf("   [%s] 🔴 Instance stopped\n", elapsed)
				default:
					fmt.Printf("   [%s] 📊 State: %s\n", elapsed, currentState)
				}
				lastState = currentState
			}
			
			// Get IP addresses
			if instance.PublicIpAddress != nil {
				publicIP = *instance.PublicIpAddress
			}
			if instance.PrivateIpAddress != nil {
				privateIP = *instance.PrivateIpAddress
			}
			
			// Check if we have what we need
			if currentState == "running" && publicIP != "" {
				fmt.Printf("   [%s] 🌐 Public IP assigned: %s\n", elapsed, publicIP)
				if privateIP != "" {
					fmt.Printf("   [%s] 🏠 Private IP: %s\n", elapsed, privateIP)
				}
				break
			}
		}
		
		time.Sleep(10 * time.Second)
	}

	if publicIP == "" {
		fmt.Println("⚠️  Instance launched but no public IP assigned yet.")
		fmt.Println("   Check AWS console for status or try: cws list")
		publicIP = "pending"
	}

	// Save to state
	instance := Instance{
		ID:                 instanceID,
		Name:               instanceName,
		Template:           templateName,
		PublicIP:           publicIP,
		State:              "running",
		LaunchTime:         time.Now(),
		EstimatedDailyCost: costPerHour * 24,
	}

	state.Instances[instanceName] = instance
	saveState(state)

	fmt.Printf("✅ Workstation '%s' launched successfully!\n", instanceName)
	fmt.Printf("   Instance ID: %s\n", instanceID)
	fmt.Printf("   Public IP: %s\n", publicIP)
	fmt.Printf("   Template: %s (%s)\n", template.Name, arch)
	fmt.Printf("   Instance Type: %s\n", instanceType)
	fmt.Printf("   Estimated cost: $%.2f/day\n", instance.EstimatedDailyCost)
	fmt.Println()
	
	if templateName == "r-research" {
		fmt.Println("🔬 R Research Environment:")
		fmt.Printf("   RStudio Server: http://%s:8787\n", publicIP)
		fmt.Println("   Username: ubuntu")
		fmt.Println("   Password: password123")
	} else if templateName == "python-research" {
		fmt.Println("🐍 Python Research Environment:")
		fmt.Printf("   Jupyter Notebook: http://%s:8888\n", publicIP)
	}
	
	fmt.Printf("\n💻 Connect via SSH: ssh ubuntu@%s\n", publicIP)
	fmt.Println("   Or use: cws connect " + instanceName)
}

func handleList() {
	state := loadState()
	
	if len(state.Instances) == 0 {
		fmt.Println("📋 No workstations found.")
		fmt.Println("   Launch one with: cws launch <template> <name>")
		return
	}

	fmt.Println("📋 Your Cloud Workstations:")
	fmt.Println()
	
	totalDailyCost := 0.0
	for _, instance := range state.Instances {
		// Get current state from AWS
		currentState := getInstanceState(instance.ID)
		
		fmt.Printf("🖥️  %s\n", instance.Name)
		fmt.Printf("   Status: %s\n", currentState)
		fmt.Printf("   Template: %s\n", instance.Template)
		fmt.Printf("   Public IP: %s\n", instance.PublicIP)
		fmt.Printf("   Cost: $%.2f/day\n", instance.EstimatedDailyCost)
		fmt.Printf("   Launched: %s\n", instance.LaunchTime.Format("2006-01-02 15:04"))
		fmt.Println()
		
		if currentState == "running" {
			totalDailyCost += instance.EstimatedDailyCost
		}
	}
	
	fmt.Printf("💰 Total daily cost (running instances): $%.2f\n", totalDailyCost)
}

func handleConnect() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws connect <name>")
		return
	}

	instanceName := os.Args[2]
	state := loadState()
	
	instance, exists := state.Instances[instanceName]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", instanceName)
		return
	}

	currentState := getInstanceState(instance.ID)
	if currentState != "running" {
		fmt.Printf("❌ Instance '%s' is %s, not running\n", instanceName, currentState)
		fmt.Printf("   Start it with: cws start %s\n", instanceName)
		return
	}

	fmt.Printf("🔗 Connecting to %s (%s)...\n", instanceName, instance.PublicIP)
	fmt.Printf("💻 SSH: ssh ubuntu@%s\n", instance.PublicIP)
	
	if instance.Template == "r-research" {
		fmt.Printf("🔬 RStudio: http://%s:8787 (ubuntu/password123)\n", instance.PublicIP)
	} else if instance.Template == "python-research" {
		fmt.Printf("🐍 Jupyter: http://%s:8888\n", instance.PublicIP)
	}
}

func handleStop() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws stop <name>")
		return
	}

	instanceName := os.Args[2]
	state := loadState()
	
	instance, exists := state.Instances[instanceName]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", instanceName)
		return
	}

	fmt.Printf("⏹️  Stopping workstation '%s'...\n", instanceName)

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instance.ID},
	}

	_, err := ec2Client.StopInstances(context.TODO(), input)
	if err != nil {
		fmt.Printf("❌ Failed to stop instance: %v\n", err)
		return
	}

	fmt.Printf("✅ Workstation '%s' stopped successfully\n", instanceName)
	fmt.Println("💰 Instance stopped - no compute charges while stopped")
	fmt.Printf("   Start again with: cws start %s\n", instanceName)
}

func handleStart() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws start <name>")
		return
	}

	instanceName := os.Args[2]
	state := loadState()
	
	instance, exists := state.Instances[instanceName]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", instanceName)
		return
	}

	fmt.Printf("▶️  Starting workstation '%s'...\n", instanceName)

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{instance.ID},
	}

	_, err := ec2Client.StartInstances(context.TODO(), input)
	if err != nil {
		fmt.Printf("❌ Failed to start instance: %v\n", err)
		return
	}

	fmt.Printf("✅ Workstation '%s' starting...\n", instanceName)
	fmt.Println("⏳ It may take 1-2 minutes to be fully available")
	fmt.Printf("   Check status with: cws list\n")
}

func handleDelete() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws delete <name>")
		return
	}

	instanceName := os.Args[2]
	state := loadState()
	
	instance, exists := state.Instances[instanceName]
	if !exists {
		fmt.Printf("❌ Instance '%s' not found\n", instanceName)
		return
	}

	fmt.Printf("🗑️  Deleting workstation '%s'...\n", instanceName)
	fmt.Println("⚠️  This will permanently delete the instance and all data!")
	fmt.Print("   Type 'yes' to confirm: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("❌ Deletion cancelled")
		return
	}

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instance.ID},
	}

	_, err := ec2Client.TerminateInstances(context.TODO(), input)
	if err != nil {
		fmt.Printf("❌ Failed to delete instance: %v\n", err)
		return
	}

	// Remove from state
	delete(state.Instances, instanceName)
	saveState(state)

	fmt.Printf("✅ Workstation '%s' deleted successfully\n", instanceName)
}

func handleTemplates() {
	fmt.Println("📚 Available Templates:")
	fmt.Println()
	
	arch := getLocalArchitecture()
	fmt.Printf("🏗️  Detected architecture: %s\n", arch)
	fmt.Println()
	
	for name, template := range templates {
		fmt.Printf("🔧 %s\n", name)
		fmt.Printf("   %s\n", template.Description)
		
		// Show architecture-specific details
		if instanceType, exists := template.InstanceType[arch]; exists {
			if cost, exists := template.EstimatedCostPerHour[arch]; exists {
				fmt.Printf("   Instance: %s ($%.4f/hour)\n", instanceType, cost)
			}
		}
		
		// Show both architectures if different
		if len(template.InstanceType) > 1 {
			fmt.Printf("   Supported architectures:\n")
			for archType, instType := range template.InstanceType {
				if cost, exists := template.EstimatedCostPerHour[archType]; exists {
					fmt.Printf("     %s: %s ($%.4f/hour)\n", archType, instType, cost)
				}
			}
		}
		
		fmt.Printf("   Ports: %v\n", template.Ports)
		fmt.Println()
	}
	
	fmt.Println("Usage: cws launch <template> <name>")
}

func handleConfig() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:")
		fmt.Println("  cws config profile <name>     Set default AWS profile")
		fmt.Println("  cws config region <region>    Set default AWS region")
		fmt.Println("  cws config show               Show current configuration")
		return
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "profile":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cws config profile <name>")
			return
		}
		profileName := os.Args[3]
		
		state := loadState()
		state.Config.DefaultProfile = profileName
		saveState(state)
		
		fmt.Printf("✅ Default AWS profile set to: %s\n", profileName)
		
	case "region":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cws config region <region>")
			fmt.Println()
			fmt.Println("Common AWS regions:")
			fmt.Println("  us-east-1      US East (N. Virginia)")
			fmt.Println("  us-east-2      US East (Ohio)")
			fmt.Println("  us-west-1      US West (N. California)")
			fmt.Println("  us-west-2      US West (Oregon)")
			fmt.Println("  eu-west-1      Europe (Ireland)")
			fmt.Println("  eu-central-1   Europe (Frankfurt)")
			fmt.Println("  ap-southeast-1 Asia Pacific (Singapore)")
			fmt.Println("  ap-northeast-1 Asia Pacific (Tokyo)")
			return
		}
		regionName := os.Args[3]
		
		state := loadState()
		state.Config.DefaultRegion = regionName
		saveState(state)
		
		fmt.Printf("✅ Default AWS region set to: %s\n", regionName)
		
	case "show":
		state := loadState()
		fmt.Println("📋 Current Configuration:")
		if state.Config.DefaultProfile != "" {
			fmt.Printf("   Default AWS profile: %s\n", state.Config.DefaultProfile)
		} else {
			fmt.Println("   Default AWS profile: <not set>")
		}
		
		if state.Config.DefaultRegion != "" {
			fmt.Printf("   Default AWS region: %s\n", state.Config.DefaultRegion)
		} else {
			fmt.Println("   Default AWS region: <not set>")
		}
		
		// Show current effective profile
		profileName := os.Getenv("AWS_PROFILE")
		if profileName == "" && state.Config.DefaultProfile != "" {
			profileName = state.Config.DefaultProfile
		}
		
		if profileName != "" {
			fmt.Printf("   Active AWS profile: %s\n", profileName)
		} else {
			fmt.Println("   Active AWS profile: default")
		}
		
		// Show current effective region
		regionName := os.Getenv("AWS_REGION")
		if regionName == "" {
			regionName = os.Getenv("AWS_DEFAULT_REGION")
		}
		if regionName == "" && state.Config.DefaultRegion != "" {
			regionName = state.Config.DefaultRegion
		}
		
		if regionName != "" {
			fmt.Printf("   Active AWS region: %s\n", regionName)
		} else {
			fmt.Println("   Active AWS region: <not set - will use profile default>")
		}
		
	default:
		fmt.Printf("❌ Unknown config command: %s\n", subcommand)
		fmt.Println("Available commands: profile, region, show")
	}
}

func handleArch() {
	arch := getLocalArchitecture()
	fmt.Printf("🏗️  Detected local architecture: %s\n", arch)
	
	fmt.Println()
	fmt.Println("📋 Architecture mapping:")
	fmt.Println("   x86_64 (amd64) → AWS x86_64 instances (t3, m5, c5, etc.)")
	fmt.Println("   arm64          → AWS arm64 instances (t4g, m6g, c6g, etc.)")
	fmt.Println()
	fmt.Println("💰 ARM instances are typically 10-20% cheaper than x86 equivalents")
	fmt.Println("🚀 ARM instances often provide better price/performance for many workloads")
}

func handleTest() {
	fmt.Println("🧪 Testing AWS Configuration and Connectivity...")
	fmt.Println()
	
	// Test 1: Get current AWS configuration
	state := loadState()
	
	profileName := os.Getenv("AWS_PROFILE")
	if profileName == "" && state.Config.DefaultProfile != "" {
		profileName = state.Config.DefaultProfile
	}
	
	regionName := os.Getenv("AWS_REGION")
	if regionName == "" {
		regionName = os.Getenv("AWS_DEFAULT_REGION")
	}
	if regionName == "" && state.Config.DefaultRegion != "" {
		regionName = state.Config.DefaultRegion
	}
	
	fmt.Printf("🔧 Profile: %s\n", getOrDefault(profileName, "default"))
	fmt.Printf("🌍 Region: %s\n", getOrDefault(regionName, "not configured"))
	fmt.Println()
	
	// Test 2: Test STS (credentials and identity)
	fmt.Print("🔐 Testing AWS credentials... ")
	
	// Create STS client from same config as EC2 client
	state = loadState()
	var configOptions []func(*config.LoadOptions) error
	
	profileName = os.Getenv("AWS_PROFILE")
	if profileName == "" && state.Config.DefaultProfile != "" {
		profileName = state.Config.DefaultProfile
	}
	if profileName != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(profileName))
	}
	
	regionName = os.Getenv("AWS_REGION")
	if regionName == "" {
		regionName = os.Getenv("AWS_DEFAULT_REGION")
	}
	if regionName == "" && state.Config.DefaultRegion != "" {
		regionName = state.Config.DefaultRegion
	}
	if regionName != "" {
		configOptions = append(configOptions, config.WithRegion(regionName))
	}
	
	cfg, err := config.LoadDefaultConfig(context.TODO(), configOptions...)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		return
	}
	
	stsClient := sts.NewFromConfig(cfg)
	callerIdentity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		fmt.Println("   💡 Check your AWS credentials configuration")
		return
	}
	fmt.Printf("✅ SUCCESS\n")
	fmt.Printf("   Account: %s\n", aws.ToString(callerIdentity.Account))
	fmt.Printf("   User/Role: %s\n", aws.ToString(callerIdentity.Arn))
	fmt.Println()
	
	// Test 3: Test EC2 permissions
	fmt.Print("🖥️  Testing EC2 permissions... ")
	
	// Try to describe regions (basic read permission)
	regionsInput := &ec2.DescribeRegionsInput{}
	regionsResult, err := ec2Client.DescribeRegions(context.TODO(), regionsInput)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		fmt.Println("   💡 Your AWS credentials may not have EC2 permissions")
		return
	}
	fmt.Printf("✅ SUCCESS\n")
	fmt.Printf("   Available regions: %d\n", len(regionsResult.Regions))
	fmt.Println()
	
	// Test 4: Test current region connectivity
	fmt.Print("🌍 Testing current region connectivity... ")
	
	// Try to describe instances in current region
	instancesInput := &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(5),
	}
	_, err = ec2Client.DescribeInstances(context.TODO(), instancesInput)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		fmt.Println("   💡 Check if the configured region is correct and accessible")
		return
	}
	fmt.Printf("✅ SUCCESS\n")
	fmt.Println("   Region is accessible and responsive")
	fmt.Println()
	
	// Test 5: Test VPC and security groups
	fmt.Print("🔒 Testing VPC and security groups... ")
	
	sgInput := &ec2.DescribeSecurityGroupsInput{
		MaxResults: aws.Int32(5),
	}
	sgResult, err := ec2Client.DescribeSecurityGroups(context.TODO(), sgInput)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		return
	}
	fmt.Printf("✅ SUCCESS\n")
	fmt.Printf("   Security groups found: %d\n", len(sgResult.SecurityGroups))
	
	// Check for default security group
	hasDefault := false
	for _, sg := range sgResult.SecurityGroups {
		if aws.ToString(sg.GroupName) == "default" {
			hasDefault = true
			break
		}
	}
	if hasDefault {
		fmt.Println("   ✅ Default security group available")
	} else {
		fmt.Println("   ⚠️  Default security group not found")
	}
	fmt.Println()
	
	// Test 6: Test template AMI access
	fmt.Print("🗂️  Testing template AMI access... ")
	
	// Test the specific AMIs used in our templates
	arch := getLocalArchitecture()
	currentRegion := getCurrentRegion()
	
	template := templates["basic-ubuntu"]
	ami, _, _, err := getTemplateForArchitecture(template, arch, currentRegion)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		if strings.Contains(err.Error(), "does not support region") {
			fmt.Printf("   💡 Supported regions: us-east-1, us-east-2, us-west-1, us-west-2\n")
		}
		return
	}
	
	// Verify the AMI exists and is accessible
	amiInput := &ec2.DescribeImagesInput{
		ImageIds: []string{ami},
	}
	amiResult, err := ec2Client.DescribeImages(context.TODO(), amiInput)
	if err != nil {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Error: %v\n", err)
		return
	}
	if len(amiResult.Images) == 0 {
		fmt.Printf("❌ FAILED\n")
		fmt.Printf("   Template AMI %s not found in region %s\n", ami, currentRegion)
		return
	}
	
	fmt.Printf("✅ SUCCESS\n")
	fmt.Printf("   Template AMI accessible: %s\n", ami)
	fmt.Printf("   AMI Name: %s\n", aws.ToString(amiResult.Images[0].Name))
	fmt.Printf("   Architecture: %s\n", arch)
	fmt.Println()
	
	// Summary
	fmt.Println("🎉 All tests passed! Your AWS configuration is ready for CloudWorkstation.")
	fmt.Println()
	fmt.Println("💡 Next steps:")
	fmt.Println("   • Launch your first workstation: cws launch basic-ubuntu test")
	fmt.Println("   • View available templates: cws templates")
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getInstanceState(instanceID string) string {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	
	result, err := ec2Client.DescribeInstances(context.TODO(), input)
	if err != nil {
		return "unknown"
	}
	
	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		return string(result.Reservations[0].Instances[0].State.Name)
	}
	
	return "unknown"
}

func getStateFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	
	stateDir := filepath.Join(homeDir, ".cloudworkstation")
	os.MkdirAll(stateDir, 0755)
	
	return filepath.Join(stateDir, "state.json")
}

func loadState() State {
	stateFile := getStateFilePath()
	
	state := State{
		Instances: make(map[string]Instance),
		Config:    Config{},
	}
	
	data, err := os.ReadFile(stateFile)
	if err != nil {
		// File doesn't exist yet, return empty state
		return state
	}
	
	err = json.Unmarshal(data, &state)
	if err != nil {
		fmt.Printf("⚠️  Failed to parse state file: %v\n", err)
		fmt.Println("   Starting with empty state...")
		return State{
			Instances: make(map[string]Instance),
			Config:    Config{},
		}
	}
	
	// Initialize Config if it doesn't exist (for backward compatibility)
	if state.Instances == nil {
		state.Instances = make(map[string]Instance)
	}
	
	return state
}

func saveState(state State) {
	stateFile := getStateFilePath()
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to marshal state: %v\n", err)
		return
	}
	
	err = os.WriteFile(stateFile, data, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to save state: %v\n", err)
	}
}
