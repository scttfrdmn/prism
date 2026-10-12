package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// TestMergeLocalStateMetadata_PreservesStorageGB (#720): AWS's instance view carries no root volume
// size, so a refresh must keep the one recorded at launch or the ledger stops accruing EBS storage.
func TestMergeLocalStateMetadata_PreservesStorageGB(t *testing.T) {
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": {Name: "ws", StorageGB: 50}}}

	built := &ctypes.Instance{Name: "ws"}
	mergeLocalStateMetadata(built, "ws", "stopped", local)
	if built.StorageGB != 50 {
		t.Fatalf("StorageGB = %v, want 50 carried from local state", built.StorageGB)
	}

	fresh := &ctypes.Instance{Name: "ws", StorageGB: 80}
	mergeLocalStateMetadata(fresh, "ws", "stopped", local)
	if fresh.StorageGB != 80 {
		t.Fatalf("StorageGB = %v, want the built value 80 kept", fresh.StorageGB)
	}
}

// ec2Stopped is AWS's view of the "ws" workspace after it stopped: no public IP, and none of the
// fields only prism knows.
func ec2Stopped(id string) ec2types.Instance {
	return ec2types.Instance{
		InstanceId:   aws.String(id),
		InstanceType: ec2types.InstanceTypeT3Medium,
		LaunchTime:   aws.Time(time.Now().Add(-48 * time.Hour)),
		State:        &ec2types.InstanceState{Name: ec2types.InstanceStateNameStopped},
		Tags: []ec2types.Tag{
			{Key: aws.String("Name"), Value: aws.String("ws")},
			{Key: aws.String("Template"), Value: aws.String("python-ml")},
		},
	}
}

// storedWorkspace is the record prism wrote at launch and kept since.
func storedWorkspace() ctypes.Instance {
	return ctypes.Instance{
		ID:              "i-1",
		Name:            "ws",
		Template:        "python-ml",
		Region:          "us-east-2",
		State:           "running",
		PublicIP:        "203.0.113.7",
		PrivateIP:       "10.0.0.7",
		Username:        "researcher",
		DCVPassword:     "s3cret",
		Services:        []ctypes.Service{{Name: "jupyter", Port: 8888}},
		AttachedVolumes: []string{"shared-efs"},
		IdleDetection:   &ctypes.IdleDetection{Enabled: true},
		AlwaysOn:        true,
		StorageGB:       80,
		IsHibernating:   true,
		LaunchTime:      time.Now().Add(-72 * time.Hour), // original launch, before the latest start
	}
}

// TestBuildInstance_KeepsPrismFieldsOnRefresh (#721): a refresh used to rebuild the record from
// AWS's view alone, so every save dropped region, volumes, services, idle settings and the rest.
func TestBuildInstance_KeepsPrismFieldsOnRefresh(t *testing.T) {
	b := NewInstanceBuilder(&MockEC2Client{}, &PricingClient{}, nil, nil, "us-east-2")
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": storedWorkspace()}}

	got := b.BuildInstance(ec2Stopped("i-1"), local)
	if got == nil {
		t.Fatal("BuildInstance returned nil")
	}

	// prism-only fields survive
	if got.Region != "us-east-2" || got.PrivateIP != "10.0.0.7" || got.Username != "researcher" ||
		got.DCVPassword != "s3cret" || !got.AlwaysOn || got.StorageGB != 80 {
		t.Errorf("prism-only scalars lost: %+v", got)
	}
	if len(got.Services) != 1 || len(got.AttachedVolumes) != 1 || got.IdleDetection == nil || !got.IdleDetection.Enabled {
		t.Errorf("prism-only collections lost: services=%v volumes=%v idle=%v", got.Services, got.AttachedVolumes, got.IdleDetection)
	}
	// AWS-owned fields come from AWS
	if got.State != "stopped" || got.PublicIP != "" || got.InstanceType != "t3.medium" {
		t.Errorf("AWS fields not refreshed: state=%q publicIP=%q type=%q", got.State, got.PublicIP, got.InstanceType)
	}
	// recomputed, not inherited: the instance reached stopped, so hibernation is over
	if got.IsHibernating {
		t.Error("IsHibernating inherited from the stored record; it must be recomputed from the observed state")
	}
}

// A stored record for a different EC2 instance under the same name (a reused name) is not merged.
func TestBuildInstance_DifferentIDIsNotMerged(t *testing.T) {
	b := NewInstanceBuilder(&MockEC2Client{}, &PricingClient{}, nil, nil, "us-east-2")
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": storedWorkspace()}}

	got := b.BuildInstance(ec2Stopped("i-2"), local)
	if got == nil {
		t.Fatal("BuildInstance returned nil")
	}
	if got.ID != "i-2" || got.Username != "" || got.DCVPassword != "" || len(got.AttachedVolumes) != 0 {
		t.Errorf("another instance's record leaked into i-2: %+v", got)
	}
}

// A stored record with no launch time must not replace AWS's: a zero LaunchTime made the estimate
// bill every hour since year 1.
func TestBuildInstance_ZeroStoredLaunchTimeUsesAWS(t *testing.T) {
	b := NewInstanceBuilder(&MockEC2Client{}, &PricingClient{}, nil, nil, "us-east-2")
	stored := storedWorkspace()
	stored.LaunchTime = time.Time{}
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": stored}}

	got := b.BuildInstance(ec2Stopped("i-1"), local)
	if got == nil || got.LaunchTime.IsZero() {
		t.Fatalf("LaunchTime = %v, want AWS's launch time", got.LaunchTime)
	}
	if hours := time.Since(got.LaunchTime).Hours(); hours < 47 || hours > 49 {
		t.Errorf("LaunchTime is %.1fh ago, want ~48h (AWS's)", hours)
	}
}

// The stored original launch time wins over AWS's, which resets on every start.
func TestBuildInstance_KeepsOriginalLaunchTime(t *testing.T) {
	b := NewInstanceBuilder(&MockEC2Client{}, &PricingClient{}, nil, nil, "us-east-2")
	local := &ctypes.State{Instances: map[string]ctypes.Instance{"ws": storedWorkspace()}}

	got := b.BuildInstance(ec2Stopped("i-1"), local)
	if hours := time.Since(got.LaunchTime).Hours(); hours < 71 || hours > 73 {
		t.Errorf("LaunchTime is %.1fh ago, want ~72h (the stored original launch)", hours)
	}
}
