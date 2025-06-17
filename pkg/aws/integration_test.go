// +build integration

package aws

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
)

const localstackEndpoint = "http://localhost:4566"

// setupLocalStackManager creates a Manager configured to use LocalStack
func setupLocalStackManager(t *testing.T) *Manager {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test - set INTEGRATION_TESTS=1 to run")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           localstackEndpoint,
					SigningRegion: "us-east-1",
				}, nil
			})),
	)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	manager := &Manager{
		cfg:             cfg,
		ec2:             ec2.NewFromConfig(cfg),
		efs:             efs.NewFromConfig(cfg),
		sts:             sts.NewFromConfig(cfg),
		region:          "us-east-1",
		templates:       getTemplates(),
		pricingCache:    make(map[string]float64),
		lastPriceUpdate: time.Time{},
		discountConfig:  ctypes.DiscountConfig{},
	}

	return manager
}

func TestIntegrationLaunchInstance(t *testing.T) {
	manager := setupLocalStackManager(t)

	// Test launching an instance
	req := ctypes.LaunchRequest{
		Template: "basic-ubuntu",
		Name:     "test-instance-1",
		Size:     "M",
		Region:   "us-east-1",
		DryRun:   false,
	}

	instance, err := manager.LaunchInstance(req)
	if err != nil {
		t.Fatalf("LaunchInstance failed: %v", err)
	}

	// Verify instance properties
	if instance.Name != req.Name {
		t.Errorf("Instance name = %s, want %s", instance.Name, req.Name)
	}

	if instance.Template != req.Template {
		t.Errorf("Instance template = %s, want %s", instance.Template, req.Template)
	}

	if instance.ID == "" {
		t.Error("Instance ID should not be empty")
	}

	if instance.EstimatedDailyCost <= 0 {
		t.Error("Instance should have positive daily cost")
	}

	// Clean up
	defer func() {
		if err := manager.DeleteInstance(req.Name); err != nil {
			t.Logf("Failed to clean up instance: %v", err)
		}
	}()
}

func TestIntegrationCreateEBSVolume(t *testing.T) {
	manager := setupLocalStackManager(t)

	req := ctypes.StorageCreateRequest{
		Name:       "test-ebs-volume",
		Size:       "100", // 100 GB
		VolumeType: "gp3",
		Region:     "us-east-1",
	}

	volume, err := manager.CreateStorage(req)
	if err != nil {
		t.Fatalf("CreateStorage failed: %v", err)
	}

	// Verify volume properties
	if volume.Name != req.Name {
		t.Errorf("Volume name = %s, want %s", volume.Name, req.Name)
	}

	if volume.VolumeID == "" {
		t.Error("Volume should have a volume ID")
	}

	if volume.VolumeType != req.VolumeType {
		t.Errorf("Volume type = %s, want %s", volume.VolumeType, req.VolumeType)
	}

	if volume.SizeGB != 100 {
		t.Errorf("Volume size = %d, want 100", volume.SizeGB)
	}

	if volume.IOPS <= 0 {
		t.Error("GP3 volume should have positive IOPS")
	}

	if volume.Throughput <= 0 {
		t.Error("GP3 volume should have positive throughput")
	}

	// Clean up
	defer func() {
		if err := manager.DeleteStorage(req.Name); err != nil {
			t.Logf("Failed to clean up storage: %v", err)
		}
	}()
}

func TestIntegrationErrorHandling(t *testing.T) {
	manager := setupLocalStackManager(t)

	// Test launching instance with invalid template
	req := ctypes.LaunchRequest{
		Template: "nonexistent-template",
		Name:     "test-error-instance",
		Size:     "M",
		Region:   "us-east-1",
	}

	_, err := manager.LaunchInstance(req)
	if err == nil {
		t.Error("LaunchInstance should fail with nonexistent template")
	}

	// Test operations on nonexistent instance
	err = manager.StartInstance("nonexistent-instance")
	if err == nil {
		t.Error("StartInstance should fail with nonexistent instance")
	}

	err = manager.StopInstance("nonexistent-instance")
	if err == nil {
		t.Error("StopInstance should fail with nonexistent instance")
	}

	_, err = manager.GetConnectionInfo("nonexistent-instance")
	if err == nil {
		t.Error("GetConnectionInfo should fail with nonexistent instance")
	}

	err = manager.DeleteInstance("nonexistent-instance")
	if err == nil {
		t.Error("DeleteInstance should fail with nonexistent instance")
	}
}