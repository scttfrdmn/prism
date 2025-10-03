package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock for EFS client
type mockEFSClientForVolume struct {
	mock.Mock
}

func (m *mockEFSClientForVolume) DescribeMountTargets(ctx context.Context, params *efs.DescribeMountTargetsInput, optFns ...func(*efs.Options)) (*efs.DescribeMountTargetsOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efs.DescribeMountTargetsOutput), args.Error(1)
}

func (m *mockEFSClientForVolume) DeleteMountTarget(ctx context.Context, params *efs.DeleteMountTargetInput, optFns ...func(*efs.Options)) (*efs.DeleteMountTargetOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efs.DeleteMountTargetOutput), args.Error(1)
}

func (m *mockEFSClientForVolume) DeleteFileSystem(ctx context.Context, params *efs.DeleteFileSystemInput, optFns ...func(*efs.Options)) (*efs.DeleteFileSystemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efs.DeleteFileSystemOutput), args.Error(1)
}

func (m *mockEFSClientForVolume) CreateFileSystem(ctx context.Context, params *efs.CreateFileSystemInput, optFns ...func(*efs.Options)) (*efs.CreateFileSystemOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efs.CreateFileSystemOutput), args.Error(1)
}

func (m *mockEFSClientForVolume) DescribeFileSystems(ctx context.Context, params *efs.DescribeFileSystemsInput, optFns ...func(*efs.Options)) (*efs.DescribeFileSystemsOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*efs.DescribeFileSystemsOutput), args.Error(1)
}

// Mock for state manager
type mockStateManagerForVolume struct {
	mock.Mock
}

func (m *mockStateManagerForVolume) LoadState() (*types.State, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.State), args.Error(1)
}

func (m *mockStateManagerForVolume) SaveState(state *types.State) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) SaveInstance(instance types.Instance) error {
	args := m.Called(instance)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) RemoveInstance(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) SaveVolume(volume types.EFSVolume) error {
	args := m.Called(volume)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) RemoveVolume(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) SaveEBSVolume(volume types.EBSVolume) error {
	args := m.Called(volume)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) RemoveEBSVolume(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *mockStateManagerForVolume) UpdateConfig(config types.Config) error {
	args := m.Called(config)
	return args.Error(0)
}

// Make sure mocks implement the interfaces
var (
	_ EFSClientInterface    = (*mockEFSClientForVolume)(nil)
	_ StateManagerInterface = (*mockStateManagerForVolume)(nil)
)

func TestVolumeDeletion(t *testing.T) {
	t.Run("successful deletion with mount targets", func(t *testing.T) {
		// Setup mock EFS client
		mockEfs := new(mockEFSClientForVolume)
		mockState := new(mockStateManagerForVolume)

		// Create manager with mocks
		manager := &Manager{
			efs:          mockEfs,
			stateManager: mockState,
		}

		// Setup state with volume
		testVolume := types.EFSVolume{
			Name:         "test-volume",
			FileSystemId: "fs-12345678",
		}
		mockState.On("LoadState").Return(&types.State{
			Volumes: map[string]types.EFSVolume{
				"test-volume": testVolume,
			},
		}, nil)

		// Setup mock responses
		mockEfs.On("DescribeMountTargets", mock.Anything, &efs.DescribeMountTargetsInput{
			FileSystemId: aws.String("fs-12345678"),
		}, mock.Anything).Return(&efs.DescribeMountTargetsOutput{
			MountTargets: []efsTypes.MountTargetDescription{
				{
					MountTargetId: aws.String("fsmt-11111111"),
				},
				{
					MountTargetId: aws.String("fsmt-22222222"),
				},
			},
		}, nil).Once()

		// Mock delete mount targets
		mockEfs.On("DeleteMountTarget", mock.Anything, &efs.DeleteMountTargetInput{
			MountTargetId: aws.String("fsmt-11111111"),
		}, mock.Anything).Return(&efs.DeleteMountTargetOutput{}, nil)
		mockEfs.On("DeleteMountTarget", mock.Anything, &efs.DeleteMountTargetInput{
			MountTargetId: aws.String("fsmt-22222222"),
		}, mock.Anything).Return(&efs.DeleteMountTargetOutput{}, nil)

		// Mock second check for mount targets (now empty)
		mockEfs.On("DescribeMountTargets", mock.Anything, &efs.DescribeMountTargetsInput{
			FileSystemId: aws.String("fs-12345678"),
		}, mock.Anything).Return(&efs.DescribeMountTargetsOutput{
			MountTargets: []efsTypes.MountTargetDescription{},
		}, nil).Once()

		// Mock delete file system
		mockEfs.On("DeleteFileSystem", mock.Anything, &efs.DeleteFileSystemInput{
			FileSystemId: aws.String("fs-12345678"),
		}, mock.Anything).Return(&efs.DeleteFileSystemOutput{}, nil)

		// Mock remove from state
		mockState.On("RemoveVolume", "test-volume").Return(nil)

		// Call the function under test
		err := manager.DeleteVolume("test-volume")

		// Verify results
		require.NoError(t, err)
		mockEfs.AssertExpectations(t)
		mockState.AssertExpectations(t)
	})

	t.Run("volume not found in state", func(t *testing.T) {
		// Setup mock state manager
		mockState := new(mockStateManagerForVolume)
		mockState.On("LoadState").Return(&types.State{
			Volumes: map[string]types.EFSVolume{},
		}, nil)

		// Create manager with mocks
		manager := &Manager{
			stateManager: mockState,
		}

		// Call the function under test
		err := manager.DeleteVolume("non-existent-volume")

		// Verify results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in state")
		mockState.AssertExpectations(t)
	})

	t.Run("error loading state", func(t *testing.T) {
		// Setup mock state manager
		mockState := new(mockStateManagerForVolume)
		mockState.On("LoadState").Return(nil, errors.New("state load error"))

		// Create manager with mocks
		manager := &Manager{
			stateManager: mockState,
		}

		// Call the function under test
		err := manager.DeleteVolume("test-volume")

		// Verify results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load state")
		mockState.AssertExpectations(t)
	})

	t.Run("error describing mount targets", func(t *testing.T) {
		// Setup mocks
		mockEfs := new(mockEFSClientForVolume)
		mockState := new(mockStateManagerForVolume)

		// Create manager with mocks
		manager := &Manager{
			efs:          mockEfs,
			stateManager: mockState,
		}

		// Setup state with volume
		testVolume := types.EFSVolume{
			Name:         "test-volume",
			FileSystemId: "fs-12345678",
		}
		mockState.On("LoadState").Return(&types.State{
			Volumes: map[string]types.EFSVolume{
				"test-volume": testVolume,
			},
		}, nil)

		// Setup mock error response
		mockEfs.On("DescribeMountTargets", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("describe mount targets error"))

		// Call the function under test
		err := manager.DeleteVolume("test-volume")

		// Verify results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list mount targets")
		mockEfs.AssertExpectations(t)
		mockState.AssertExpectations(t)
	})

	t.Run("error deleting mount target", func(t *testing.T) {
		// Setup mocks
		mockEfs := new(mockEFSClientForVolume)
		mockState := new(mockStateManagerForVolume)

		// Create manager with mocks
		manager := &Manager{
			efs:          mockEfs,
			stateManager: mockState,
		}

		// Setup state with volume
		testVolume := types.EFSVolume{
			Name:         "test-volume",
			FileSystemId: "fs-12345678",
		}
		mockState.On("LoadState").Return(&types.State{
			Volumes: map[string]types.EFSVolume{
				"test-volume": testVolume,
			},
		}, nil)

		// Setup mock responses
		mockEfs.On("DescribeMountTargets", mock.Anything, mock.Anything, mock.Anything).
			Return(&efs.DescribeMountTargetsOutput{
				MountTargets: []efsTypes.MountTargetDescription{
					{
						MountTargetId: aws.String("fsmt-11111111"),
					},
				},
			}, nil)

		// Mock delete mount target with error
		mockEfs.On("DeleteMountTarget", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("delete mount target error"))

		// Call the function under test
		err := manager.DeleteVolume("test-volume")

		// Verify results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete mount target")
		mockEfs.AssertExpectations(t)
		mockState.AssertExpectations(t)
	})

	t.Run("error deleting file system", func(t *testing.T) {
		// Setup mocks
		mockEfs := new(mockEFSClientForVolume)
		mockState := new(mockStateManagerForVolume)

		// Create manager with mocks
		manager := &Manager{
			efs:          mockEfs,
			stateManager: mockState,
		}

		// Setup state with volume
		testVolume := types.EFSVolume{
			Name:         "test-volume",
			FileSystemId: "fs-12345678",
		}
		mockState.On("LoadState").Return(&types.State{
			Volumes: map[string]types.EFSVolume{
				"test-volume": testVolume,
			},
		}, nil)

		// Setup mock responses - no mount targets
		mockEfs.On("DescribeMountTargets", mock.Anything, mock.Anything, mock.Anything).
			Return(&efs.DescribeMountTargetsOutput{
				MountTargets: []efsTypes.MountTargetDescription{},
			}, nil)

		// Mock delete file system with error
		mockEfs.On("DeleteFileSystem", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("delete file system error"))

		// Call the function under test
		err := manager.DeleteVolume("test-volume")

		// Verify results
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete file system")
		mockEfs.AssertExpectations(t)
		mockState.AssertExpectations(t)
	})
}
