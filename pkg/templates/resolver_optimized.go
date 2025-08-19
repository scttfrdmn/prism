package templates

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OptimizedResolver provides parallel template processing for improved performance
type OptimizedResolver struct {
	*TemplateResolver
	
	// Caching for performance
	templateCache sync.Map // map[string]*CachedTemplate
	amiCache      sync.Map // map[string]*AMIMapping
	
	// Configuration
	cacheTimeout    time.Duration
	maxConcurrency  int
}

// CachedTemplate represents a cached template with metadata
type CachedTemplate struct {
	Template  *Template
	CachedAt  time.Time
	Hash      string
}

// CachedAMIMapping represents a cached AMI mapping
type CachedAMIMapping struct {
	Mapping   map[string]map[string]string // region -> arch -> AMI ID
	CachedAt  time.Time
	Region    string
	Arch      string
}

// NewOptimizedResolver creates an optimized template resolver with caching
func NewOptimizedResolver() *OptimizedResolver {
	return &OptimizedResolver{
		TemplateResolver: NewTemplateResolver(),
		cacheTimeout:     5 * time.Minute,
		maxConcurrency:   4,
	}
}

// ResolveTemplateOptimized resolves template with parallel processing and caching
func (r *OptimizedResolver) ResolveTemplateOptimized(ctx context.Context, template *Template, region, architecture, packageManagerOverride, size string) (*RuntimeTemplate, error) {
	// Create context with timeout for the entire operation
	resolveCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	
	// Select package manager (use override if provided)
	var packageManager PackageManagerType
	if packageManagerOverride != "" {
		packageManager = PackageManagerType(packageManagerOverride)
	} else {
		packageManager = r.Parser.Strategy.SelectPackageManager(template)
	}
	
	// Parallel processing channels
	type scriptResult struct {
		script string
		err    error
	}
	type amiResult struct {
		mapping map[string]map[string]string
		err     error
	}
	type mappingResult struct {
		instanceType map[string]string
		ports        []int
		cost         map[string]float64
		idleConfig   *IdleDetectionConfig
	}
	
	scriptChan := make(chan scriptResult, 1)
	amiChan := make(chan amiResult, 1)
	mappingChan := make(chan mappingResult, 1)
	
	// Start parallel goroutines
	go func() {
		script, err := r.generateScriptAsync(template, packageManager)
		scriptChan <- scriptResult{script: script, err: err}
	}()
	
	go func() {
		mapping, err := r.getAMIMappingAsync(resolveCtx, template, region, architecture)
		amiChan <- amiResult{mapping: mapping, err: err}
	}()
	
	go func() {
		instanceType := r.TemplateResolver.getInstanceTypeMapping(template, architecture, size)
		ports := r.TemplateResolver.getPortMapping(template)
		cost := r.TemplateResolver.getCostMapping(template, architecture)
		idleConfig := r.TemplateResolver.ensureIdleDetectionConfig(template)
		
		mappingChan <- mappingResult{
			instanceType: instanceType,
			ports:        ports,
			cost:         cost,
			idleConfig:   idleConfig,
		}
	}()
	
	// Collect results
	var userDataScript string
	var amiMapping map[string]map[string]string
	var instanceTypeMapping map[string]string
	var ports []int
	var costMapping map[string]float64
	var idleDetectionConfig *IdleDetectionConfig
	
	for i := 0; i < 3; i++ {
		select {
		case result := <-scriptChan:
			if result.err != nil {
				return nil, fmt.Errorf("failed to generate installation script: %w", result.err)
			}
			userDataScript = result.script
			
		case result := <-amiChan:
			if result.err != nil {
				return nil, fmt.Errorf("failed to get AMI mapping: %w", result.err)
			}
			amiMapping = result.mapping
			
		case result := <-mappingChan:
			instanceTypeMapping = result.instanceType
			ports = result.ports
			costMapping = result.cost
			idleDetectionConfig = result.idleConfig
			
		case <-resolveCtx.Done():
			return nil, fmt.Errorf("template resolution timed out: %w", resolveCtx.Err())
		}
	}
	
	// Create runtime template
	runtimeTemplate := &RuntimeTemplate{
		Name:                 template.Name,
		Slug:                 template.Slug,
		Description:          template.Description,
		LongDescription:      template.LongDescription,
		AMI:                  amiMapping,
		InstanceType:         instanceTypeMapping,
		UserData:             userDataScript,
		Ports:                ports,
		EstimatedCostPerHour: costMapping,
		IdleDetection:        idleDetectionConfig,
	}
	
	return runtimeTemplate, nil
}

// generateScriptAsync generates installation script asynchronously with caching
func (r *OptimizedResolver) generateScriptAsync(template *Template, packageManager PackageManagerType) (string, error) {
	// Use direct UserData if provided
	if template.UserData != "" {
		return r.ensureIdleDetection(template.UserData, template, packageManager), nil
	}
	
	// Check cache first
	cacheKey := fmt.Sprintf("script_%s_%s", template.Slug, string(packageManager))
	if cached, ok := r.templateCache.Load(cacheKey); ok {
		cachedTemplate := cached.(*CachedTemplate)
		if time.Since(cachedTemplate.CachedAt) < r.cacheTimeout {
			return cachedTemplate.Template.UserData, nil
		}
	}
	
	// Generate script
	generatedScript, err := r.ScriptGen.GenerateScript(template, packageManager)
	if err != nil {
		return "", err
	}
	
	// Ensure idle detection is present
	userDataScript := r.ensureIdleDetection(generatedScript, template, packageManager)
	
	// Cache the result
	r.templateCache.Store(cacheKey, &CachedTemplate{
		Template: &Template{UserData: userDataScript},
		CachedAt: time.Now(),
		Hash:     r.hashTemplate(template),
	})
	
	return userDataScript, nil
}

// getAMIMappingAsync gets AMI mapping asynchronously with caching
func (r *OptimizedResolver) getAMIMappingAsync(ctx context.Context, template *Template, region, architecture string) (map[string]map[string]string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("ami_%s_%s_%s", template.Base, region, architecture)
	if cached, ok := r.amiCache.Load(cacheKey); ok {
		cachedAMI := cached.(*CachedAMIMapping)
		if time.Since(cachedAMI.CachedAt) < r.cacheTimeout {
			return cachedAMI.Mapping, nil
		}
	}
	
	// Get AMI mapping
	amiMapping, err := r.TemplateResolver.getAMIMapping(template, region, architecture)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	r.amiCache.Store(cacheKey, &CachedAMIMapping{
		Mapping:  amiMapping,
		CachedAt: time.Now(),
		Region:   region,
		Arch:     architecture,
	})
	
	return amiMapping, nil
}

// hashTemplate creates a hash of the template for cache invalidation
func (r *OptimizedResolver) hashTemplate(template *Template) string {
	// Simple hash based on template content - in production, use proper hashing
	return fmt.Sprintf("%s_%s_%d", template.Name, template.Base, len(template.Packages.System))
}

// ClearCache clears all cached data
func (r *OptimizedResolver) ClearCache() {
	r.templateCache.Range(func(key, value interface{}) bool {
		r.templateCache.Delete(key)
		return true
	})
	r.amiCache.Range(func(key, value interface{}) bool {
		r.amiCache.Delete(key)
		return true
	})
}

// GetCacheStats returns cache statistics for monitoring
func (r *OptimizedResolver) GetCacheStats() map[string]int {
	templateCount := 0
	amiCount := 0
	
	r.templateCache.Range(func(key, value interface{}) bool {
		templateCount++
		return true
	})
	r.amiCache.Range(func(key, value interface{}) bool {
		amiCount++
		return true
	})
	
	return map[string]int{
		"template_cache_size": templateCount,
		"ami_cache_size":      amiCount,
	}
}