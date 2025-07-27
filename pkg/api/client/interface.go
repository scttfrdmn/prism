// Package client provides the CloudWorkstation API client interface and implementation.
package client

import (
	"context"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Options represents configuration options for the API client
type Options struct {
	AWSProfile      string
	AWSRegion       string
	InvitationToken string
	OwnerAccount    string
	S3ConfigPath    string
}

// CloudWorkstationAPI defines the interface for interacting with the CloudWorkstation API
type CloudWorkstationAPI interface {
	// Configuration
	SetOptions(Options)
	
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

// Registry-specific response types for API operations

// RegistryStatusResponse represents the response from GetRegistryStatus
type RegistryStatusResponse struct {
	// Active indicates if the registry is active
	Active bool `json:"active"`
	
	// LastSync is when the registry was last synchronized
	LastSync *time.Time `json:"last_sync,omitempty"`
	
	// TemplateCount is the number of templates in the registry
	TemplateCount int `json:"template_count"`
	
	// AMICount is the total number of AMIs in the registry
	AMICount int `json:"ami_count"`
	
	// Status provides additional status information
	Status string `json:"status"`
}

// AMIReferenceResponse represents an AMI reference response
type AMIReferenceResponse struct {
	// AMIID is the AMI identifier
	AMIID string `json:"ami_id"`
	
	// Region is the AWS region where the AMI is located
	Region string `json:"region"`
	
	// Architecture is the AMI architecture (x86_64 or arm64)
	Architecture string `json:"architecture"`
	
	// TemplateName is the name of the template this AMI was built from
	TemplateName string `json:"template_name"`
	
	// Version is the semantic version of the template
	Version string `json:"version"`
	
	// BuildDate is when the AMI was built
	BuildDate time.Time `json:"build_date"`
	
	// Status is the current status of the AMI
	Status string `json:"status"`
	
	// Tags contains metadata tags for the AMI
	Tags map[string]string `json:"tags,omitempty"`
}