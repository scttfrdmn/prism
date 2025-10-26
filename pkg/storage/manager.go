package storage

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// StorageManager is the main interface for Prism storage operations
type StorageManager struct {
	efsManager       *EFSManager
	ebsManager       *EBSManager
	fsxManager       *FSxManager
	s3Manager        *S3Manager
	analyticsManager *AnalyticsManager
	region           string
}

// NewStorageManager creates a new storage manager with all storage backends
func NewStorageManager(cfg aws.Config) *StorageManager {
	return &StorageManager{
		efsManager:       NewEFSManager(cfg),
		ebsManager:       NewEBSManager(cfg),
		fsxManager:       NewFSxManager(cfg),
		s3Manager:        NewS3Manager(cfg),
		analyticsManager: NewAnalyticsManager(cfg),
		region:           cfg.Region,
	}
}

// CreateStorage creates a storage resource of the specified type
func (m *StorageManager) CreateStorage(req StorageRequest) (*StorageInfo, error) {
	switch req.Type {
	case StorageTypeEFS:
		return m.efsManager.CreateEFSFilesystem(req)
	case StorageTypeEBS:
		return m.ebsManager.CreateEBSVolume(req)
	case StorageTypeFSx:
		return m.fsxManager.CreateFSxFilesystem(req)
	case StorageTypeS3:
		return m.s3Manager.CreateS3MountPoint(req)
	default:
		return nil, fmt.Errorf("unsupported storage type: %v", req.Type)
	}
}

// ListStorage lists all storage resources across all types
func (m *StorageManager) ListStorage() ([]StorageInfo, error) {
	var allStorage []StorageInfo

	// Get EFS filesystems
	efsStorage, err := m.efsManager.ListEFSFilesystems()
	if err != nil {
		return nil, fmt.Errorf("failed to list EFS storage: %w", err)
	}
	allStorage = append(allStorage, efsStorage...)

	// Get EBS volumes
	ebsStorage, err := m.ebsManager.ListEBSVolumes()
	if err != nil {
		return nil, fmt.Errorf("failed to list EBS storage: %w", err)
	}
	allStorage = append(allStorage, ebsStorage...)

	// Get FSx filesystems
	fsxStorage, err := m.fsxManager.ListFSxFilesystems()
	if err != nil {
		return nil, fmt.Errorf("failed to list FSx storage: %w", err)
	}
	allStorage = append(allStorage, fsxStorage...)

	// Get S3 mount points
	s3Storage, err := m.s3Manager.ListS3MountPoints()
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 storage: %w", err)
	}
	allStorage = append(allStorage, s3Storage...)

	return allStorage, nil
}

// ListStorageByType lists storage resources of a specific type
func (m *StorageManager) ListStorageByType(storageType StorageType) ([]StorageInfo, error) {
	switch storageType {
	case StorageTypeEFS:
		return m.efsManager.ListEFSFilesystems()
	case StorageTypeEBS:
		return m.ebsManager.ListEBSVolumes()
	case StorageTypeFSx:
		return m.fsxManager.ListFSxFilesystems()
	case StorageTypeS3:
		return m.s3Manager.ListS3MountPoints()
	default:
		return nil, fmt.Errorf("unsupported storage type: %v", storageType)
	}
}

// DeleteStorage deletes a storage resource by name and type
func (m *StorageManager) DeleteStorage(name string, storageType StorageType) error {
	switch storageType {
	case StorageTypeEFS:
		return m.efsManager.DeleteEFSFilesystem(name)
	case StorageTypeEBS:
		return m.ebsManager.DeleteEBSVolume(name)
	case StorageTypeFSx:
		return m.fsxManager.DeleteFSxFilesystem(name, true)
	case StorageTypeS3:
		return m.s3Manager.DeleteS3MountPoint(name)
	default:
		return fmt.Errorf("unsupported storage type: %v", storageType)
	}
}

// GetMountCommand generates the appropriate mount command for a storage resource
func (m *StorageManager) GetMountCommand(name string, storageType StorageType, mountPoint string) (string, error) {
	switch storageType {
	case StorageTypeEFS:
		storage, err := m.findEFSByName(name)
		if err != nil {
			return "", err
		}
		return m.efsManager.GetMountCommand(storage.FilesystemID, mountPoint), nil

	case StorageTypeEBS:
		storage, err := m.findEBSByName(name)
		if err != nil {
			return "", err
		}
		return m.ebsManager.GetMountCommand(storage.VolumeID, mountPoint, storage.EBSConfig), nil

	case StorageTypeFSx:
		storage, err := m.findFSxByName(name)
		if err != nil {
			return "", err
		}
		return m.fsxManager.GetMountCommand(FSxTypeLustre, storage.FilesystemID, mountPoint), nil

	case StorageTypeS3:
		storage, err := m.findS3ByName(name)
		if err != nil {
			return "", err
		}
		mountMethod := S3MountMethodS3FS // Default method
		if storage.S3Config != nil {
			mountMethod = storage.S3Config.MountMethod
		}
		return m.s3Manager.GetMountCommand(storage.BucketName, mountPoint, mountMethod), nil

	default:
		return "", fmt.Errorf("unsupported storage type: %v", storageType)
	}
}

// GetInstallScript generates installation scripts for storage mounting tools
func (m *StorageManager) GetInstallScript(storageType StorageType, options ...interface{}) (string, error) {
	switch storageType {
	case StorageTypeEFS:
		return m.efsManager.GetInstallScript(), nil

	case StorageTypeEBS:
		return m.ebsManager.GetInstallScript(), nil

	case StorageTypeFSx:
		if len(options) > 0 {
			if _, ok := options[0].(FSxFilesystemType); ok {
				return m.fsxManager.GetInstallScript(), nil
			}
		}
		return m.fsxManager.GetInstallScript(), nil // Default

	case StorageTypeS3:
		if len(options) > 0 {
			if s3Method, ok := options[0].(S3MountMethod); ok {
				return m.s3Manager.GetInstallScript(s3Method), nil
			}
		}
		return m.s3Manager.GetInstallScript(S3MountMethodS3FS), nil // Default

	default:
		return "", fmt.Errorf("unsupported storage type: %v", storageType)
	}
}

// GetStorageAnalytics retrieves comprehensive storage analytics
func (m *StorageManager) GetStorageAnalytics(period AnalyticsPeriod, resources []string) (*CostAnalysis, error) {
	// Convert resource names to StorageResource structs
	storageResources, err := m.resolveStorageResources(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage resources: %w", err)
	}

	// Calculate time period
	endTime := time.Now()
	var startTime time.Time
	switch period {
	case AnalyticsPeriodDaily:
		startTime = endTime.AddDate(0, 0, -1)
	case AnalyticsPeriodWeekly:
		startTime = endTime.AddDate(0, 0, -7)
	case AnalyticsPeriodMonthly:
		startTime = endTime.AddDate(0, -1, 0)
	case AnalyticsPeriodYearly:
		startTime = endTime.AddDate(-1, 0, 0)
	default:
		startTime = endTime.AddDate(0, 0, -7) // Default to weekly
	}

	analyticsReq := AnalyticsRequest{
		Resources: storageResources,
		StartTime: startTime,
		EndTime:   endTime,
		Period:    period,
	}

	return m.analyticsManager.GetStorageAnalytics(analyticsReq)
}

// GetUsagePatterns analyzes storage usage patterns for optimization
func (m *StorageManager) GetUsagePatterns(resources []string, days int) (*UsagePatternAnalysis, error) {
	// Resolve storage resources by listing all available storage
	allStorage, err := m.ListStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage resources: %w", err)
	}

	// If specific resources requested, filter to those
	storageMap := make(map[string]StorageInfo)
	for _, storage := range allStorage {
		storageMap[storage.Name] = storage
	}

	// If no specific resources requested, analyze all
	if len(resources) == 0 {
		resources = make([]string, 0, len(allStorage))
		for _, storage := range allStorage {
			resources = append(resources, storage.Name)
		}
	}

	// Create analytics request
	analyticsReq := AnalyticsRequest{
		StartTime: time.Now().AddDate(0, 0, -days),
		EndTime:   time.Now(),
	}

	usageAnalysis, err := m.analyticsManager.GetUsagePatternAnalysis(analyticsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage pattern analysis: %w", err)
	}

	// Convert UsageAnalysis to UsagePatternAnalysis with REAL data
	resourcePatterns := make(map[string]ResourceUsagePattern)

	for _, pattern := range usageAnalysis.Patterns {
		// Skip if this resource wasn't requested
		if len(resources) > 0 {
			found := false
			for _, reqResource := range resources {
				if pattern.Resource == reqResource {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Get storage info for this resource
		storageInfo, exists := storageMap[pattern.Resource]
		if !exists {
			continue
		}

		// Create resource usage pattern with real data
		resourcePattern := ResourceUsagePattern{
			ResourceName:     pattern.Resource,
			ResourceType:     storageInfo.Type,
			DataPoints:       make([]UsageDataPoint, 0),
			PeakUsageHours:   m.calculatePeakUsageHours(pattern),
			UsageVariability: pattern.Confidence, // Use confidence as variability metric
			TrendDirection:   pattern.Pattern,    // Pattern becomes trend
		}

		resourcePatterns[pattern.Resource] = resourcePattern
	}

	return &UsagePatternAnalysis{
		AnalysisPeriod:         fmt.Sprintf("%d days", days),
		ResourcePatterns:       resourcePatterns,
		PatternRecommendations: usageAnalysis.Recommendations,
	}, nil
}

// calculatePeakUsageHours determines peak usage hours from usage pattern
func (m *StorageManager) calculatePeakUsageHours(pattern UsagePattern) []int {
	// Parse pattern description to identify peak hours
	// Patterns typically describe usage like "high-usage-weekdays" or "consistent-24x7"

	switch pattern.Pattern {
	case "high-usage-weekdays":
		// Business hours: 9 AM - 5 PM
		return []int{9, 10, 11, 12, 13, 14, 15, 16, 17}
	case "high-usage-nights":
		// Night hours: 6 PM - 2 AM
		return []int{18, 19, 20, 21, 22, 23, 0, 1, 2}
	case "consistent-24x7":
		// All hours
		return []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	case "sporadic":
		// No consistent peak hours
		return []int{}
	default:
		// Default to business hours if unknown pattern
		return []int{9, 10, 11, 12, 13, 14, 15, 16, 17}
	}
}

// OptimizeStorageForWorkload applies workload-specific optimizations
func (m *StorageManager) OptimizeStorageForWorkload(name string, storageType StorageType, workload WorkloadType) error {
	switch storageType {
	case StorageTypeEFS:
		return m.optimizeEFSForWorkload(name, workload)
	case StorageTypeEBS:
		return m.optimizeEBSForWorkload(name, workload)
	case StorageTypeFSx:
		return m.optimizeFSxForWorkload(name, workload)
	case StorageTypeS3:
		return m.optimizeS3ForWorkload(name, workload)
	default:
		return fmt.Errorf("unsupported storage type: %v", storageType)
	}
}

// GetStorageRecommendations provides optimization recommendations for all storage
func (m *StorageManager) GetStorageRecommendations() ([]OptimizationRecommendation, error) {
	// Get analytics for the last 30 days
	allStorage, err := m.ListStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage: %w", err)
	}

	var resourceNames []string
	for _, storage := range allStorage {
		resourceNames = append(resourceNames, storage.Name)
	}

	analytics, err := m.GetStorageAnalytics(AnalyticsPeriodMonthly, resourceNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	// Convert string recommendations to OptimizationRecommendation structs
	var recommendations []OptimizationRecommendation
	for _, rec := range analytics.Recommendations {
		recommendations = append(recommendations, OptimizationRecommendation{
			Type:             OptimizationTypeCost,
			Priority:         OptimizationPriorityMedium,
			Resource:         "General",
			Title:            "Storage Optimization",
			Description:      rec,
			PotentialSavings: 0.0, // Would be calculated from actual analysis
		})
	}
	return recommendations, nil
}

// AttachStorageToInstance attaches storage to an EC2 instance
func (m *StorageManager) AttachStorageToInstance(storageName string, storageType StorageType, instanceID string) error {
	switch storageType {
	case StorageTypeEBS:
		return m.ebsManager.AttachEBSVolume(storageName, instanceID)
	case StorageTypeEFS:
		// EFS doesn't need attachment - it's network-based
		return nil
	case StorageTypeFSx:
		// FSx doesn't need attachment - it's network-based
		return nil
	case StorageTypeS3:
		// S3 doesn't need attachment - it's API-based
		return nil
	default:
		return fmt.Errorf("unsupported storage type for attachment: %v", storageType)
	}
}

// DetachStorageFromInstance detaches storage from an EC2 instance
func (m *StorageManager) DetachStorageFromInstance(storageName string, storageType StorageType, instanceID string) error {
	switch storageType {
	case StorageTypeEBS:
		return m.ebsManager.DetachEBSVolume(storageName, instanceID)
	case StorageTypeEFS:
		return nil // EFS doesn't need detachment
	case StorageTypeFSx:
		return nil // FSx doesn't need detachment
	case StorageTypeS3:
		return nil // S3 doesn't need detachment
	default:
		return fmt.Errorf("unsupported storage type for detachment: %v", storageType)
	}
}

// Helper methods for finding storage resources

func (m *StorageManager) findEFSByName(name string) (*StorageInfo, error) {
	efsStorages, err := m.efsManager.ListEFSFilesystems()
	if err != nil {
		return nil, err
	}

	for _, storage := range efsStorages {
		if storage.Name == name {
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("EFS storage not found: %s", name)
}

func (m *StorageManager) findEBSByName(name string) (*StorageInfo, error) {
	ebsStorages, err := m.ebsManager.ListEBSVolumes()
	if err != nil {
		return nil, err
	}

	for _, storage := range ebsStorages {
		if storage.Name == name {
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("EBS storage not found: %s", name)
}

func (m *StorageManager) findFSxByName(name string) (*StorageInfo, error) {
	fsxStorages, err := m.fsxManager.ListFSxFilesystems()
	if err != nil {
		return nil, err
	}

	for _, storage := range fsxStorages {
		if storage.Name == name {
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("FSx storage not found: %s", name)
}

func (m *StorageManager) findS3ByName(name string) (*StorageInfo, error) {
	s3Storages, err := m.s3Manager.ListS3MountPoints()
	if err != nil {
		return nil, err
	}

	for _, storage := range s3Storages {
		if storage.Name == name {
			return &storage, nil
		}
	}
	return nil, fmt.Errorf("S3 storage not found: %s", name)
}

func (m *StorageManager) resolveStorageResources(names []string) ([]StorageResource, error) {
	var resources []StorageResource

	allStorage, err := m.ListStorage()
	if err != nil {
		return nil, err
	}

	// If no names specified, use all storage
	if len(names) == 0 {
		for _, storage := range allStorage {
			resources = append(resources, StorageResource{
				Name:       storage.Name,
				Type:       storage.Type,
				ResourceID: m.getResourceID(storage),
			})
		}
		return resources, nil
	}

	// Resolve specific storage names
	for _, name := range names {
		found := false
		for _, storage := range allStorage {
			if storage.Name == name {
				resources = append(resources, StorageResource{
					Name:       storage.Name,
					Type:       storage.Type,
					ResourceID: m.getResourceID(storage),
				})
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("storage resource not found: %s", name)
		}
	}

	return resources, nil
}

func (m *StorageManager) getResourceID(storage StorageInfo) string {
	switch storage.Type {
	case StorageTypeEFS:
		return storage.FilesystemID
	case StorageTypeEBS:
		return storage.VolumeID
	case StorageTypeFSx:
		return storage.FilesystemID
	case StorageTypeS3:
		return storage.BucketName
	default:
		return storage.Name
	}
}

// Workload optimization methods

func (m *StorageManager) optimizeEFSForWorkload(name string, workload WorkloadType) error {
	storage, err := m.findEFSByName(name)
	if err != nil {
		return err
	}

	// Apply workload-specific EFS optimizations
	switch workload {
	case WorkloadTypeML:
		return m.efsManager.OptimizeForWorkload(storage.FilesystemID, EFSWorkloadML)
	case WorkloadTypeBigData:
		return m.efsManager.OptimizeForWorkload(storage.FilesystemID, EFSWorkloadBigData)
	case WorkloadTypeGeneral:
		return m.efsManager.OptimizeForWorkload(storage.FilesystemID, EFSWorkloadGeneral)
	default:
		return fmt.Errorf("unsupported workload type for EFS: %v", workload)
	}
}

func (m *StorageManager) optimizeEBSForWorkload(name string, workload WorkloadType) error {
	storage, err := m.findEBSByName(name)
	if err != nil {
		return err
	}

	// Apply workload-specific EBS optimizations
	switch workload {
	case WorkloadTypeML:
		return m.ebsManager.OptimizeForWorkload(storage.VolumeID, EBSWorkloadML)
	case WorkloadTypeBigData:
		return m.ebsManager.OptimizeForWorkload(storage.VolumeID, EBSWorkloadBigData)
	case WorkloadTypeGeneral:
		return m.ebsManager.OptimizeForWorkload(storage.VolumeID, EBSWorkloadGeneral)
	default:
		return fmt.Errorf("unsupported workload type for EBS: %v", workload)
	}
}

func (m *StorageManager) optimizeFSxForWorkload(name string, workload WorkloadType) error {
	storage, err := m.findFSxByName(name)
	if err != nil {
		return err
	}

	// Apply workload-specific FSx optimizations
	switch workload {
	case WorkloadTypeHPC:
		return m.fsxManager.OptimizeForWorkload(storage.FilesystemID, "hpc")
	case WorkloadTypeBigData:
		return m.fsxManager.OptimizeForWorkload(storage.FilesystemID, "bigdata")
	case WorkloadTypeGeneral:
		return m.fsxManager.OptimizeForWorkload(storage.FilesystemID, "general")
	default:
		return fmt.Errorf("unsupported workload type for FSx: %v", workload)
	}
}

func (m *StorageManager) optimizeS3ForWorkload(name string, workload WorkloadType) error {
	storage, err := m.findS3ByName(name)
	if err != nil {
		return err
	}

	// Apply workload-specific S3 optimizations
	switch workload {
	case WorkloadTypeBigData:
		return m.s3Manager.OptimizeBucketForWorkload(storage.BucketName, "bigdata")
	case WorkloadTypeArchival:
		return m.s3Manager.OptimizeBucketForWorkload(storage.BucketName, "archival")
	case WorkloadTypeGeneral:
		return m.s3Manager.OptimizeBucketForWorkload(storage.BucketName, "frequent_access")
	default:
		return fmt.Errorf("unsupported workload type for S3: %v", workload)
	}
}

// Batch operations

// CreateMultiTierStorage creates a multi-tier storage setup for complex workloads
func (m *StorageManager) CreateMultiTierStorage(name string, config MultiTierStorageConfig) (*MultiTierStorageInfo, error) {
	info := &MultiTierStorageInfo{
		Name:         name,
		CreationTime: time.Now(),
		Tiers:        make(map[string]StorageInfo),
	}

	// Create hot tier (fast access)
	if config.HotTier != nil {
		hotStorage, err := m.CreateStorage(*config.HotTier)
		if err != nil {
			return nil, fmt.Errorf("failed to create hot tier: %w", err)
		}
		info.Tiers["hot"] = *hotStorage
	}

	// Create warm tier (balanced)
	if config.WarmTier != nil {
		warmStorage, err := m.CreateStorage(*config.WarmTier)
		if err != nil {
			return nil, fmt.Errorf("failed to create warm tier: %w", err)
		}
		info.Tiers["warm"] = *warmStorage
	}

	// Create cold tier (archival)
	if config.ColdTier != nil {
		coldStorage, err := m.CreateStorage(*config.ColdTier)
		if err != nil {
			return nil, fmt.Errorf("failed to create cold tier: %w", err)
		}
		info.Tiers["cold"] = *coldStorage
	}

	return info, nil
}

// GetStorageHealth performs health checks across all storage types
func (m *StorageManager) GetStorageHealth() (*StorageHealthReport, error) {
	report := &StorageHealthReport{
		Timestamp:     time.Now(),
		OverallHealth: "healthy",
		ServiceHealth: make(map[string]string),
		Issues:        []StorageHealthIssue{},
	}

	// Check health of each storage service
	m.checkEFSHealth(report)
	m.checkEBSHealth(report)
	m.checkFSxHealth(report)
	m.checkS3Health(report)

	// Determine overall health based on issues
	m.determineOverallHealth(report)

	return report, nil
}

// checkEFSHealth checks EFS filesystem health
func (m *StorageManager) checkEFSHealth(report *StorageHealthReport) {
	efsStorages, err := m.efsManager.ListEFSFilesystems()
	if err != nil {
		report.ServiceHealth["EFS"] = "unhealthy"
		report.Issues = append(report.Issues, StorageHealthIssue{
			Service:   "EFS",
			Severity:  "high",
			Message:   fmt.Sprintf("Failed to list EFS filesystems: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	report.ServiceHealth["EFS"] = "healthy"
	for _, storage := range efsStorages {
		if storage.State != "available" {
			report.Issues = append(report.Issues, StorageHealthIssue{
				Service:   "EFS",
				Resource:  storage.Name,
				Severity:  "medium",
				Message:   fmt.Sprintf("EFS filesystem %s is in state: %s", storage.Name, storage.State),
				Timestamp: time.Now(),
			})
		}
	}
}

// checkEBSHealth checks EBS volume health
func (m *StorageManager) checkEBSHealth(report *StorageHealthReport) {
	ebsStorages, err := m.ebsManager.ListEBSVolumes()
	if err != nil {
		report.ServiceHealth["EBS"] = "unhealthy"
		report.Issues = append(report.Issues, StorageHealthIssue{
			Service:   "EBS",
			Severity:  "high",
			Message:   fmt.Sprintf("Failed to list EBS volumes: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	report.ServiceHealth["EBS"] = "healthy"
	for _, storage := range ebsStorages {
		if !m.isEBSStateHealthy(storage.State) {
			report.Issues = append(report.Issues, StorageHealthIssue{
				Service:   "EBS",
				Resource:  storage.Name,
				Severity:  "medium",
				Message:   fmt.Sprintf("EBS volume %s is in state: %s", storage.Name, storage.State),
				Timestamp: time.Now(),
			})
		}
	}
}

// checkFSxHealth checks FSx filesystem health
func (m *StorageManager) checkFSxHealth(report *StorageHealthReport) {
	fsxStorages, err := m.fsxManager.ListFSxFilesystems()
	if err != nil {
		report.ServiceHealth["FSx"] = "unhealthy"
		report.Issues = append(report.Issues, StorageHealthIssue{
			Service:   "FSx",
			Severity:  "high",
			Message:   fmt.Sprintf("Failed to list FSx filesystems: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	report.ServiceHealth["FSx"] = "healthy"
	for _, storage := range fsxStorages {
		if storage.State != "AVAILABLE" {
			report.Issues = append(report.Issues, StorageHealthIssue{
				Service:   "FSx",
				Resource:  storage.Name,
				Severity:  "medium",
				Message:   fmt.Sprintf("FSx filesystem %s is in state: %s", storage.Name, storage.State),
				Timestamp: time.Now(),
			})
		}
	}
}

// checkS3Health checks S3 mount point health
func (m *StorageManager) checkS3Health(report *StorageHealthReport) {
	s3Storages, err := m.s3Manager.ListS3MountPoints()
	if err != nil {
		report.ServiceHealth["S3"] = "unhealthy"
		report.Issues = append(report.Issues, StorageHealthIssue{
			Service:   "S3",
			Severity:  "high",
			Message:   fmt.Sprintf("Failed to list S3 buckets: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	report.ServiceHealth["S3"] = "healthy"
	for _, storage := range s3Storages {
		if storage.State != "available" {
			report.Issues = append(report.Issues, StorageHealthIssue{
				Service:   "S3",
				Severity:  "medium",
				Message:   fmt.Sprintf("S3 storage %s is in %s state", storage.Name, storage.State),
				Timestamp: time.Now(),
			})
		}
	}
}

// isEBSStateHealthy checks if EBS volume state is healthy
func (m *StorageManager) isEBSStateHealthy(state string) bool {
	return state == "available" || state == "in-use"
}

// determineOverallHealth sets overall health based on issue severity
func (m *StorageManager) determineOverallHealth(report *StorageHealthReport) {
	if len(report.Issues) == 0 {
		return
	}

	highSeverityCount := 0
	for _, issue := range report.Issues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}

	if highSeverityCount > 0 {
		report.OverallHealth = "unhealthy"
	} else {
		report.OverallHealth = "degraded"
	}
}
