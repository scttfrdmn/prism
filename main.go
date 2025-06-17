package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Template defines a cloud workstation template
type Template struct {
	Name         string
	Description  string
	AMI          string
	InstanceType string
	UserData     string
	Ports        []int
	EstimatedCostPerHour float64
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
}

// Hard-coded templates for MVP
var templates = map[string]Template{
	"r-research": {
		Name:         "R Research Environment",
		Description:  "R + RStudio Server + tidyverse packages",
		AMI:          "ami-0c02fb55956c7d316", // Ubuntu 22.04 LTS (will be replaced with custom AMI)
		InstanceType: "t3.medium",
		UserData: `#!/bin/bash
apt update -y
apt install -y r-base r-base-dev
# Install RStudio Server
wget https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2023.06.1-524-amd64.deb
dpkg -i rstudio-server-2023.06.1-524-amd64.deb || true
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
		Ports:                []int{22, 8787},
		EstimatedCostPerHour: 0.0464,
	},
	"python-research": {
		Name:         "Python Research Environment", 
		Description:  "Python + Jupyter + data science packages",
		AMI:          "ami-0c02fb55956c7d316", // Ubuntu 22.04 LTS
		InstanceType: "t3.medium",
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
		Ports:                []int{22, 8888},
		EstimatedCostPerHour: 0.0464,
	},
	"basic-ubuntu": {
		Name:         "Basic Ubuntu",
		Description:  "Plain Ubuntu 22.04 for general use",
		AMI:          "ami-0c02fb55956c7d316", // Ubuntu 22.04 LTS
		InstanceType: "t3.small",
		UserData: `#!/bin/bash
apt update -y
apt install -y curl wget git vim
echo "Setup complete" > /var/log/cws-setup.log
`,
		Ports:                []int{22},
		EstimatedCostPerHour: 0.0232,
	},
}

var ec2Client *ec2.Client

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	// Initialize AWS client
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("❌ Failed to load AWS config: %v\n", err)
		fmt.Println("💡 Make sure AWS CLI is configured: aws configure")
		os.Exit(1)
	}
	ec2Client = ec2.NewFromConfig(cfg)

	command := os.Args[1]
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
	case "templates":
		handleTemplates()
	default:
		fmt.Printf("❌ Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Cloud Workstation Platform - Launch research environments in seconds")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cws launch <template> <name>    Launch a new workstation")
	fmt.Println("  cws list                        List all workstations")
	fmt.Println("  cws connect <name>              Connect to workstation")
	fmt.Println("  cws stop <name>                 Stop workstation")
	fmt.Println("  cws start <name>                Start stopped workstation") 
	fmt.Println("  cws delete <name>               Delete workstation")
	fmt.Println("  cws templates                   List available templates")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cws launch r-research my-analysis")
	fmt.Println("  cws launch python-research ml-project")
	fmt.Println("  cws connect my-analysis")
	fmt.Println("  cws list")
}

func handleLaunch() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cws launch <template> <name>")
		handleTemplates()
		return
	}

	templateName := os.Args[2]
	instanceName := os.Args[3]

	template, exists := templates[templateName]
	if !exists {
		fmt.Printf("❌ Template '%s' not found\n", templateName)
		handleTemplates()
		return
	}

	fmt.Printf("🚀 Launching %s workstation '%s'...\n", template.Name, instanceName)

	// Check if instance name already exists
	state := loadState()
	if _, exists := state.Instances[instanceName]; exists {
		fmt.Printf("❌ Instance '%s' already exists\n", instanceName)
		return
	}

	// Launch EC2 instance
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(template.AMI),
		InstanceType: types.InstanceType(template.InstanceType),
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
	
	// Wait for instance to get IP address
	fmt.Print("⏳ Waiting for instance to start")
	var publicIP string
	for i := 0; i < 30; i++ { // Wait up to 5 minutes
		time.Sleep(10 * time.Second)
		fmt.Print(".")
		
		describeInput := &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}
		
		describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)
		if err != nil {
			continue
		}
		
		if len(describeResult.Reservations) > 0 && len(describeResult.Reservations[0].Instances) > 0 {
			instance := describeResult.Reservations[0].Instances[0]
			if instance.PublicIpAddress != nil {
				publicIP = *instance.PublicIpAddress
				break
			}
		}
	}
	fmt.Println()

	if publicIP == "" {
		fmt.Println("⚠️  Instance launched but no public IP yet. Check AWS console.")
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
		EstimatedDailyCost: template.EstimatedCostPerHour * 24,
	}

	state.Instances[instanceName] = instance
	saveState(state)

	fmt.Printf("✅ Workstation '%s' launched successfully!\n", instanceName)
	fmt.Printf("   Instance ID: %s\n", instanceID)
	fmt.Printf("   Public IP: %s\n", publicIP)
	fmt.Printf("   Template: %s\n", template.Name)
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
	
	for name, template := range templates {
		fmt.Printf("🔧 %s\n", name)
		fmt.Printf("   %s\n", template.Description)
		fmt.Printf("   Instance: %s ($%.4f/hour)\n", template.InstanceType, template.EstimatedCostPerHour)
		fmt.Printf("   Ports: %v\n", template.Ports)
		fmt.Println()
	}
	
	fmt.Println("Usage: cws launch <template> <name>")
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
		return State{Instances: make(map[string]Instance)}
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
