package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// MockEC2Client provides a mock implementation of EC2ClientInterface for testing
type MockEC2Client struct {
	RunInstancesFunc                  func(ctx context.Context, params *ec2.RunInstancesInput) (*ec2.RunInstancesOutput, error)
	DescribeInstancesFunc             func(ctx context.Context, params *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
	TerminateInstancesFunc            func(ctx context.Context, params *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error)
	CreateImageFunc                   func(ctx context.Context, params *ec2.CreateImageInput) (*ec2.CreateImageOutput, error)
	DescribeImagesFunc                func(ctx context.Context, params *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)
	CopyImageFunc                     func(ctx context.Context, params *ec2.CopyImageInput) (*ec2.CopyImageOutput, error)
	DescribeSecurityGroupsFunc        func(ctx context.Context, params *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error)
	CreateSecurityGroupFunc           func(ctx context.Context, params *ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error)
	AuthorizeSecurityGroupIngressFunc func(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	DescribeVpcsFunc                  func(ctx context.Context, params *ec2.DescribeVpcsInput) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnetsFunc               func(ctx context.Context, params *ec2.DescribeSubnetsInput) (*ec2.DescribeSubnetsOutput, error)
	DescribeRouteTablesFunc           func(ctx context.Context, params *ec2.DescribeRouteTablesInput) (*ec2.DescribeRouteTablesOutput, error)
	StartInstancesFunc                func(ctx context.Context, params *ec2.StartInstancesInput) (*ec2.StartInstancesOutput, error)
	StopInstancesFunc                 func(ctx context.Context, params *ec2.StopInstancesInput) (*ec2.StopInstancesOutput, error)
	CreateVolumeFunc                  func(ctx context.Context, params *ec2.CreateVolumeInput) (*ec2.CreateVolumeOutput, error)
	DeleteVolumeFunc                  func(ctx context.Context, params *ec2.DeleteVolumeInput) (*ec2.DeleteVolumeOutput, error)
	DescribeVolumesFunc               func(ctx context.Context, params *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error)
	AttachVolumeFunc                  func(ctx context.Context, params *ec2.AttachVolumeInput) (*ec2.AttachVolumeOutput, error)
	DetachVolumeFunc                  func(ctx context.Context, params *ec2.DetachVolumeInput) (*ec2.DetachVolumeOutput, error)
	DescribeKeyPairsFunc              func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error)
	ImportKeyPairFunc                 func(ctx context.Context, params *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error)
	DeleteKeyPairFunc                 func(ctx context.Context, params *ec2.DeleteKeyPairInput) (*ec2.DeleteKeyPairOutput, error)
	GetConsoleOutputFunc              func(ctx context.Context, params *ec2.GetConsoleOutputInput) (*ec2.GetConsoleOutputOutput, error)
	ModifyInstanceAttributeFunc       func(ctx context.Context, params *ec2.ModifyInstanceAttributeInput) (*ec2.ModifyInstanceAttributeOutput, error)
	DeregisterImageFunc               func(ctx context.Context, params *ec2.DeregisterImageInput) (*ec2.DeregisterImageOutput, error)
	DeleteSnapshotFunc                func(ctx context.Context, params *ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error)
	DescribeSnapshotsFunc             func(ctx context.Context, params *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error)
	ModifyImageAttributeFunc          func(ctx context.Context, params *ec2.ModifyImageAttributeInput) (*ec2.ModifyImageAttributeOutput, error)
	CreateTagsFunc                    func(ctx context.Context, params *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
}

func (m *MockEC2Client) RunInstances(ctx context.Context, params *ec2.RunInstancesInput, optFns ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error) {
	if m.RunInstancesFunc != nil {
		return m.RunInstancesFunc(ctx, params)
	}
	return nil, nil
}

func (m *MockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if m.DescribeInstancesFunc != nil {
		return m.DescribeInstancesFunc(ctx, params)
	}
	return &ec2.DescribeInstancesOutput{}, nil
}

func (m *MockEC2Client) TerminateInstances(ctx context.Context, params *ec2.TerminateInstancesInput, optFns ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error) {
	if m.TerminateInstancesFunc != nil {
		return m.TerminateInstancesFunc(ctx, params)
	}
	return &ec2.TerminateInstancesOutput{}, nil
}

func (m *MockEC2Client) CreateImage(ctx context.Context, params *ec2.CreateImageInput, optFns ...func(*ec2.Options)) (*ec2.CreateImageOutput, error) {
	if m.CreateImageFunc != nil {
		return m.CreateImageFunc(ctx, params)
	}
	return &ec2.CreateImageOutput{}, nil
}

func (m *MockEC2Client) DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	if m.DescribeImagesFunc != nil {
		return m.DescribeImagesFunc(ctx, params)
	}
	return &ec2.DescribeImagesOutput{}, nil
}

func (m *MockEC2Client) CopyImage(ctx context.Context, params *ec2.CopyImageInput, optFns ...func(*ec2.Options)) (*ec2.CopyImageOutput, error) {
	if m.CopyImageFunc != nil {
		return m.CopyImageFunc(ctx, params)
	}
	return &ec2.CopyImageOutput{}, nil
}

func (m *MockEC2Client) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	if m.DescribeSecurityGroupsFunc != nil {
		return m.DescribeSecurityGroupsFunc(ctx, params)
	}
	return &ec2.DescribeSecurityGroupsOutput{}, nil
}

func (m *MockEC2Client) CreateSecurityGroup(ctx context.Context, params *ec2.CreateSecurityGroupInput, optFns ...func(*ec2.Options)) (*ec2.CreateSecurityGroupOutput, error) {
	if m.CreateSecurityGroupFunc != nil {
		return m.CreateSecurityGroupFunc(ctx, params)
	}
	return &ec2.CreateSecurityGroupOutput{}, nil
}

func (m *MockEC2Client) AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	if m.AuthorizeSecurityGroupIngressFunc != nil {
		return m.AuthorizeSecurityGroupIngressFunc(ctx, params)
	}
	return &ec2.AuthorizeSecurityGroupIngressOutput{}, nil
}

func (m *MockEC2Client) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	if m.DescribeVpcsFunc != nil {
		return m.DescribeVpcsFunc(ctx, params)
	}
	return &ec2.DescribeVpcsOutput{}, nil
}

func (m *MockEC2Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	if m.DescribeSubnetsFunc != nil {
		return m.DescribeSubnetsFunc(ctx, params)
	}
	return &ec2.DescribeSubnetsOutput{}, nil
}

func (m *MockEC2Client) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	if m.DescribeRouteTablesFunc != nil {
		return m.DescribeRouteTablesFunc(ctx, params)
	}
	return &ec2.DescribeRouteTablesOutput{}, nil
}

func (m *MockEC2Client) StartInstances(ctx context.Context, params *ec2.StartInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
	if m.StartInstancesFunc != nil {
		return m.StartInstancesFunc(ctx, params)
	}
	return &ec2.StartInstancesOutput{}, nil
}

func (m *MockEC2Client) StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
	if m.StopInstancesFunc != nil {
		return m.StopInstancesFunc(ctx, params)
	}
	return &ec2.StopInstancesOutput{}, nil
}

func (m *MockEC2Client) CreateVolume(ctx context.Context, params *ec2.CreateVolumeInput, optFns ...func(*ec2.Options)) (*ec2.CreateVolumeOutput, error) {
	if m.CreateVolumeFunc != nil {
		return m.CreateVolumeFunc(ctx, params)
	}
	return &ec2.CreateVolumeOutput{}, nil
}

func (m *MockEC2Client) DeleteVolume(ctx context.Context, params *ec2.DeleteVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVolumeOutput, error) {
	if m.DeleteVolumeFunc != nil {
		return m.DeleteVolumeFunc(ctx, params)
	}
	return &ec2.DeleteVolumeOutput{}, nil
}

func (m *MockEC2Client) DescribeVolumes(ctx context.Context, params *ec2.DescribeVolumesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	if m.DescribeVolumesFunc != nil {
		return m.DescribeVolumesFunc(ctx, params)
	}
	return &ec2.DescribeVolumesOutput{}, nil
}

func (m *MockEC2Client) AttachVolume(ctx context.Context, params *ec2.AttachVolumeInput, optFns ...func(*ec2.Options)) (*ec2.AttachVolumeOutput, error) {
	if m.AttachVolumeFunc != nil {
		return m.AttachVolumeFunc(ctx, params)
	}
	return &ec2.AttachVolumeOutput{}, nil
}

func (m *MockEC2Client) DetachVolume(ctx context.Context, params *ec2.DetachVolumeInput, optFns ...func(*ec2.Options)) (*ec2.DetachVolumeOutput, error) {
	if m.DetachVolumeFunc != nil {
		return m.DetachVolumeFunc(ctx, params)
	}
	return &ec2.DetachVolumeOutput{}, nil
}

func (m *MockEC2Client) DescribeKeyPairs(ctx context.Context, params *ec2.DescribeKeyPairsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error) {
	if m.DescribeKeyPairsFunc != nil {
		return m.DescribeKeyPairsFunc(ctx, params)
	}
	return &ec2.DescribeKeyPairsOutput{}, nil
}

func (m *MockEC2Client) ImportKeyPair(ctx context.Context, params *ec2.ImportKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error) {
	if m.ImportKeyPairFunc != nil {
		return m.ImportKeyPairFunc(ctx, params)
	}
	return &ec2.ImportKeyPairOutput{}, nil
}

func (m *MockEC2Client) DeleteKeyPair(ctx context.Context, params *ec2.DeleteKeyPairInput, optFns ...func(*ec2.Options)) (*ec2.DeleteKeyPairOutput, error) {
	if m.DeleteKeyPairFunc != nil {
		return m.DeleteKeyPairFunc(ctx, params)
	}
	return &ec2.DeleteKeyPairOutput{}, nil
}

func (m *MockEC2Client) GetConsoleOutput(ctx context.Context, params *ec2.GetConsoleOutputInput, optFns ...func(*ec2.Options)) (*ec2.GetConsoleOutputOutput, error) {
	if m.GetConsoleOutputFunc != nil {
		return m.GetConsoleOutputFunc(ctx, params)
	}
	return &ec2.GetConsoleOutputOutput{}, nil
}

func (m *MockEC2Client) ModifyInstanceAttribute(ctx context.Context, params *ec2.ModifyInstanceAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyInstanceAttributeOutput, error) {
	if m.ModifyInstanceAttributeFunc != nil {
		return m.ModifyInstanceAttributeFunc(ctx, params)
	}
	return &ec2.ModifyInstanceAttributeOutput{}, nil
}

func (m *MockEC2Client) DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	if m.DeregisterImageFunc != nil {
		return m.DeregisterImageFunc(ctx, params)
	}
	return &ec2.DeregisterImageOutput{}, nil
}

func (m *MockEC2Client) DeleteSnapshot(ctx context.Context, params *ec2.DeleteSnapshotInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSnapshotOutput, error) {
	if m.DeleteSnapshotFunc != nil {
		return m.DeleteSnapshotFunc(ctx, params)
	}
	return &ec2.DeleteSnapshotOutput{}, nil
}

func (m *MockEC2Client) DescribeSnapshots(ctx context.Context, params *ec2.DescribeSnapshotsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSnapshotsOutput, error) {
	if m.DescribeSnapshotsFunc != nil {
		return m.DescribeSnapshotsFunc(ctx, params)
	}
	return &ec2.DescribeSnapshotsOutput{}, nil
}

func (m *MockEC2Client) ModifyImageAttribute(ctx context.Context, params *ec2.ModifyImageAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyImageAttributeOutput, error) {
	if m.ModifyImageAttributeFunc != nil {
		return m.ModifyImageAttributeFunc(ctx, params)
	}
	return &ec2.ModifyImageAttributeOutput{}, nil
}

func (m *MockEC2Client) CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error) {
	if m.CreateTagsFunc != nil {
		return m.CreateTagsFunc(ctx, params)
	}
	return &ec2.CreateTagsOutput{}, nil
}

// MockEFSClient provides a mock implementation of EFSClientInterface for testing
type MockEFSClient struct {
	DescribeMountTargetsFunc func(ctx context.Context, params *efs.DescribeMountTargetsInput) (*efs.DescribeMountTargetsOutput, error)
	DeleteMountTargetFunc    func(ctx context.Context, params *efs.DeleteMountTargetInput) (*efs.DeleteMountTargetOutput, error)
	DeleteFileSystemFunc     func(ctx context.Context, params *efs.DeleteFileSystemInput) (*efs.DeleteFileSystemOutput, error)
	CreateFileSystemFunc     func(ctx context.Context, params *efs.CreateFileSystemInput) (*efs.CreateFileSystemOutput, error)
	DescribeFileSystemsFunc  func(ctx context.Context, params *efs.DescribeFileSystemsInput) (*efs.DescribeFileSystemsOutput, error)
}

func (m *MockEFSClient) DescribeMountTargets(ctx context.Context, params *efs.DescribeMountTargetsInput, optFns ...func(*efs.Options)) (*efs.DescribeMountTargetsOutput, error) {
	if m.DescribeMountTargetsFunc != nil {
		return m.DescribeMountTargetsFunc(ctx, params)
	}
	return &efs.DescribeMountTargetsOutput{}, nil
}

func (m *MockEFSClient) DeleteMountTarget(ctx context.Context, params *efs.DeleteMountTargetInput, optFns ...func(*efs.Options)) (*efs.DeleteMountTargetOutput, error) {
	if m.DeleteMountTargetFunc != nil {
		return m.DeleteMountTargetFunc(ctx, params)
	}
	return &efs.DeleteMountTargetOutput{}, nil
}

func (m *MockEFSClient) DeleteFileSystem(ctx context.Context, params *efs.DeleteFileSystemInput, optFns ...func(*efs.Options)) (*efs.DeleteFileSystemOutput, error) {
	if m.DeleteFileSystemFunc != nil {
		return m.DeleteFileSystemFunc(ctx, params)
	}
	return &efs.DeleteFileSystemOutput{}, nil
}

func (m *MockEFSClient) CreateFileSystem(ctx context.Context, params *efs.CreateFileSystemInput, optFns ...func(*efs.Options)) (*efs.CreateFileSystemOutput, error) {
	if m.CreateFileSystemFunc != nil {
		return m.CreateFileSystemFunc(ctx, params)
	}
	return &efs.CreateFileSystemOutput{}, nil
}

func (m *MockEFSClient) DescribeFileSystems(ctx context.Context, params *efs.DescribeFileSystemsInput, optFns ...func(*efs.Options)) (*efs.DescribeFileSystemsOutput, error) {
	if m.DescribeFileSystemsFunc != nil {
		return m.DescribeFileSystemsFunc(ctx, params)
	}
	return &efs.DescribeFileSystemsOutput{}, nil
}

// MockSSMClient provides a mock implementation of SSMClientInterface for testing
type MockSSMClient struct {
	SendCommandFunc                 func(ctx context.Context, params *ssm.SendCommandInput) (*ssm.SendCommandOutput, error)
	GetCommandInvocationFunc        func(ctx context.Context, params *ssm.GetCommandInvocationInput) (*ssm.GetCommandInvocationOutput, error)
	DescribeInstanceInformationFunc func(ctx context.Context, params *ssm.DescribeInstanceInformationInput) (*ssm.DescribeInstanceInformationOutput, error)
	PutParameterFunc                func(ctx context.Context, params *ssm.PutParameterInput) (*ssm.PutParameterOutput, error)
	GetParameterFunc                func(ctx context.Context, params *ssm.GetParameterInput) (*ssm.GetParameterOutput, error)
	DeleteParameterFunc             func(ctx context.Context, params *ssm.DeleteParameterInput) (*ssm.DeleteParameterOutput, error)
	GetParametersByPathFunc         func(ctx context.Context, params *ssm.GetParametersByPathInput) (*ssm.GetParametersByPathOutput, error)
}

func (m *MockSSMClient) SendCommand(ctx context.Context, params *ssm.SendCommandInput, optFns ...func(*ssm.Options)) (*ssm.SendCommandOutput, error) {
	if m.SendCommandFunc != nil {
		return m.SendCommandFunc(ctx, params)
	}
	return &ssm.SendCommandOutput{}, nil
}

func (m *MockSSMClient) GetCommandInvocation(ctx context.Context, params *ssm.GetCommandInvocationInput, optFns ...func(*ssm.Options)) (*ssm.GetCommandInvocationOutput, error) {
	if m.GetCommandInvocationFunc != nil {
		return m.GetCommandInvocationFunc(ctx, params)
	}
	return &ssm.GetCommandInvocationOutput{}, nil
}

func (m *MockSSMClient) DescribeInstanceInformation(ctx context.Context, params *ssm.DescribeInstanceInformationInput, optFns ...func(*ssm.Options)) (*ssm.DescribeInstanceInformationOutput, error) {
	if m.DescribeInstanceInformationFunc != nil {
		return m.DescribeInstanceInformationFunc(ctx, params)
	}
	return &ssm.DescribeInstanceInformationOutput{}, nil
}

func (m *MockSSMClient) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	if m.PutParameterFunc != nil {
		return m.PutParameterFunc(ctx, params)
	}
	return &ssm.PutParameterOutput{}, nil
}

func (m *MockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.GetParameterFunc != nil {
		return m.GetParameterFunc(ctx, params)
	}
	return &ssm.GetParameterOutput{}, nil
}

func (m *MockSSMClient) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	if m.DeleteParameterFunc != nil {
		return m.DeleteParameterFunc(ctx, params)
	}
	return &ssm.DeleteParameterOutput{}, nil
}

func (m *MockSSMClient) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.GetParametersByPathFunc != nil {
		return m.GetParametersByPathFunc(ctx, params)
	}
	return &ssm.GetParametersByPathOutput{}, nil
}

// MockSTSClient provides a mock implementation of STSClientInterface for testing
type MockSTSClient struct {
	GetCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput) (*sts.GetCallerIdentityOutput, error)
}

func (m *MockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.GetCallerIdentityFunc != nil {
		return m.GetCallerIdentityFunc(ctx, params)
	}
	return &sts.GetCallerIdentityOutput{}, nil
}

// MockStateManager provides a mock implementation of StateManagerInterface for testing
type MockStateManager struct {
	LoadStateFunc       func() (*types.State, error)
	SaveStateFunc       func(*types.State) error
	SaveInstanceFunc    func(types.Instance) error
	RemoveInstanceFunc  func(string) error
	SaveVolumeFunc      func(types.EFSVolume) error
	RemoveVolumeFunc    func(string) error
	SaveEBSVolumeFunc   func(types.EBSVolume) error
	RemoveEBSVolumeFunc func(string) error
	UpdateConfigFunc    func(types.Config) error
}

func (m *MockStateManager) LoadState() (*types.State, error) {
	if m.LoadStateFunc != nil {
		return m.LoadStateFunc()
	}
	return &types.State{}, nil
}

func (m *MockStateManager) SaveState(state *types.State) error {
	if m.SaveStateFunc != nil {
		return m.SaveStateFunc(state)
	}
	return nil
}

func (m *MockStateManager) SaveInstance(instance types.Instance) error {
	if m.SaveInstanceFunc != nil {
		return m.SaveInstanceFunc(instance)
	}
	return nil
}

func (m *MockStateManager) RemoveInstance(name string) error {
	if m.RemoveInstanceFunc != nil {
		return m.RemoveInstanceFunc(name)
	}
	return nil
}

func (m *MockStateManager) SaveVolume(volume types.EFSVolume) error {
	if m.SaveVolumeFunc != nil {
		return m.SaveVolumeFunc(volume)
	}
	return nil
}

func (m *MockStateManager) RemoveVolume(name string) error {
	if m.RemoveVolumeFunc != nil {
		return m.RemoveVolumeFunc(name)
	}
	return nil
}

func (m *MockStateManager) SaveEBSVolume(volume types.EBSVolume) error {
	if m.SaveEBSVolumeFunc != nil {
		return m.SaveEBSVolumeFunc(volume)
	}
	return nil
}

func (m *MockStateManager) RemoveEBSVolume(name string) error {
	if m.RemoveEBSVolumeFunc != nil {
		return m.RemoveEBSVolumeFunc(name)
	}
	return nil
}

func (m *MockStateManager) UpdateConfig(config types.Config) error {
	if m.UpdateConfigFunc != nil {
		return m.UpdateConfigFunc(config)
	}
	return nil
}
