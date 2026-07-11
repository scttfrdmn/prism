package launch

import (
	spawn "github.com/spore-host/spawn/pkg/aws"
)

// Inputs carries the already-resolved values Prism computes during its launch
// pipeline (template extraction, networking resolution, user-data encoding, IAM
// profile check) into the pure ToLaunchConfig mapping. Everything here is
// resolved upstream in pkg/aws so that ToLaunchConfig stays free of AWS calls
// and side effects, and can be unit-tested without a client.
//
// Phase 1 deliberately maps only the "what to launch" fields that reproduce
// today's RunInstances behavior. spawn's own lifecycle fields (TTL, IdleTimeout,
// DNSName, HibernateOnIdle, Slack…) are intentionally left unset: those values
// travel as prism:-namespaced entries in Tags so the already-deployed spored
// (SPORED_TAG_PREFIX=prism) keeps reading them unchanged. See the Phase 2
// tag-namespace note in the plan.
type Inputs struct {
	InstanceType string
	Region       string
	AMI          string

	// Networking, resolved by NetworkingResolver.
	SubnetID        string
	SecurityGroupID string
	KeyName         string

	// UserDataEncoded is the gzip-compressed, base64-encoded cloud-init script.
	// spawn passes UserData verbatim to RunInstances, so this encoding is
	// preserved end to end.
	UserDataEncoded string

	// PrimaryUsername is the template's primary Linux user.
	PrimaryUsername string

	// RootVolumeGB is the resolved root EBS size (template default already applied).
	RootVolumeGB int

	// Hibernate reflects req.IdlePolicy after the idle/spot-conflict and
	// hibernation-support validations have already passed in pkg/aws.
	Hibernate bool

	// Spot requests a one-time spot instance. Phase 1 sets no max price to match
	// today's on-demand-capped behavior; SpotMaxPrice is a Phase 3 opt-in.
	Spot bool

	// CapacityReservationID targets a pre-reserved EC2 Capacity Block (req.CapacityBlockID).
	CapacityReservationID string

	// IAMInstanceProfile is the resolved instance-profile name, or "" when the
	// profile does not exist (the check runs in pkg/aws before mapping).
	IAMInstanceProfile string

	// HourlyRate is the true per-hour compute rate, threaded through for
	// spawn's spawn:price-per-hour tag and Prism's cost seed.
	HourlyRate float64

	// Name is the workspace/instance name.
	Name string

	// Tags is the fully-built EC2 tag set (including the prism:-namespaced
	// lifecycle tags spored reads), constructed in pkg/aws where version,
	// current user, and test-mode are known. spawn appends these verbatim.
	Tags map[string]string
}

// ToLaunchConfig maps resolved Prism launch inputs into a spawn.LaunchConfig.
//
// It is a pure function: given the same Inputs it always returns the same
// config, with no AWS calls, clock reads, or environment lookups. Those impure
// concerns are resolved upstream and passed in via Inputs (notably Tags).
func ToLaunchConfig(in Inputs) spawn.LaunchConfig {
	cfg := spawn.LaunchConfig{
		InstanceType:       in.InstanceType,
		Region:             in.Region,
		AMI:                in.AMI,
		SubnetID:           in.SubnetID,
		KeyName:            in.KeyName,
		UserData:           in.UserDataEncoded,
		Username:           in.PrimaryUsername,
		RootVolumeSizeGiB:  rootVolumeInt32(in.RootVolumeGB),
		Hibernate:          in.Hibernate,
		Spot:               in.Spot,
		ReservationID:      in.CapacityReservationID,
		IamInstanceProfile: in.IAMInstanceProfile,
		PricePerHour:       in.HourlyRate,
		Name:               in.Name,
		Tags:               in.Tags,
	}

	// Pre-resolved SG: spawn only creates a default SG when this is empty, which
	// Prism never wants — networking is always resolved upstream.
	if in.SecurityGroupID != "" {
		cfg.SecurityGroupIDs = []string{in.SecurityGroupID}
	}

	return cfg
}

// rootVolumeInt32 converts a root-volume size (GiB) to the int32 spawn expects,
// clamping to a sane range. EBS root volumes are far below int32's ceiling; the
// clamp exists only to make the narrowing conversion provably safe.
func rootVolumeInt32(gb int) int32 {
	const maxEBSVolumeGiB = 65536 // 64 TiB, AWS's gp3 maximum
	if gb <= 0 {
		return 0
	}
	if gb > maxEBSVolumeGiB {
		return maxEBSVolumeGiB
	}
	return int32(gb)
}
