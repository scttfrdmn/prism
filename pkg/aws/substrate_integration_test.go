//go:build substrate
// +build substrate

package aws

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/prism/pkg/state"
	ctypes "github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/substrate/emulator"
)

// setupSubstrateVPC creates a VPC and public subnet in the Substrate server,
// returning their IDs. LaunchRequest can set VpcID/SubnetID directly to skip
// DiscoverDefaultVPC (which requires an isDefault=true VPC that only exists
// after RunInstances auto-creates it).
func setupSubstrateVPC(t *testing.T, cfg aws.Config) (vpcID, subnetID string) {
	t.Helper()
	ctx := context.Background()
	ec2Client := ec2.NewFromConfig(cfg)

	vpcResp, err := ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	})
	if err != nil {
		t.Fatalf("CreateVpc: %v", err)
	}
	vpcID = *vpcResp.Vpc.VpcId

	subnetResp, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		VpcId:     aws.String(vpcID),
		CidrBlock: aws.String("10.0.1.0/24"),
	})
	if err != nil {
		t.Fatalf("CreateSubnet: %v", err)
	}
	subnetID = *subnetResp.Subnet.SubnetId
	return vpcID, subnetID
}

// setupSubstrateManager starts an in-process Substrate server and returns a
// Manager wired to it. No Docker, no external processes — the server is
// automatically shut down when the test ends via t.Cleanup().
func setupSubstrateManager(t *testing.T) (*Manager, *emulator.TestServer) {
	t.Helper()

	// Isolated state directory so tests don't touch ~/.prism
	t.Setenv("PRISM_STATE_DIR", t.TempDir())

	ts := emulator.StartTestServer(t)

	cfg := aws.Config{
		Region: "us-east-1",
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               ts.URL,
					SigningRegion:     "us-east-1",
					HostnameImmutable: true,
				}, nil
			},
		),
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}

	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)

	stateMgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("state.NewManager: %v", err)
	}

	manager := &Manager{
		cfg:                 cfg,
		ec2:                 ec2Client,
		efs:                 efs.NewFromConfig(cfg),
		iam:                 iam.NewFromConfig(cfg),
		ssm:                 ssmClient,
		sts:                 sts.NewFromConfig(cfg),
		region:              "us-east-1",
		templates:           getTemplates(),
		discountConfig:      ctypes.DiscountConfig{},
		stateManager:        stateMgr,
		amiResolver:         NewUniversalAMIResolver(ec2Client),
		amiDiscovery:        NewAMIDiscovery(ssmClient),
		healthMonitor:       NewHealthMonitor(cfg, "us-east-1"),
		quotaManager:        NewQuotaManager(cfg, "us-east-1"),
		availabilityManager: NewAvailabilityManager(cfg, "us-east-1"),
		architectureCache:   make(map[string]string),
	}

	return manager, ts
}

func TestSubstrateLaunchInstance(t *testing.T) {
	manager, ts := setupSubstrateManager(t)

	// Seed AMI IDs that LaunchInstance resolves via SSM (substrate#267)
	ts.SeedSSMParameters(map[string]string{
		"/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id": "ami-ubuntu2404-amd64",
		"/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id": "ami-ubuntu2404-arm64",
		"/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp3/ami-id": "ami-ubuntu2204-amd64",
		"/aws/service/canonical/ubuntu/server/22.04/stable/current/arm64/hvm/ebs-gp3/ami-id": "ami-ubuntu2204-arm64",
		"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64":              "ami-al2023-x86",
		"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64":               "ami-al2023-arm64",
	})

	// Write a minimal template YAML to a temp dir so GetTemplateInfo can find it.
	// LaunchInstance calls the package-level templates.GetTemplateInfo which scans
	// filesystem dirs; it does not use the Manager's internal templates map.
	templateDir := t.TempDir()
	templateYAML := `name: "basic-ubuntu"
description: "Substrate test template"
base: "ubuntu-24.04"
package_manager: "apt"
users:
  - name: ubuntu
    groups: [sudo]
instance_defaults:
  ports: [22]
`
	if err := os.WriteFile(filepath.Join(templateDir, "basic-ubuntu.yml"), []byte(templateYAML), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	t.Setenv("PRISM_TEMPLATE_DIR", templateDir)

	// Pre-create VPC+subnet so ResolveNetworking skips DiscoverDefaultVPC
	// (Substrate only auto-creates the default VPC on RunInstances, not before)
	vpcID, subnetID := setupSubstrateVPC(t, manager.cfg)

	req := ctypes.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "test-instance-1",
		Size:     "M",
		Region:   "us-east-1",
		DryRun:   false,
		VpcID:    vpcID,
		SubnetID: subnetID,
	}

	instance, err := manager.LaunchInstance(req)
	if err != nil {
		t.Fatalf("LaunchInstance failed: %v", err)
	}
	if instance.Name != req.Name {
		t.Errorf("Instance name = %s, want %s", instance.Name, req.Name)
	}
	if instance.ID == "" {
		t.Error("Instance ID should not be empty")
	}
	if instance.HourlyRate <= 0 {
		t.Error("Instance should have positive hourly rate")
	}

	t.Cleanup(func() {
		if err := manager.DeleteInstance(req.Name); err != nil {
			t.Logf("cleanup: DeleteInstance: %v", err)
		}
	})
}

func TestSubstrateCreateEBSVolume(t *testing.T) {
	manager, _ := setupSubstrateManager(t)

	req := ctypes.StorageCreateRequest{
		Name:       "test-ebs-volume",
		Size:       "100",
		VolumeType: "gp3",
		Region:     "us-east-1",
	}

	volume, err := manager.CreateStorage(req)
	if err != nil {
		t.Fatalf("CreateStorage failed: %v", err)
	}
	if volume.Name != req.Name {
		t.Errorf("Volume name = %s, want %s", volume.Name, req.Name)
	}
	if volume.VolumeID == "" {
		t.Error("Volume should have a volume ID")
	}
	if volume.VolumeType != req.VolumeType {
		t.Errorf("Volume type = %s, want %s", volume.VolumeType, req.VolumeType)
	}
	if volume.SizeGB == nil || *volume.SizeGB != 100 {
		t.Errorf("Volume size = %v, want 100", volume.SizeGB)
	}

	t.Cleanup(func() {
		if err := manager.DeleteStorage(req.Name); err != nil {
			t.Logf("cleanup: DeleteStorage: %v", err)
		}
	})
}

func TestSubstrateEBSAttachDetach(t *testing.T) {
	manager, ts := setupSubstrateManager(t)

	// Seed AMI IDs needed by LaunchInstance (substrate#267)
	ts.SeedSSMParameters(map[string]string{
		"/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id": "ami-ubuntu2404-amd64",
		"/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id": "ami-ubuntu2404-arm64",
		"/aws/service/canonical/ubuntu/server/22.04/stable/current/amd64/hvm/ebs-gp3/ami-id": "ami-ubuntu2204-amd64",
		"/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64":              "ami-al2023-x86",
	})

	// Write minimal template to temp dir (same pattern as TestSubstrateLaunchInstance)
	templateDir := t.TempDir()
	templateYAML := `name: "basic-ubuntu"
description: "Substrate test template"
base: "ubuntu-24.04"
package_manager: "apt"
users:
  - name: ubuntu
    groups: [sudo]
instance_defaults:
  ports: [22]
`
	if err := os.WriteFile(filepath.Join(templateDir, "basic-ubuntu.yml"), []byte(templateYAML), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	t.Setenv("PRISM_TEMPLATE_DIR", templateDir)

	// Pre-create VPC+subnet (same reason as TestSubstrateLaunchInstance)
	vpcID, subnetID := setupSubstrateVPC(t, manager.cfg)

	// Launch an instance
	launchReq := ctypes.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "attach-test-instance",
		Size:     "M",
		Region:   "us-east-1",
		VpcID:    vpcID,
		SubnetID: subnetID,
	}
	instance, err := manager.LaunchInstance(launchReq)
	if err != nil {
		t.Fatalf("LaunchInstance failed: %v", err)
	}
	t.Cleanup(func() { manager.DeleteInstance(launchReq.Name) })

	// Create a volume
	volReq := ctypes.StorageCreateRequest{
		Name:       "attach-test-vol",
		Size:       "50",
		VolumeType: "gp3",
		Region:     "us-east-1",
	}
	vol, err := manager.CreateStorage(volReq)
	if err != nil {
		t.Fatalf("CreateStorage failed: %v", err)
	}
	t.Cleanup(func() { manager.DeleteStorage(volReq.Name) })

	// Attach
	if err := manager.AttachStorage(instance.Name, vol.Name); err != nil {
		t.Fatalf("AttachStorage failed: %v", err)
	}

	// Detach
	if err := manager.DetachStorage(vol.Name); err != nil {
		t.Fatalf("DetachStorage failed: %v", err)
	}
}

func TestSubstrateIAMInstanceProfile(t *testing.T) {
	_, ts := setupSubstrateManager(t)

	ctx := context.Background()
	cfg := aws.Config{
		Region: "us-east-1",
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: ts.URL, HostnameImmutable: true, SigningRegion: "us-east-1"}, nil
			},
		),
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
	iamClient := iam.NewFromConfig(cfg)

	// Create role
	_, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String("test-ec2-role"),
		AssumeRolePolicyDocument: aws.String(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}`),
	})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Create instance profile
	_, err = iamClient.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String("test-ec2-profile"),
	})
	if err != nil {
		t.Fatalf("CreateInstanceProfile failed: %v", err)
	}

	// Associate role with profile
	_, err = iamClient.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String("test-ec2-profile"),
		RoleName:            aws.String("test-ec2-role"),
	})
	if err != nil {
		t.Fatalf("AddRoleToInstanceProfile failed: %v", err)
	}

	// Verify via GetInstanceProfile
	resp, err := iamClient.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String("test-ec2-profile"),
	})
	if err != nil {
		t.Fatalf("GetInstanceProfile failed: %v", err)
	}
	if len(resp.InstanceProfile.Roles) != 1 {
		t.Errorf("Expected 1 role in profile, got %d", len(resp.InstanceProfile.Roles))
	}
	if *resp.InstanceProfile.Roles[0].RoleName != "test-ec2-role" {
		t.Errorf("Role name = %s, want test-ec2-role", *resp.InstanceProfile.Roles[0].RoleName)
	}
}

func TestSubstrateSSMRunCommand(t *testing.T) {
	_, ts := setupSubstrateManager(t)

	ctx := context.Background()
	cfg := aws.Config{
		Region: "us-east-1",
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: ts.URL, HostnameImmutable: true, SigningRegion: "us-east-1"}, nil
			},
		),
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
	ssmClient := ssm.NewFromConfig(cfg)

	// Send a command
	sendResp, err := ssmClient.SendCommand(ctx, &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{"i-test12345678"},
		Parameters: map[string][]string{
			"commands": {"echo hello"},
		},
	})
	if err != nil {
		t.Fatalf("SendCommand failed: %v", err)
	}
	if sendResp.Command == nil || sendResp.Command.CommandId == nil {
		t.Fatal("SendCommand returned nil CommandId")
	}
	commandID := *sendResp.Command.CommandId

	// Poll for completion
	getResp, err := ssmClient.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
		CommandId:  aws.String(commandID),
		InstanceId: aws.String("i-test12345678"),
	})
	if err != nil {
		t.Fatalf("GetCommandInvocation failed: %v", err)
	}
	if string(getResp.Status) != "Success" {
		t.Errorf("Command status = %s, want Success", getResp.Status)
	}
}

func TestSubstrateErrorHandling(t *testing.T) {
	manager, _ := setupSubstrateManager(t)

	// LaunchInstance with bad template should fail early (before any AWS call)
	_, err := manager.LaunchInstance(ctypes.LaunchRequest{
		Template: "nonexistent-template",
		Name:     "test-error-instance",
		Size:     "M",
		Region:   "us-east-1",
	})
	if err == nil {
		t.Error("LaunchInstance should fail with nonexistent template")
	}

	// DeleteStorage on nonexistent volume should fail
	if err := manager.DeleteStorage("nonexistent-volume"); err == nil {
		t.Error("DeleteStorage should fail for nonexistent volume")
	}
}
