// Package api provides the CloudWorkstation REST API client implementation.
package api

import (
	"context"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ClientOptions represents configuration options for the API client
type ClientOptions struct {
	AWSProfile      string
	AWSRegion       string
	InvitationToken string
	OwnerAccount    string
	S3ConfigPath    string
}


// CloudWorkstationAPI defines the interface for interacting with the CloudWorkstation API
type CloudWorkstationAPI interface {
	// Configuration
	SetOptions(ClientOptions)
	// Instance operations
	LaunchInstance(context.Context, types.LaunchRequest) (*types.LaunchResponse, error)
	ListInstances(context.Context) (*types.ListResponse, error)
	GetInstance(context.Context, string) (*types.Instance, error)
	StartInstance(context.Context, string) error
	StopInstance(context.Context, string) error
	DeleteInstance(context.Context, string) error
	ConnectInstance(context.Context, string) (string, error)

	// Template operations
	ListTemplates(context.Context) (map[string]types.Template, error)
	GetTemplate(context.Context, string) (*types.Template, error)

	// Volume operations (EFS)
	CreateVolume(context.Context, types.VolumeCreateRequest) (*types.EFSVolume, error)
	ListVolumes(context.Context) ([]types.EFSVolume, error)
	GetVolume(context.Context, string) (*types.EFSVolume, error)
	DeleteVolume(context.Context, string) error
	AttachVolume(context.Context, string, string) error
	DetachVolume(context.Context, string) error

	// Storage operations (EBS)
	CreateStorage(context.Context, types.StorageCreateRequest) (*types.EBSVolume, error)
	ListStorage(context.Context) ([]types.EBSVolume, error)
	GetStorage(context.Context, string) (*types.EBSVolume, error)
	DeleteStorage(context.Context, string) error
	AttachStorage(context.Context, string, string) error
	DetachStorage(context.Context, string) error

	// Status operations
	GetStatus(context.Context) (*types.DaemonStatus, error)
	Ping(context.Context) error
	
	// Registry operations
	GetRegistryStatus(context.Context) (*RegistryStatusResponse, error)
	SetRegistryStatus(context.Context, bool) error
	LookupAMI(context.Context, string, string, string) (*AMIReferenceResponse, error)
	ListTemplateAMIs(context.Context, string) ([]AMIReferenceResponse, error)
}