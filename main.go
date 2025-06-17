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
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
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
	AttachedVolumes []string  `json:"attached_volumes"`  // EFS volume names
	AttachedEBSVolumes []string `json:"attached_ebs_volumes"` // EBS volume IDs
}

// EFSVolume represents a persistent EFS file system
type EFSVolume struct {
	Name            string    `json:"name"`             // User-friendly name
	FileSystemId    string    `json:"filesystem_id"`    // AWS EFS ID  
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	MountTargets    []string  `json:"mount_targets"`    // Mount target IDs
	State           string    `json:"state"`            // available, creating, deleting
	PerformanceMode string    `json:"performance_mode"` // generalPurpose, maxIO
	ThroughputMode  string    `json:"throughput_mode"`  // bursting, provisioned
	EstimatedCostGB float64   `json:"estimated_cost_gb"` // $/GB/month
	SizeBytes       int64     `json:"size_bytes"`       // Current size
}

// EBSVolume represents a secondary EBS volume for high-performance storage
type EBSVolume struct {
	Name           string    `json:"name"`           // User-friendly name
	VolumeID       string    `json:"volume_id"`      // AWS EBS volume ID
	Region         string    `json:"region"`
	CreationTime   time.Time `json:"creation_time"`
	State          string    `json:"state"`          // available, creating, in-use, deleting
	VolumeType     string    `json:"volume_type"`    // gp3, io2, etc.
	SizeGB         int32     `json:"size_gb"`        // Volume size in GB
	IOPS           int32     `json:"iops"`           // Provisioned IOPS (for io2)
	Throughput     int32     `json:"throughput"`     // Throughput in MB/s (for gp3)
	EstimatedCostGB float64  `json:"estimated_cost_gb"` // $/GB/month
	AttachedTo     string    `json:"attached_to"`    // Instance name if attached
}

// State manages the application state
type State struct {
	Instances  map[string]Instance  `json:"instances"`
	Volumes    map[string]EFSVolume `json:"volumes"`
	EBSVolumes map[string]EBSVolume `json:"ebs_volumes"`
	Config     Config               `json:"config"`
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
var efsClient *efs.Client

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
	case "volume":
		handleVolume()
	case "storage":
		handleStorage()
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
	efsClient = efs.NewFromConfig(cfg)
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
	fmt.Println("  cws volume <command>            Manage EFS volumes (create, list, info, delete)")
	fmt.Println("  cws storage <command>           Manage EBS volumes (create, list, info, delete, attach, detach)")
	fmt.Println("  cws arch                        Show detected architecture")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cws launch r-research my-analysis")
	fmt.Println("  cws launch python-research ml-project --dry-run")
	fmt.Println("  cws launch r-research data-analysis --volume research-data")
	fmt.Println("  cws volume create research-data")
	fmt.Println("  cws connect my-analysis")
	fmt.Println("  cws list")
	fmt.Println("  cws config profile research")
	fmt.Println("  cws config region us-east-1")
}

func handleLaunch() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws launch <template> <name> [options]")
		fmt.Println("Options:")
		fmt.Println("  --dry-run              Validate configuration without launching instance")
		fmt.Println("  --volume <volume-name> Attach EFS volume to instance")
		fmt.Println("  --storage <size>       Create EBS volume (XS=100GB, S=500GB, M=1TB, L=2TB, XL=4TB)")
		fmt.Println("  --storage-type <type>  EBS volume type (gp3, io2) - default: gp3")
		handleTemplates()
		return
	}

	templateName := os.Args[2]
	instanceName := os.Args[3]
	
	// Parse additional flags
	dryRun := false
	volumeName := ""
	storageSize := ""
	storageType := "gp3" // Default to gp3
	
	for i := 4; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--dry-run":
			dryRun = true
		case "--volume":
			if i+1 < len(os.Args) {
				volumeName = os.Args[i+1]
				i++ // Skip the next argument as it's the volume name
			} else {
				fmt.Println("❌ --volume flag requires a volume name")
				return
			}
		case "--storage":
			if i+1 < len(os.Args) {
				storageSize = os.Args[i+1]
				i++ // Skip the next argument as it's the storage size
			} else {
				fmt.Println("❌ --storage flag requires a size (XS, S, M, L, XL)")
				return
			}
		case "--storage-type":
			if i+1 < len(os.Args) {
				storageType = os.Args[i+1]
				if storageType != "gp3" && storageType != "io2" {
					fmt.Println("❌ Storage type must be gp3 or io2")
					return
				}
				i++ // Skip the next argument as it's the storage type
			} else {
				fmt.Println("❌ --storage-type flag requires a type (gp3, io2)")
				return
			}
		}
	}

	template, exists := templates[templateName]
	if !exists {
		fmt.Printf("❌ Template '%s' not found\n", templateName)
		handleTemplates()
		return
	}

	// Validate volume if specified
	state := loadState()
	var volume *EFSVolume
	if volumeName != "" {
		vol, exists := state.Volumes[volumeName]
		if !exists {
			fmt.Printf("❌ Volume '%s' not found\n", volumeName)
			fmt.Println("💡 Use 'cws volume list' to see available volumes")
			return
		}
		if vol.State != "available" {
			fmt.Printf("❌ Volume '%s' is not available (current state: %s)\n", volumeName, vol.State)
			return
		}
		volume = &vol
	}

	// Detect local architecture and get appropriate template values
	arch := getLocalArchitecture()
	region := getCurrentRegion()
	if region == "" {
		fmt.Printf("❌ No AWS region configured\n")
		fmt.Println("💡 Set a region with: cws config region <region>")
		return
	}
	
	// Validate region matches volume region if volume is specified
	if volume != nil && volume.Region != region {
		fmt.Printf("❌ Volume '%s' is in region %s, but you're launching in region %s\n", volumeName, volume.Region, region)
		fmt.Println("💡 Change region or use a volume in the same region")
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
	if volume != nil {
		fmt.Printf("   EFS Volume: %s (%s)\n", volume.Name, volume.FileSystemId)
	}
	if storageSize != "" {
		sizeGB, iops, throughput, cost := parseStorageConfiguration(storageSize, storageType)
		fmt.Printf("   EBS Storage: %s (%dGB %s, ~$%.2f/month)\n", storageSize, sizeGB, storageType, cost)
		if storageType == "io2" {
			fmt.Printf("   Provisioned IOPS: %d\n", iops)
		}
		if storageType == "gp3" && throughput > 125 {
			fmt.Printf("   Provisioned Throughput: %d MB/s\n", throughput)
		}
	}
	
	if dryRun {
		fmt.Println()
		fmt.Println("✅ Dry run complete! Configuration validated successfully.")
		fmt.Printf("💡 To actually launch: cws launch %s %s\n", templateName, instanceName)
		return
	}
	fmt.Println()

	// Prepare UserData with EFS mounting if volume is specified
	userData := template.UserData
	if volume != nil {
		userData = addEFSMountToUserData(userData, volume.FileSystemId, region)
	}

	// Launch EC2 instance
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		UserData:     aws.String(base64.StdEncoding.EncodeToString([]byte(userData))),
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
	attachedVolumes := []string{}
	if volume != nil {
		attachedVolumes = []string{volume.Name}
	}
	
	attachedEBSVolumes := []string{}
	// TODO: Add EBS volume creation during launch if storageSize is specified
	
	instance := Instance{
		ID:                 instanceID,
		Name:               instanceName,
		Template:           templateName,
		PublicIP:           publicIP,
		PrivateIP:          privateIP,
		State:              "running",
		LaunchTime:         time.Now(),
		EstimatedDailyCost: costPerHour * 24,
		AttachedVolumes:    attachedVolumes,
		AttachedEBSVolumes: attachedEBSVolumes,
	}

	state.Instances[instanceName] = instance
	saveState(state)

	fmt.Printf("✅ Workstation '%s' launched successfully!\n", instanceName)
	fmt.Printf("   Instance ID: %s\n", instanceID)
	fmt.Printf("   Public IP: %s\n", publicIP)
	fmt.Printf("   Template: %s (%s)\n", template.Name, arch)
	fmt.Printf("   Instance Type: %s\n", instanceType)
	fmt.Printf("   Estimated cost: $%.2f/day\n", instance.EstimatedDailyCost)
	if volume != nil {
		fmt.Printf("   EFS Volume: %s mounted at /efs\n", volume.Name)
	}
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
		Instances:  make(map[string]Instance),
		Volumes:    make(map[string]EFSVolume),
		EBSVolumes: make(map[string]EBSVolume),
		Config:     Config{},
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
			Instances:  make(map[string]Instance),
			Volumes:    make(map[string]EFSVolume),
			EBSVolumes: make(map[string]EBSVolume),
			Config:     Config{},
		}
	}
	
	// Initialize maps if they don't exist (for backward compatibility)
	if state.Instances == nil {
		state.Instances = make(map[string]Instance)
	}
	if state.Volumes == nil {
		state.Volumes = make(map[string]EFSVolume)
	}
	if state.EBSVolumes == nil {
		state.EBSVolumes = make(map[string]EBSVolume)
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

func handleVolume() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws volume <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  create <name>    Create a new EFS volume")
		fmt.Println("  list             List all EFS volumes")
		fmt.Println("  info <name>      Show detailed volume information")
		fmt.Println("  delete <name>    Delete EFS volume (with confirmation)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  cws volume create research-data")
		fmt.Println("  cws volume list")
		fmt.Println("  cws volume info research-data")
		fmt.Println("  cws volume delete research-data")
		return
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "create":
		handleVolumeCreate()
	case "list":
		handleVolumeList()
	case "info":
		handleVolumeInfo()
	case "delete":
		handleVolumeDelete()
	default:
		fmt.Printf("❌ Unknown volume command: %s\n", subcommand)
		fmt.Println("Use 'cws volume' to see available commands")
	}
}

func handleVolumeCreate() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws volume create <name>")
		fmt.Println("Example: cws volume create research-data")
		return
	}

	volumeName := os.Args[3]
	region := getCurrentRegion()
	if region == "" {
		fmt.Println("❌ No AWS region configured")
		fmt.Println("💡 Set a region with: cws config region <region>")
		return
	}

	// Check if volume name already exists
	state := loadState()
	if _, exists := state.Volumes[volumeName]; exists {
		fmt.Printf("❌ Volume '%s' already exists\n", volumeName)
		return
	}

	fmt.Printf("🗄️  Creating EFS volume '%s' in %s...\n", volumeName, region)

	// Create EFS file system
	createInput := &efs.CreateFileSystemInput{
		CreationToken:   aws.String(fmt.Sprintf("cws-%s-%d", volumeName, time.Now().Unix())),
		PerformanceMode: efsTypes.PerformanceModeGeneralPurpose, // Default to general purpose
		ThroughputMode:  efsTypes.ThroughputModeBursting,        // Default to bursting
		Tags: []efsTypes.Tag{
			{Key: aws.String("Name"), Value: aws.String(volumeName)},
			{Key: aws.String("CreatedBy"), Value: aws.String("cloudworkstation")},
		},
	}

	result, err := efsClient.CreateFileSystem(context.TODO(), createInput)
	if err != nil {
		fmt.Printf("❌ Failed to create EFS volume: %v\n", err)
		return
	}

	fileSystemId := *result.FileSystemId
	fmt.Printf("🎯 EFS File System ID: %s\n", fileSystemId)

	// Wait for file system to become available
	fmt.Println("⏳ Waiting for EFS volume to become available...")
	startTime := time.Now()
	
	for i := 0; i < 60; i++ { // Wait up to 10 minutes
		elapsed := time.Since(startTime).Round(time.Second)
		
		describeInput := &efs.DescribeFileSystemsInput{
			FileSystemId: aws.String(fileSystemId),
		}
		
		describeResult, err := efsClient.DescribeFileSystems(context.TODO(), describeInput)
		if err != nil {
			fmt.Printf("   [%s] ⚠️  Error checking status: %v\n", elapsed, err)
			time.Sleep(10 * time.Second)
			continue
		}
		
		if len(describeResult.FileSystems) > 0 {
			fs := describeResult.FileSystems[0]
			currentState := string(fs.LifeCycleState)
			
			switch currentState {
			case "creating":
				fmt.Printf("   [%s] 🟡 Creating EFS volume...\n", elapsed)
			case "available":
				fmt.Printf("   [%s] 🟢 EFS volume is available!\n", elapsed)
				
				// Save volume to state
				volume := EFSVolume{
					Name:            volumeName,
					FileSystemId:    fileSystemId,
					Region:          region,
					CreationTime:    time.Now(),
					MountTargets:    []string{}, // Will be populated when mount targets are created
					State:           currentState,
					PerformanceMode: string(fs.PerformanceMode),
					ThroughputMode:  string(fs.ThroughputMode),
					EstimatedCostGB: 0.30, // $0.30/GB/month for standard storage
					SizeBytes:       0,    // Will be updated when data is stored
				}
				
				state.Volumes[volumeName] = volume
				saveState(state)
				
				fmt.Printf("✅ EFS volume '%s' created successfully!\n", volumeName)
				fmt.Printf("📋 Volume Details:\n")
				fmt.Printf("   Name: %s\n", volume.Name)
				fmt.Printf("   File System ID: %s\n", volume.FileSystemId)
				fmt.Printf("   Region: %s\n", volume.Region)
				fmt.Printf("   Performance Mode: %s\n", volume.PerformanceMode)
				fmt.Printf("   Throughput Mode: %s\n", volume.ThroughputMode)
				fmt.Printf("   Estimated Cost: $%.2f/GB/month\n", volume.EstimatedCostGB)
				fmt.Println()
				fmt.Printf("💡 To use this volume, specify it when launching: cws launch <template> <name> --volume %s\n", volumeName)
				return
			case "deleting":
				fmt.Printf("❌ Volume is being deleted\n")
				return
			default:
				fmt.Printf("   [%s] 🔄 State: %s\n", elapsed, currentState)
			}
		}
		
		time.Sleep(10 * time.Second)
	}
	
	fmt.Println("⚠️  Timeout waiting for EFS volume to become available")
	fmt.Printf("💡 Check AWS console for file system %s status\n", fileSystemId)
}

func handleVolumeList() {
	state := loadState()
	
	if len(state.Volumes) == 0 {
		fmt.Println("📭 No EFS volumes found")
		fmt.Println("💡 Create one with: cws volume create <name>")
		return
	}

	fmt.Printf("📋 EFS Volumes (%d total):\n\n", len(state.Volumes))
	fmt.Printf("%-20s %-25s %-12s %-15s %-12s\n", "NAME", "FILE SYSTEM ID", "STATE", "REGION", "SIZE")
	fmt.Printf("%-20s %-25s %-12s %-15s %-12s\n", "----", "--------------", "-----", "------", "----")
	
	for _, volume := range state.Volumes {
		sizeStr := "0 GB"
		if volume.SizeBytes > 0 {
			sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
			sizeStr = fmt.Sprintf("%.2f GB", sizeGB)
		}
		
		fmt.Printf("%-20s %-25s %-12s %-15s %-12s\n", 
			volume.Name, 
			volume.FileSystemId, 
			volume.State, 
			volume.Region,
			sizeStr,
		)
	}
	
	fmt.Println()
	fmt.Println("💡 Use 'cws volume info <name>' for detailed information")
}

func handleVolumeInfo() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws volume info <name>")
		fmt.Println("Example: cws volume info research-data")
		return
	}

	volumeName := os.Args[3]
	state := loadState()
	
	volume, exists := state.Volumes[volumeName]
	if !exists {
		fmt.Printf("❌ Volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws volume list' to see available volumes")
		return
	}

	fmt.Printf("📋 EFS Volume Information: %s\n\n", volume.Name)
	fmt.Printf("Basic Details:\n")
	fmt.Printf("  Name: %s\n", volume.Name)
	fmt.Printf("  File System ID: %s\n", volume.FileSystemId)
	fmt.Printf("  Region: %s\n", volume.Region)
	fmt.Printf("  State: %s\n", volume.State)
	fmt.Printf("  Created: %s\n", volume.CreationTime.Format("2006-01-02 15:04:05 MST"))
	
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Performance Mode: %s\n", volume.PerformanceMode)
	fmt.Printf("  Throughput Mode: %s\n", volume.ThroughputMode)
	
	fmt.Printf("\nStorage:\n")
	if volume.SizeBytes > 0 {
		sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
		fmt.Printf("  Current Size: %.2f GB\n", sizeGB)
		monthlyCost := sizeGB * volume.EstimatedCostGB
		fmt.Printf("  Estimated Monthly Cost: $%.2f\n", monthlyCost)
	} else {
		fmt.Printf("  Current Size: 0 GB (no data stored yet)\n")
		fmt.Printf("  Base Cost: $%.2f/GB/month\n", volume.EstimatedCostGB)
	}
	
	fmt.Printf("\nMount Targets:\n")
	if len(volume.MountTargets) > 0 {
		for i, mtId := range volume.MountTargets {
			fmt.Printf("  %d. %s\n", i+1, mtId)
		}
	} else {
		fmt.Printf("  None created yet\n")
		fmt.Printf("  💡 Mount targets will be created automatically when attaching to instances\n")
	}
}

func handleVolumeDelete() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws volume delete <name>")
		fmt.Println("Example: cws volume delete research-data")
		return
	}

	volumeName := os.Args[3]
	state := loadState()
	
	volume, exists := state.Volumes[volumeName]
	if !exists {
		fmt.Printf("❌ Volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws volume list' to see available volumes")
		return
	}

	// Check if volume is attached to any instances
	attachedInstances := []string{}
	for instanceName, instance := range state.Instances {
		for _, attachedVolume := range instance.AttachedVolumes {
			if attachedVolume == volumeName {
				attachedInstances = append(attachedInstances, instanceName)
			}
		}
	}
	
	if len(attachedInstances) > 0 {
		fmt.Printf("❌ Cannot delete volume '%s' - it is attached to the following instances:\n", volumeName)
		for _, instanceName := range attachedInstances {
			fmt.Printf("   - %s\n", instanceName)
		}
		fmt.Println("💡 Detach the volume from all instances before deleting")
		return
	}

	fmt.Printf("⚠️  WARNING: This will permanently delete EFS volume '%s'!\n", volumeName)
	fmt.Printf("   File System ID: %s\n", volume.FileSystemId)
	fmt.Printf("   Region: %s\n", volume.Region)
	if volume.SizeBytes > 0 {
		sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
		fmt.Printf("   Current Size: %.2f GB\n", sizeGB)
		fmt.Printf("   ⚠️  ALL DATA WILL BE LOST!\n")
	}
	fmt.Println()
	fmt.Print("Type 'yes' to confirm deletion: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "yes" {
		fmt.Println("❌ Deletion cancelled")
		return
	}

	fmt.Printf("🗑️  Deleting EFS volume '%s'...\n", volumeName)

	// First, delete all mount targets
	if len(volume.MountTargets) > 0 {
		fmt.Println("⏳ Removing mount targets...")
		for _, mtId := range volume.MountTargets {
			deleteMtInput := &efs.DeleteMountTargetInput{
				MountTargetId: aws.String(mtId),
			}
			
			_, err := efsClient.DeleteMountTarget(context.TODO(), deleteMtInput)
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete mount target %s: %v\n", mtId, err)
			} else {
				fmt.Printf("   ✅ Deleted mount target: %s\n", mtId)
			}
		}
		
		// Wait for mount targets to be deleted
		fmt.Println("⏳ Waiting for mount targets to be removed...")
		time.Sleep(10 * time.Second)
	}

	// Delete the file system
	deleteInput := &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(volume.FileSystemId),
	}
	
	_, err := efsClient.DeleteFileSystem(context.TODO(), deleteInput)
	if err != nil {
		fmt.Printf("❌ Failed to delete EFS volume: %v\n", err)
		return
	}

	// Remove from state
	delete(state.Volumes, volumeName)
	saveState(state)

	fmt.Printf("✅ EFS volume '%s' deletion initiated\n", volumeName)
	fmt.Println("💡 The volume will be fully deleted within a few minutes")
}

// addEFSMountToUserData modifies the UserData script to include EFS mounting commands
func addEFSMountToUserData(originalUserData, fileSystemId, region string) string {
	// EFS mounting commands to append to the user data
	efsMountCommands := fmt.Sprintf(`

# CloudWorkstation EFS Volume Mount
echo "Setting up EFS volume mounting..." >> /var/log/cws-setup.log

# Install EFS utilities
apt-get update -y
apt-get install -y amazon-efs-utils

# Create mount point
mkdir -p /efs
mkdir -p /home/ubuntu/efs

# Mount EFS file system
echo "%s.efs.%s.amazonaws.com:/ /efs efs defaults,_netdev" >> /etc/fstab
mount -a

# Give ubuntu user access to EFS
chown ubuntu:ubuntu /efs
chmod 755 /efs

# Create symlink in ubuntu home directory for easy access
ln -sf /efs /home/ubuntu/efs
chown -h ubuntu:ubuntu /home/ubuntu/efs

echo "EFS volume mounted successfully at /efs" >> /var/log/cws-setup.log
`, fileSystemId, region)

	return originalUserData + efsMountCommands
}

// parseStorageConfiguration converts t-shirt sizes to EBS specifications
func parseStorageConfiguration(size, volumeType string) (sizeGB int32, iops int32, throughput int32, monthlyCost float64) {
	switch size {
	case "XS":
		sizeGB = 100
		iops = 3000   // Default for gp3, minimum for io2
		throughput = 125 // Default for gp3
		if volumeType == "gp3" {
			monthlyCost = 100 * 0.08  // $0.08/GB/month for gp3
		} else {
			monthlyCost = 100*0.125 + 3000*0.065  // $0.125/GB/month + $0.065/IOPS/month for io2
		}
	case "S":
		sizeGB = 500
		iops = 5000
		throughput = 250
		if volumeType == "gp3" {
			monthlyCost = 500 * 0.08
		} else {
			monthlyCost = 500*0.125 + 5000*0.065
		}
	case "M":
		sizeGB = 1000
		iops = 10000
		throughput = 500
		if volumeType == "gp3" {
			monthlyCost = 1000*0.08 + (500-125)*0.04  // Base + extra throughput cost
		} else {
			monthlyCost = 1000*0.125 + 10000*0.065
		}
	case "L":
		sizeGB = 2000
		iops = 20000
		throughput = 1000
		if volumeType == "gp3" {
			monthlyCost = 2000*0.08 + (1000-125)*0.04
		} else {
			monthlyCost = 2000*0.125 + 20000*0.065
		}
	case "XL":
		sizeGB = 4000
		iops = 32000  // Max for most instance types
		throughput = 1000
		if volumeType == "gp3" {
			monthlyCost = 4000*0.08 + (1000-125)*0.04
		} else {
			monthlyCost = 4000*0.125 + 32000*0.065
		}
	default:
		// Default to M if invalid size provided
		sizeGB = 1000
		iops = 10000
		throughput = 500
		if volumeType == "gp3" {
			monthlyCost = 1000*0.08 + (500-125)*0.04
		} else {
			monthlyCost = 1000*0.125 + 10000*0.065
		}
	}
	
	return sizeGB, iops, throughput, monthlyCost
}

// handleStorage manages EBS volume operations
func handleStorage() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: cws storage <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  create <name> <size> [type]  Create EBS volume (size: XS,S,M,L,XL; type: gp3,io2)")
		fmt.Println("  list                         List all EBS volumes")
		fmt.Println("  info <name>                  Show detailed volume information")
		fmt.Println("  attach <volume> <instance>   Attach volume to instance")
		fmt.Println("  detach <volume>              Detach volume from instance")
		fmt.Println("  delete <name>                Delete EBS volume (with confirmation)")
		fmt.Println()
		fmt.Println("Storage Sizes:")
		fmt.Println("  XS = 100GB   (~$8/month)")
		fmt.Println("  S  = 500GB   (~$40/month)")
		fmt.Println("  M  = 1TB     (~$95/month)")
		fmt.Println("  L  = 2TB     (~$195/month)")
		fmt.Println("  XL = 4TB     (~$355/month)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  cws storage create ml-data L io2")
		fmt.Println("  cws storage list")
		fmt.Println("  cws storage attach ml-data my-instance")
		fmt.Println("  cws storage detach ml-data")
		return
	}

	subcommand := os.Args[2]
	switch subcommand {
	case "create":
		handleStorageCreate()
	case "list":
		handleStorageList()
	case "info":
		handleStorageInfo()
	case "attach":
		handleStorageAttach()
	case "detach":
		handleStorageDetach()
	case "delete":
		handleStorageDelete()
	default:
		fmt.Printf("❌ Unknown storage command: %s\n", subcommand)
		fmt.Println("Use 'cws storage' to see available commands")
	}
}

func handleStorageCreate() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: cws storage create <name> <size> [type]")
		fmt.Println("  name: Volume name")
		fmt.Println("  size: XS, S, M, L, XL")
		fmt.Println("  type: gp3 (default), io2")
		fmt.Println("Example: cws storage create ml-data L io2")
		return
	}

	volumeName := os.Args[3]
	sizeStr := os.Args[4]
	volumeType := "gp3"
	if len(os.Args) > 5 {
		volumeType = os.Args[5]
		if volumeType != "gp3" && volumeType != "io2" {
			fmt.Println("❌ Volume type must be gp3 or io2")
			return
		}
	}

	region := getCurrentRegion()
	if region == "" {
		fmt.Println("❌ No AWS region configured")
		fmt.Println("💡 Set a region with: cws config region <region>")
		return
	}

	// Check if volume name already exists
	state := loadState()
	if _, exists := state.EBSVolumes[volumeName]; exists {
		fmt.Printf("❌ EBS volume '%s' already exists\n", volumeName)
		return
	}

	// Parse storage configuration
	sizeGB, iops, throughput, monthlyCost := parseStorageConfiguration(sizeStr, volumeType)
	
	fmt.Printf("💾 Creating EBS volume '%s' (%s)...\n", volumeName, sizeStr)
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Size: %dGB\n", sizeGB)
	fmt.Printf("   Type: %s\n", volumeType)
	if volumeType == "io2" {
		fmt.Printf("   Provisioned IOPS: %d\n", iops)
	}
	if volumeType == "gp3" && throughput > 125 {
		fmt.Printf("   Throughput: %d MB/s\n", throughput)
	}
	fmt.Printf("   Estimated Cost: $%.2f/month\n", monthlyCost)
	fmt.Println()

	// Create EBS volume
	var createInput *ec2.CreateVolumeInput
	
	if volumeType == "gp3" {
		createInput = &ec2.CreateVolumeInput{
			Size:               aws.Int32(sizeGB),
			VolumeType:         types.VolumeTypeGp3,
			AvailabilityZone:   aws.String(region + "a"), // Use first AZ
			Iops:               aws.Int32(iops),
			Throughput:         aws.Int32(throughput),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeVolume,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(volumeName)},
						{Key: aws.String("CreatedBy"), Value: aws.String("cloudworkstation")},
						{Key: aws.String("Size"), Value: aws.String(sizeStr)},
					},
				},
			},
		}
	} else { // io2
		createInput = &ec2.CreateVolumeInput{
			Size:               aws.Int32(sizeGB),
			VolumeType:         types.VolumeTypeIo2,
			AvailabilityZone:   aws.String(region + "a"), // Use first AZ
			Iops:               aws.Int32(iops),
			TagSpecifications: []types.TagSpecification{
				{
					ResourceType: types.ResourceTypeVolume,
					Tags: []types.Tag{
						{Key: aws.String("Name"), Value: aws.String(volumeName)},
						{Key: aws.String("CreatedBy"), Value: aws.String("cloudworkstation")},
						{Key: aws.String("Size"), Value: aws.String(sizeStr)},
					},
				},
			},
		}
	}

	result, err := ec2Client.CreateVolume(context.TODO(), createInput)
	if err != nil {
		fmt.Printf("❌ Failed to create EBS volume: %v\n", err)
		return
	}

	volumeID := *result.VolumeId
	fmt.Printf("🎯 EBS Volume ID: %s\n", volumeID)

	// Wait for volume to become available
	fmt.Println("⏳ Waiting for EBS volume to become available...")
	startTime := time.Now()
	
	for i := 0; i < 30; i++ { // Wait up to 5 minutes
		elapsed := time.Since(startTime).Round(time.Second)
		
		describeInput := &ec2.DescribeVolumesInput{
			VolumeIds: []string{volumeID},
		}
		
		describeResult, err := ec2Client.DescribeVolumes(context.TODO(), describeInput)
		if err != nil {
			fmt.Printf("   [%s] ⚠️  Error checking status: %v\n", elapsed, err)
			time.Sleep(10 * time.Second)
			continue
		}
		
		if len(describeResult.Volumes) > 0 {
			volume := describeResult.Volumes[0]
			currentState := string(volume.State)
			
			switch currentState {
			case "creating":
				fmt.Printf("   [%s] 🟡 Creating EBS volume...\n", elapsed)
			case "available":
				fmt.Printf("   [%s] 🟢 EBS volume is available!\n", elapsed)
				
				// Save volume to state
				ebsVolume := EBSVolume{
					Name:           volumeName,
					VolumeID:       volumeID,
					Region:         region,
					CreationTime:   time.Now(),
					State:          currentState,
					VolumeType:     volumeType,
					SizeGB:         sizeGB,
					IOPS:           iops,
					Throughput:     throughput,
					EstimatedCostGB: monthlyCost / float64(sizeGB),
					AttachedTo:     "",
				}
				
				state.EBSVolumes[volumeName] = ebsVolume
				saveState(state)
				
				fmt.Printf("✅ EBS volume '%s' created successfully!\n", volumeName)
				fmt.Printf("💡 To attach: cws storage attach %s <instance-name>\n", volumeName)
				return
			case "error":
				fmt.Printf("❌ Volume creation failed\n")
				return
			default:
				fmt.Printf("   [%s] 🔄 State: %s\n", elapsed, currentState)
			}
		}
		
		time.Sleep(10 * time.Second)
	}
	
	fmt.Println("⚠️  Timeout waiting for EBS volume to become available")
	fmt.Printf("💡 Check AWS console for volume %s status\n", volumeID)
}

func handleStorageList() {
	state := loadState()
	
	if len(state.EBSVolumes) == 0 {
		fmt.Println("📭 No EBS volumes found")
		fmt.Println("💡 Create one with: cws storage create <name> <size> [type]")
		return
	}

	fmt.Printf("📋 EBS Volumes (%d total):\n\n", len(state.EBSVolumes))
	fmt.Printf("%-20s %-15s %-8s %-10s %-15s %-12s\n", "NAME", "VOLUME ID", "SIZE", "TYPE", "STATE", "ATTACHED TO")
	fmt.Printf("%-20s %-15s %-8s %-10s %-15s %-12s\n", "----", "---------", "----", "----", "-----", "-----------")
	
	for _, volume := range state.EBSVolumes {
		attachedTo := volume.AttachedTo
		if attachedTo == "" {
			attachedTo = "-"
		}
		
		fmt.Printf("%-20s %-15s %-8dGB %-10s %-15s %-12s\n", 
			volume.Name, 
			volume.VolumeID, 
			volume.SizeGB,
			volume.VolumeType,
			volume.State, 
			attachedTo,
		)
	}
	
	fmt.Println()
	fmt.Println("💡 Use 'cws storage info <name>' for detailed information")
}

func handleStorageInfo() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws storage info <name>")
		fmt.Println("Example: cws storage info ml-data")
		return
	}

	volumeName := os.Args[3]
	state := loadState()
	
	volume, exists := state.EBSVolumes[volumeName]
	if !exists {
		fmt.Printf("❌ EBS volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws storage list' to see available volumes")
		return
	}

	fmt.Printf("📋 EBS Volume Information: %s\n\n", volume.Name)
	fmt.Printf("Basic Details:\n")
	fmt.Printf("  Name: %s\n", volume.Name)
	fmt.Printf("  Volume ID: %s\n", volume.VolumeID)
	fmt.Printf("  Region: %s\n", volume.Region)
	fmt.Printf("  State: %s\n", volume.State)
	fmt.Printf("  Created: %s\n", volume.CreationTime.Format("2006-01-02 15:04:05 MST"))
	
	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Size: %dGB\n", volume.SizeGB)
	fmt.Printf("  Volume Type: %s\n", volume.VolumeType)
	if volume.IOPS > 0 {
		fmt.Printf("  IOPS: %d\n", volume.IOPS)
	}
	if volume.Throughput > 0 && volume.VolumeType == "gp3" {
		fmt.Printf("  Throughput: %d MB/s\n", volume.Throughput)
	}
	
	fmt.Printf("\nCost:\n")
	monthlyCost := float64(volume.SizeGB) * volume.EstimatedCostGB
	fmt.Printf("  Estimated Monthly Cost: $%.2f\n", monthlyCost)
	
	fmt.Printf("\nAttachment:\n")
	if volume.AttachedTo != "" {
		fmt.Printf("  Attached to: %s\n", volume.AttachedTo)
		fmt.Printf("  Mount point: /data (automatic)\n")
	} else {
		fmt.Printf("  Not attached\n")
		fmt.Printf("  💡 To attach: cws storage attach %s <instance-name>\n", volume.Name)
	}
}

func handleStorageAttach() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: cws storage attach <volume-name> <instance-name>")
		fmt.Println("Example: cws storage attach ml-data my-instance")
		return
	}

	volumeName := os.Args[3]
	instanceName := os.Args[4]
	
	state := loadState()
	
	// Check if volume exists
	volume, volumeExists := state.EBSVolumes[volumeName]
	if !volumeExists {
		fmt.Printf("❌ EBS volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws storage list' to see available volumes")
		return
	}
	
	// Check if instance exists
	instance, instanceExists := state.Instances[instanceName]
	if !instanceExists {
		fmt.Printf("❌ Instance '%s' not found\n", instanceName)
		fmt.Println("💡 Use 'cws list' to see available instances")
		return
	}
	
	// Check if volume is already attached
	if volume.AttachedTo != "" {
		fmt.Printf("❌ Volume '%s' is already attached to instance '%s'\n", volumeName, volume.AttachedTo)
		return
	}
	
	// Check if instance is running
	if instance.State != "running" {
		fmt.Printf("❌ Instance '%s' is not running (current state: %s)\n", instanceName, instance.State)
		fmt.Println("💡 Start the instance first with: cws start " + instanceName)
		return
	}

	fmt.Printf("🔗 Attaching EBS volume '%s' to instance '%s'...\n", volumeName, instanceName)

	// Find next available device name
	deviceName := "/dev/sdf" // Start with /dev/sdf for secondary volumes
	
	// Attach volume
	attachInput := &ec2.AttachVolumeInput{
		VolumeId:   aws.String(volume.VolumeID),
		InstanceId: aws.String(instance.ID),
		Device:     aws.String(deviceName),
	}
	
	_, err := ec2Client.AttachVolume(context.TODO(), attachInput)
	if err != nil {
		fmt.Printf("❌ Failed to attach volume: %v\n", err)
		return
	}

	// Update state
	volume.AttachedTo = instanceName
	state.EBSVolumes[volumeName] = volume
	
	// Add to instance's attached volumes list
	instance.AttachedEBSVolumes = append(instance.AttachedEBSVolumes, volume.VolumeID)
	state.Instances[instanceName] = instance
	
	saveState(state)

	fmt.Printf("✅ Volume '%s' attached successfully!\n", volumeName)
	fmt.Printf("💡 The volume will be automatically mounted at /data on next boot\n")
	fmt.Printf("💡 To manually mount: sudo mount %s /data\n", deviceName)
}

func handleStorageDetach() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws storage detach <volume-name>")
		fmt.Println("Example: cws storage detach ml-data")
		return
	}

	volumeName := os.Args[3]
	state := loadState()
	
	volume, exists := state.EBSVolumes[volumeName]
	if !exists {
		fmt.Printf("❌ EBS volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws storage list' to see available volumes")
		return
	}
	
	if volume.AttachedTo == "" {
		fmt.Printf("❌ Volume '%s' is not attached to any instance\n", volumeName)
		return
	}

	fmt.Printf("🔌 Detaching EBS volume '%s' from instance '%s'...\n", volumeName, volume.AttachedTo)

	// Detach volume
	detachInput := &ec2.DetachVolumeInput{
		VolumeId: aws.String(volume.VolumeID),
	}
	
	_, err := ec2Client.DetachVolume(context.TODO(), detachInput)
	if err != nil {
		fmt.Printf("❌ Failed to detach volume: %v\n", err)
		return
	}

	// Update state
	instanceName := volume.AttachedTo
	volume.AttachedTo = ""
	state.EBSVolumes[volumeName] = volume
	
	// Remove from instance's attached volumes list
	if instance, exists := state.Instances[instanceName]; exists {
		updatedEBSVolumes := []string{}
		for _, volID := range instance.AttachedEBSVolumes {
			if volID != volume.VolumeID {
				updatedEBSVolumes = append(updatedEBSVolumes, volID)
			}
		}
		instance.AttachedEBSVolumes = updatedEBSVolumes
		state.Instances[instanceName] = instance
	}
	
	saveState(state)

	fmt.Printf("✅ Volume '%s' detached successfully!\n", volumeName)
}

func handleStorageDelete() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws storage delete <volume-name>")
		fmt.Println("Example: cws storage delete ml-data")
		return
	}

	volumeName := os.Args[3]
	state := loadState()
	
	volume, exists := state.EBSVolumes[volumeName]
	if !exists {
		fmt.Printf("❌ EBS volume '%s' not found\n", volumeName)
		fmt.Println("💡 Use 'cws storage list' to see available volumes")
		return
	}
	
	// Check if volume is attached
	if volume.AttachedTo != "" {
		fmt.Printf("❌ Cannot delete volume '%s' - it is attached to instance '%s'\n", volumeName, volume.AttachedTo)
		fmt.Printf("💡 Detach first with: cws storage detach %s\n", volumeName)
		return
	}

	fmt.Printf("⚠️  WARNING: This will permanently delete EBS volume '%s'!\n", volumeName)
	fmt.Printf("   Volume ID: %s\n", volume.VolumeID)
	fmt.Printf("   Size: %dGB (%s)\n", volume.SizeGB, volume.VolumeType)
	fmt.Printf("   ⚠️  ALL DATA WILL BE LOST!\n")
	fmt.Println()
	fmt.Print("Type 'yes' to confirm deletion: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "yes" {
		fmt.Println("❌ Deletion cancelled")
		return
	}

	fmt.Printf("🗑️  Deleting EBS volume '%s'...\n", volumeName)

	// Delete volume
	deleteInput := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(volume.VolumeID),
	}
	
	_, err := ec2Client.DeleteVolume(context.TODO(), deleteInput)
	if err != nil {
		fmt.Printf("❌ Failed to delete volume: %v\n", err)
		return
	}

	// Remove from state
	delete(state.EBSVolumes, volumeName)
	saveState(state)

	fmt.Printf("✅ EBS volume '%s' deleted successfully!\n", volumeName)
}
