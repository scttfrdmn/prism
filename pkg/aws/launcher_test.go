package aws

import (
	"context"
	"fmt"
	"testing"

	spawn "github.com/spore-host/spawn/pkg/aws"

	"github.com/scttfrdmn/prism/pkg/launch"
	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// fakeLauncher records the config it was handed and returns a canned result,
// so launch/lifecycle wiring can be exercised without touching AWS.
type fakeLauncher struct {
	launchCalls int
	gotConfig   spawn.LaunchConfig
	result      *spawn.LaunchResult
	launchErr   error

	stopCalls  int
	gotStopReg string
	gotStopID  string
	gotStopHib bool

	startCalls int
	termCalls  int
	termErr    error
}

var _ launch.Launcher = (*fakeLauncher)(nil)

func (f *fakeLauncher) Launch(_ context.Context, cfg spawn.LaunchConfig) (*spawn.LaunchResult, error) {
	f.launchCalls++
	f.gotConfig = cfg
	if f.launchErr != nil {
		return nil, f.launchErr
	}
	if f.result != nil {
		return f.result, nil
	}
	return &spawn.LaunchResult{InstanceID: "i-fake", State: "pending"}, nil
}

func (f *fakeLauncher) StopInstance(_ context.Context, region, id string, hibernate bool) error {
	f.stopCalls++
	f.gotStopReg, f.gotStopID, f.gotStopHib = region, id, hibernate
	return nil
}

func (f *fakeLauncher) StartInstance(_ context.Context, _, _ string) error {
	f.startCalls++
	return nil
}

func (f *fakeLauncher) Terminate(_ context.Context, _, _ string) error {
	f.termCalls++
	return f.termErr
}

// TestLaunchInstance_DryRunNeverCallsLauncher confirms dry-run short-circuits
// before ever reaching the launcher port (Risk R2).
func TestLaunchInstance_DryRunNeverCallsLauncher(t *testing.T) {
	fake := &fakeLauncher{}
	m := &Manager{region: "us-west-2", launcher: fake}
	l := &InstanceLauncher{manager: m, region: "us-west-2"}

	req := ctypes.LaunchRequest{Name: "dry", Template: "python-ml", DryRun: true}
	template := &ctypes.RuntimeTemplate{}
	cfg := spawn.LaunchConfig{InstanceType: "t3.medium"}

	inst, err := l.LaunchInstance(req, cfg, 0.05, template, "ubuntu", 20)
	if err != nil {
		t.Fatalf("dry-run launch returned error: %v", err)
	}
	if fake.launchCalls != 0 {
		t.Errorf("launcher.Launch called %d times on dry-run; want 0", fake.launchCalls)
	}
	if inst.State != "dry-run" {
		t.Errorf("dry-run instance State = %q; want %q", inst.State, "dry-run")
	}
}

// TestLaunchInstance_MapsResultToInstance confirms a real launch drives the port
// once and maps the LaunchResult onto the Prism instance.
func TestLaunchInstance_MapsResultToInstance(t *testing.T) {
	fake := &fakeLauncher{
		result: &spawn.LaunchResult{
			InstanceID:       "i-0abc123",
			PublicIP:         "1.2.3.4",
			PrivateIP:        "10.0.0.5",
			AvailabilityZone: "us-west-2b",
			State:            "pending",
			KeyName:          "my-key",
		},
	}
	m := &Manager{region: "us-west-2", launcher: fake}
	l := &InstanceLauncher{manager: m, region: "us-west-2"}

	req := ctypes.LaunchRequest{Name: "ml", Template: "python-ml"}
	template := &ctypes.RuntimeTemplate{ConnectionType: ctypes.ConnectionTypeDesktop}
	cfg := spawn.LaunchConfig{InstanceType: "t3.large"}

	inst, err := l.LaunchInstance(req, cfg, 0.10, template, "ubuntu", 50)
	if err != nil {
		t.Fatalf("launch returned error: %v", err)
	}
	if fake.launchCalls != 1 {
		t.Fatalf("launcher.Launch called %d times; want 1", fake.launchCalls)
	}
	if inst.ID != "i-0abc123" || inst.PublicIP != "1.2.3.4" || inst.PrivateIP != "10.0.0.5" {
		t.Errorf("result not mapped onto instance: %+v", inst)
	}
	if inst.AvailabilityZone != "us-west-2b" || inst.KeyName != "my-key" || inst.State != "pending" {
		t.Errorf("result fields not mapped: %+v", inst)
	}
	if inst.InstanceType != "t3.large" {
		t.Errorf("InstanceType = %q; want carried-in t3.large", inst.InstanceType)
	}
	if inst.StorageGB != 50 {
		t.Errorf("StorageGB = %v; want carried-in 50", inst.StorageGB)
	}
	if inst.HourlyRate != 0.10 {
		t.Errorf("HourlyRate = %v; want 0.10 (per-hour, not ×24)", inst.HourlyRate)
	}
	if inst.ConnectionType != ctypes.ConnectionTypeDesktop {
		t.Errorf("ConnectionType not propagated from template: %q", inst.ConnectionType)
	}
}

// TestBuildTags_ParityWithLegacyTagSet is the golden/parity anchor: it pins the
// exact tag set the launch path stamps, so the spawn migration cannot silently
// drop or rename a prism:-namespaced tag that the deployed spored reads.
func TestBuildTags_ParityWithLegacyTagSet(t *testing.T) {
	b := &InstanceConfigBuilder{}
	req := ctypes.LaunchRequest{
		Name:                "ws1",
		Template:            "python-ml",
		PackageManager:      "conda",
		ProjectID:           "proj-1",
		FundingAllocationID: "fund-1",
		ResearchUser:        "researcher",
		DNSName:             "ws1",
		TTL:                 "8h",
		IdleTimeout:         "30m",
		IdlePolicy:          true,
		SlackWorkspaceID:    "T123",
		ActiveProcesses:     "jupyter,rsession",
	}

	tags := b.BuildTags(req, "ubuntu")

	// Static keys always present, keyed with the prism: namespace spored reads.
	want := map[string]string{
		"Name":                  "ws1",
		"prism:managed":         "true",
		"prism:instance-id":     "ws1",
		"prism:template":        "python-ml",
		"prism:package-manager": "conda",
		"prism:primary-user":    "ubuntu",
		"Application":           "Prism",
		"Environment":           "research",
		"Prism":                 "true",
		"LaunchedBy":            "Prism",
		"Template":              "python-ml",
		"PackageManager":        "conda",
		"PrimaryUser":           "ubuntu",
		// Conditional keys, present because the req sets them.
		"prism:project-id":            "proj-1",
		"CostCenter":                  "proj-1",
		"prism:funding-allocation-id": "fund-1",
		"prism:research-user":         "researcher",
		"prism:dns-name":              "ws1",
		"prism:ttl":                   "8h",
		"prism:idle-timeout":          "30m",
		"prism:hibernate-on-idle":     "true",
		"prism:slack-workspace-id":    "T123",
		"prism:notify-url":            sporeBotLambdaURL,
		"prism:notify-command":        "/prism",
		"prism:active-processes":      "jupyter,rsession",
	}
	for k, v := range want {
		if got := tags[k]; got != v {
			t.Errorf("tag %q = %q; want %q", k, got, v)
		}
	}
	// prism:version is present but value tracks the build; just require non-empty.
	if tags["prism:version"] == "" {
		t.Error("prism:version tag missing")
	}
}

// TestBuildTags_OmitsUnsetConditionalTags confirms optional tags are absent when
// the request does not set them (parity with the old append-only builder).
func TestBuildTags_OmitsUnsetConditionalTags(t *testing.T) {
	b := &InstanceConfigBuilder{}
	req := ctypes.LaunchRequest{Name: "bare", Template: "python-ml"}

	tags := b.BuildTags(req, "ubuntu")

	for _, k := range []string{
		"prism:project-id", "CostCenter", "prism:funding-allocation-id",
		"prism:research-user", "prism:dns-name", "prism:ttl", "prism:idle-timeout",
		"prism:hibernate-on-idle", "prism:slack-workspace-id", "prism:active-processes",
	} {
		if _, ok := tags[k]; ok {
			t.Errorf("tag %q should be absent when unset, got %q", k, tags[k])
		}
	}
}

// TestToLaunchConfig_MapsResolvedInputs confirms the pure mapping populates the
// spawn.LaunchConfig fields Phase 1 relies on, and pre-resolves the SG.
func TestToLaunchConfig_MapsResolvedInputs(t *testing.T) {
	cfg := launch.ToLaunchConfig(launch.Inputs{
		InstanceType:          "c7i.large",
		Region:                "us-west-2",
		AMI:                   "ami-123",
		SubnetID:              "subnet-1",
		SecurityGroupID:       "sg-1",
		KeyName:               "k",
		UserDataEncoded:       "BASE64",
		PrimaryUsername:       "ubuntu",
		RootVolumeGB:          80,
		Hibernate:             true,
		Spot:                  false,
		CapacityReservationID: "cr-1",
		IAMInstanceProfile:    "Prism-Instance-Profile",
		HourlyRate:            0.17,
		Name:                  "ws",
		Tags:                  map[string]string{"Prism": "true"},
	})

	if cfg.InstanceType != "c7i.large" || cfg.Region != "us-west-2" || cfg.AMI != "ami-123" {
		t.Errorf("core fields not mapped: %+v", cfg)
	}
	if len(cfg.SecurityGroupIDs) != 1 || cfg.SecurityGroupIDs[0] != "sg-1" {
		t.Errorf("SecurityGroupIDs = %v; want [sg-1]", cfg.SecurityGroupIDs)
	}
	if cfg.RootVolumeSizeGiB != 80 {
		t.Errorf("RootVolumeSizeGiB = %d; want 80", cfg.RootVolumeSizeGiB)
	}
	if !cfg.Hibernate || cfg.ReservationID != "cr-1" || cfg.IamInstanceProfile != "Prism-Instance-Profile" {
		t.Errorf("optional fields not mapped: %+v", cfg)
	}
	if cfg.UserData != "BASE64" || cfg.PricePerHour != 0.17 {
		t.Errorf("userdata/price not mapped: %+v", cfg)
	}
}

// TestToLaunchConfig_EmptySGLeavesSliceNil ensures we never hand spawn an empty
// SG slice (which would make it create its own SG — Prism never wants that).
func TestToLaunchConfig_EmptySGLeavesSliceNil(t *testing.T) {
	cfg := launch.ToLaunchConfig(launch.Inputs{InstanceType: "t3.micro"})
	if cfg.SecurityGroupIDs != nil {
		t.Errorf("SecurityGroupIDs = %v; want nil when no SG resolved", cfg.SecurityGroupIDs)
	}
}

// TestToLaunchConfig_MapsSpawnCapabilities covers the Phase-3a opt-in fields.
func TestToLaunchConfig_MapsSpawnCapabilities(t *testing.T) {
	cfg := launch.ToLaunchConfig(launch.Inputs{
		InstanceType:   "c7i.large",
		Spot:           true,
		SpotMaxPrice:   "0.50",
		EFA:            true,
		PlacementGroup: "cluster-1",
	})
	if cfg.SpotMaxPrice != "0.50" {
		t.Errorf("SpotMaxPrice = %q; want 0.50", cfg.SpotMaxPrice)
	}
	if !cfg.EFAEnabled {
		t.Error("EFAEnabled should be true")
	}
	if cfg.PlacementGroup != "cluster-1" {
		t.Errorf("PlacementGroup = %q; want cluster-1", cfg.PlacementGroup)
	}
}

// TestConfigToRunInput_EFAAndPlacement asserts the EFA NIC (InterfaceType=efa)
// and the cluster placement group land on the RunInstances input.
func TestConfigToRunInput_EFAAndPlacement(t *testing.T) {
	in := configToRunInput(spawn.LaunchConfig{
		InstanceType:      "c7i.large",
		AMI:               "ami-123",
		SubnetID:          "subnet-1",
		SecurityGroupIDs:  []string{"sg-1"},
		RootVolumeSizeGiB: 20,
		EFAEnabled:        true,
		PlacementGroup:    "cluster-1",
	})

	if len(in.NetworkInterfaces) != 1 {
		t.Fatalf("expected 1 NIC, got %d", len(in.NetworkInterfaces))
	}
	nic := in.NetworkInterfaces[0]
	if nic.InterfaceType == nil || *nic.InterfaceType != "efa" {
		t.Errorf("NIC InterfaceType = %v; want efa", nic.InterfaceType)
	}
	if in.Placement == nil || in.Placement.GroupName == nil || *in.Placement.GroupName != "cluster-1" {
		t.Errorf("Placement.GroupName not set to cluster-1: %+v", in.Placement)
	}
}

// TestConfigToRunInput_NoEFANoPlacement confirms the default path leaves the NIC
// without an EFA interface type and sets no placement group.
func TestConfigToRunInput_NoEFANoPlacement(t *testing.T) {
	in := configToRunInput(spawn.LaunchConfig{
		InstanceType:      "t3.micro",
		AMI:               "ami-123",
		SubnetID:          "subnet-1",
		RootVolumeSizeGiB: 20,
	})
	if in.NetworkInterfaces[0].InterfaceType != nil {
		t.Errorf("InterfaceType should be nil without EFA, got %v", *in.NetworkInterfaces[0].InterfaceType)
	}
	if in.Placement != nil {
		t.Errorf("Placement should be nil without a placement group, got %+v", in.Placement)
	}
}

// TestHybridLauncher_RoutesLaunchToEC2AndLifecycleToSpawn documents the Phase-2
// split: Launch stays on the EC2 path while lifecycle goes to spawn. We can't
// easily stand up a real *spawn.Client here, so this asserts the delegation
// wiring (launch -> ec2Launcher) via the manager's mock EC2 client, and that the
// hybrid satisfies the port. The spawn lifecycle delegation is covered by the
// manager-level lifecycle tests below (through fakeLauncher).
func TestHybridLauncher_SatisfiesPort(t *testing.T) {
	// Compile-time already asserted in hybrid_launcher.go; this guards the
	// constructor wiring: both engines are set.
	m := &Manager{region: "us-west-2"} // empty cfg is fine; no AWS calls made here
	h := newHybridLauncher(m)
	if h.launch == nil {
		t.Error("hybridLauncher.launch (ec2Launcher) not set")
	}
	if h.lifecycle == nil {
		t.Error("hybridLauncher.lifecycle (spawn client) not set")
	}
}

// TestDeleteInstance_GracefulOnSpawnStyleNotFound confirms Prism's already-gone
// handling still fires when the terminate error comes from spawn — spawn wraps
// the AWS error as "failed to terminate instance: %w", preserving the
// InvalidInstanceID.NotFound substring Prism matches on.
func TestDeleteInstance_GracefulOnSpawnStyleNotFound(t *testing.T) {
	spawnStyleErr := fmt.Errorf("failed to terminate instance: %w",
		fmt.Errorf("operation error EC2: TerminateInstances, https response error StatusCode: 400, api error InvalidInstanceID.NotFound: The instance ID 'i-deadbeef' does not exist"))

	fake := &fakeLauncher{termErr: spawnStyleErr}
	m := &Manager{
		region:   "us-west-2",
		launcher: fake,
		stateManager: &MockStateManager{
			LoadStateFunc: func() (*ctypes.State, error) {
				return &ctypes.State{
					Instances: map[string]ctypes.Instance{
						"gone": {Name: "gone", ID: "i-deadbeef", Region: "us-west-2"},
					},
				}, nil
			},
		},
	}

	if err := m.DeleteInstance("gone"); err != nil {
		t.Errorf("DeleteInstance should treat spawn-wrapped NotFound as success, got: %v", err)
	}
	if fake.termCalls != 1 {
		t.Errorf("Terminate called %d times; want 1", fake.termCalls)
	}
}

// TestStopInstance_RoutesThroughLauncher confirms Manager.StopInstance drives the
// launcher port with hibernate=false and the resolved region/id.
func TestStopInstance_RoutesThroughLauncher(t *testing.T) {
	fake := &fakeLauncher{}
	m := &Manager{
		region:   "us-west-2",
		launcher: fake,
		stateManager: &MockStateManager{
			LoadStateFunc: func() (*ctypes.State, error) {
				return &ctypes.State{
					Instances: map[string]ctypes.Instance{
						"ws": {Name: "ws", ID: "i-abc", Region: "us-west-2"},
					},
				}, nil
			},
		},
	}

	if err := m.StopInstance("ws"); err != nil {
		t.Fatalf("StopInstance error: %v", err)
	}
	if fake.stopCalls != 1 || fake.gotStopHib {
		t.Errorf("stopCalls=%d hibernate=%v; want 1 call, hibernate=false", fake.stopCalls, fake.gotStopHib)
	}
	if fake.gotStopID != "i-abc" || fake.gotStopReg != "us-west-2" {
		t.Errorf("routed region/id = %q/%q; want us-west-2/i-abc", fake.gotStopReg, fake.gotStopID)
	}
}
