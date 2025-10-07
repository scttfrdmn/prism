package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Manager handles S3 mount point operations for CloudWorkstation
type S3Manager struct {
	s3Client *s3.Client
	region   string
}

// NewS3Manager creates a new S3 manager instance
func NewS3Manager(cfg aws.Config) *S3Manager {
	return &S3Manager{
		s3Client: s3.NewFromConfig(cfg),
		region:   cfg.Region,
	}
}

// CreateS3MountPoint creates a new S3 bucket for mounting
func (m *S3Manager) CreateS3MountPoint(req StorageRequest) (*StorageInfo, error) {
	ctx := context.Background()

	bucketName := req.Name
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// Add location constraint for regions other than us-east-1
	if m.region != "us-east-1" {
		createInput.CreateBucketConfiguration = &s3Types.CreateBucketConfiguration{
			LocationConstraint: s3Types.BucketLocationConstraint(m.region),
		}
	}

	_, err := m.s3Client.CreateBucket(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	// Apply bucket configuration for mounting
	err = m.configureBucketForMounting(ctx, bucketName, req.S3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure bucket: %w", err)
	}

	storageInfo := &StorageInfo{
		Name:         req.Name,
		Type:         StorageTypeS3,
		BucketName:   bucketName,
		State:        "available",
		Size:         0, // S3 buckets don't have a fixed size
		CreationTime: time.Now(),
		Region:       m.region,
		S3Config:     req.S3Config,
	}

	return storageInfo, nil
}

// ListS3MountPoints lists all CloudWorkstation S3 buckets
func (m *S3Manager) ListS3MountPoints() ([]StorageInfo, error) {
	ctx := context.Background()

	result, err := m.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}

	var storageInfos []StorageInfo
	for _, bucket := range result.Buckets {
		// Check if this is a CloudWorkstation bucket by checking tags
		isCloudWorkstationBucket, err := m.isCloudWorkstationBucket(ctx, *bucket.Name)
		if err != nil {
			// Log error but continue with other buckets
			continue
		}

		// Only include CloudWorkstation-managed buckets
		if !isCloudWorkstationBucket {
			continue
		}

		storageInfo := StorageInfo{
			Name:         *bucket.Name,
			Type:         StorageTypeS3,
			BucketName:   *bucket.Name,
			State:        "available",
			Size:         0, // S3 buckets don't have a fixed size
			CreationTime: *bucket.CreationDate,
			Region:       m.region,
		}
		storageInfos = append(storageInfos, storageInfo)
	}

	return storageInfos, nil
}

// DeleteS3MountPoint deletes an S3 bucket
func (m *S3Manager) DeleteS3MountPoint(name string) error {
	ctx := context.Background()

	// Empty the bucket first (required before deletion)
	err := m.emptyBucket(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to empty bucket: %w", err)
	}

	// Delete the bucket
	_, err = m.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(name),
	})
	if err != nil {
		return fmt.Errorf("failed to delete S3 bucket: %w", err)
	}

	return nil
}

// GetMountCommand generates the S3 mount command
func (m *S3Manager) GetMountCommand(bucketName string, mountPoint string, method S3MountMethod) string {
	switch method {
	case S3MountMethodS3FS:
		return fmt.Sprintf(`#!/bin/bash
# Install s3fs-fuse
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y s3fs-fuse
elif command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update && sudo apt-get install -y s3fs-fuse
fi

# Create mount point
sudo mkdir -p %s

# Mount S3 bucket
s3fs %s %s -o use_cache=/tmp -o allow_other -o uid=$(id -u) -o gid=$(id -g)

echo "S3 bucket %s mounted at %s"
`, mountPoint, bucketName, mountPoint, bucketName, mountPoint)

	case S3MountMethodGoofys:
		return fmt.Sprintf(`#!/bin/bash
# Install goofys
wget -O /tmp/goofys https://github.com/kahing/goofys/releases/latest/download/goofys
chmod +x /tmp/goofys
sudo mv /tmp/goofys /usr/local/bin/

# Create mount point
sudo mkdir -p %s

# Mount S3 bucket
goofys %s %s

echo "S3 bucket %s mounted at %s using goofys"
`, mountPoint, bucketName, mountPoint, bucketName, mountPoint)

	case S3MountMethodMountpoint:
		return fmt.Sprintf(`#!/bin/bash
# Install AWS Mountpoint for S3
wget -O /tmp/mountpoint-s3.rpm https://s3.amazonaws.com/mountpoint-s3-release/latest/x86_64/mountpoint-s3.rpm
sudo yum install -y /tmp/mountpoint-s3.rpm

# Create mount point
sudo mkdir -p %s

# Mount S3 bucket
mount-s3 %s %s

echo "S3 bucket %s mounted at %s using mountpoint-s3"
`, mountPoint, bucketName, mountPoint, bucketName, mountPoint)

	default:
		return "# Unknown S3 mount method"
	}
}

// GetInstallScript returns the S3 mounting tools installation script
func (m *S3Manager) GetInstallScript(method S3MountMethod) string {
	switch method {
	case S3MountMethodS3FS:
		return `#!/bin/bash
# Install s3fs-fuse
if command -v yum >/dev/null 2>&1; then
    sudo yum install -y s3fs-fuse
elif command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update && sudo apt-get install -y s3fs-fuse
fi
echo "s3fs-fuse installation complete"
`
	case S3MountMethodGoofys:
		return `#!/bin/bash
# Install goofys
wget -O /tmp/goofys https://github.com/kahing/goofys/releases/latest/download/goofys
chmod +x /tmp/goofys
sudo mv /tmp/goofys /usr/local/bin/
echo "goofys installation complete"
`
	default:
		return `#!/bin/bash
echo "S3 mount tools installation - method not specified"
`
	}
}

// configureBucketForMounting optimizes bucket configuration for mounting
func (m *S3Manager) configureBucketForMounting(ctx context.Context, bucketName string, config *S3Configuration) error {
	// Simplified configuration - bucket optimization features would be implemented here
	// Current implementation is a placeholder for future enhancements
	return nil
}

// emptyBucket removes all objects from a bucket
func (m *S3Manager) emptyBucket(ctx context.Context, bucketName string) error {
	// List and delete all objects - simplified implementation
	listResult, err := m.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return err
	}

	// Delete objects if any exist
	if len(listResult.Contents) > 0 {
		var objectIds []s3Types.ObjectIdentifier
		for _, obj := range listResult.Contents {
			objectIds = append(objectIds, s3Types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		_, err = m.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3Types.Delete{
				Objects: objectIds,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// isCloudWorkstationBucket checks if a bucket is managed by CloudWorkstation
func (m *S3Manager) isCloudWorkstationBucket(ctx context.Context, bucketName string) (bool, error) {
	// Get bucket tags
	taggingOutput, err := m.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// If error is NoSuchTagSet, bucket has no tags - not managed by CloudWorkstation
		return false, nil
	}

	// Check for CloudWorkstation tag
	for _, tag := range taggingOutput.TagSet {
		if tag.Key != nil && *tag.Key == "ManagedBy" {
			if tag.Value != nil && *tag.Value == "CloudWorkstation" {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetBucketMetrics retrieves S3 bucket metrics for analytics
func (m *S3Manager) GetBucketMetrics(bucketName string) (*S3Metrics, error) {
	ctx := context.Background()

	// Get bucket size and object count (simplified - real implementation would use CloudWatch)
	listResult, err := m.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list bucket objects: %w", err)
	}

	metrics := &S3Metrics{
		BucketName:  bucketName,
		ObjectCount: int64(len(listResult.Contents)),
		TotalSize:   0, // Would be calculated from object sizes
		LastUpdated: time.Now(),
	}

	// Calculate total size
	for _, obj := range listResult.Contents {
		if obj.Size != nil {
			metrics.TotalSize += *obj.Size
		}
	}

	return metrics, nil
}

// OptimizeBucketForWorkload optimizes S3 bucket for specific workloads
func (m *S3Manager) OptimizeBucketForWorkload(bucketName string, workload string) error {
	// Simplified implementation - placeholder for workload-specific optimizations
	return fmt.Errorf("S3 workload optimization not yet implemented in this version")
}

// Note: This is a simplified implementation for Phase 5C foundation.
// Full S3 integration with advanced mounting options, security configurations,
// and performance optimizations would be implemented in future iterations.
