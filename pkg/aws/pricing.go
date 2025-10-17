package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	pricingtypes "github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// PricingCache caches AWS pricing data to avoid repeated API calls
type PricingCache struct {
	mu          sync.RWMutex
	prices      map[string]float64 // key: "region:instanceType"
	lastUpdated time.Time
	ttl         time.Duration
}

// NewPricingCache creates a new pricing cache with 24-hour TTL
func NewPricingCache() *PricingCache {
	return &PricingCache{
		prices: make(map[string]float64),
		ttl:    24 * time.Hour,
	}
}

// Get retrieves a cached price if available and not expired
func (c *PricingCache) Get(region, instanceType string) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Since(c.lastUpdated) > c.ttl {
		return 0, false
	}

	key := fmt.Sprintf("%s:%s", region, instanceType)
	price, exists := c.prices[key]
	return price, exists
}

// Set stores a price in the cache
func (c *PricingCache) Set(region, instanceType string, price float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", region, instanceType)
	c.prices[key] = price
	c.lastUpdated = time.Now()
}

// PricingClient wraps AWS Pricing API with caching
type PricingClient struct {
	client *pricing.Client
	cache  *PricingCache
}

// NewPricingClient creates a new pricing client with the given AWS config
func NewPricingClient(cfg aws.Config) *PricingClient {
	// AWS Pricing API is only available in us-east-1
	pricingCfg := cfg.Copy()
	pricingCfg.Region = "us-east-1"

	return &PricingClient{
		client: pricing.NewFromConfig(pricingCfg),
		cache:  NewPricingCache(),
	}
}

// GetInstanceHourlyRate fetches the on-demand hourly rate for an instance type in a region
func (p *PricingClient) GetInstanceHourlyRate(ctx context.Context, region, instanceType string) (float64, error) {
	// Check cache first
	if price, ok := p.cache.Get(region, instanceType); ok {
		return price, nil
	}

	// Query AWS Pricing API
	price, err := p.queryPricingAPI(ctx, region, instanceType)
	if err != nil {
		// Fall back to hardcoded estimates on error
		return getHourlyRate(instanceType), fmt.Errorf("pricing API failed, using estimate: %w", err)
	}

	// Cache the result
	p.cache.Set(region, instanceType, price)
	return price, nil
}

// queryPricingAPI queries the AWS Pricing API for instance pricing
func (p *PricingClient) queryPricingAPI(ctx context.Context, region, instanceType string) (float64, error) {
	// Map region codes to AWS Pricing API location names
	regionLocations := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "EU (Ireland)",
		"eu-west-2":      "EU (London)",
		"eu-west-3":      "EU (Paris)",
		"eu-central-1":   "EU (Frankfurt)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
	}

	location, ok := regionLocations[region]
	if !ok {
		return 0, fmt.Errorf("unknown region: %s", region)
	}

	// Build filters for the pricing query
	filters := []pricingtypes.Filter{
		{
			Field: aws.String("ServiceCode"),
			Value: aws.String("AmazonEC2"),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("instanceType"),
			Value: aws.String(instanceType),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("location"),
			Value: aws.String(location),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("tenancy"),
			Value: aws.String("Shared"),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("operatingSystem"),
			Value: aws.String("Linux"),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("preInstalledSw"),
			Value: aws.String("NA"),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
		{
			Field: aws.String("capacitystatus"),
			Value: aws.String("Used"),
			Type:  pricingtypes.FilterTypeTermMatch,
		},
	}

	// Query the pricing API
	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int32(1),
	}

	result, err := p.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("pricing API query failed: %w", err)
	}

	if len(result.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing data found for %s in %s", instanceType, region)
	}

	// Parse the pricing data (AWS returns JSON strings)
	price, err := p.parsePriceFromJSON(result.PriceList[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse pricing data: %w", err)
	}

	return price, nil
}

// parsePriceFromJSON extracts the on-demand hourly price from AWS Pricing API JSON response
func (p *PricingClient) parsePriceFromJSON(priceListJSON string) (float64, error) {
	var priceData map[string]interface{}
	if err := json.Unmarshal([]byte(priceListJSON), &priceData); err != nil {
		return 0, fmt.Errorf("failed to unmarshal price data: %w", err)
	}

	// Navigate the nested JSON structure to find on-demand pricing
	// Structure: terms.OnDemand.{offerTermCode}.priceDimensions.{dimensionCode}.pricePerUnit.USD
	terms, ok := priceData["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("terms field not found")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("OnDemand terms not found")
	}

	// Get the first (and usually only) on-demand offer
	for _, offerData := range onDemand {
		offer, ok := offerData.(map[string]interface{})
		if !ok {
			continue
		}

		priceDimensions, ok := offer["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		// Get the first price dimension
		for _, dimensionData := range priceDimensions {
			dimension, ok := dimensionData.(map[string]interface{})
			if !ok {
				continue
			}

			pricePerUnit, ok := dimension["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			usdPrice, ok := pricePerUnit["USD"].(string)
			if !ok {
				continue
			}

			// Parse the price string to float64
			var price float64
			if _, err := fmt.Sscanf(usdPrice, "%f", &price); err != nil {
				return 0, fmt.Errorf("failed to parse price: %w", err)
			}

			return price, nil
		}
	}

	return 0, fmt.Errorf("price not found in response")
}

// GetEBSVolumeHourlyRate calculates the hourly cost for an EBS volume
func (p *PricingClient) GetEBSVolumeHourlyRate(volumeType string, sizeGB int) float64 {
	// EBS pricing per GB-month, converted to per GB-hour
	// These are us-east-1 rates as of 2024
	pricePerGBMonth := map[string]float64{
		"gp3":      0.08,
		"gp2":      0.10,
		"io1":      0.125,
		"io2":      0.125,
		"st1":      0.045,
		"sc1":      0.015,
		"standard": 0.05,
	}

	ratePerGBMonth, ok := pricePerGBMonth[volumeType]
	if !ok {
		ratePerGBMonth = 0.10 // Default to gp2 rate
	}

	// Convert monthly cost to hourly: monthly / (30 days * 24 hours)
	ratePerGBHour := ratePerGBMonth / (30 * 24)
	return float64(sizeGB) * ratePerGBHour
}

// regionFromAZ extracts the region from an availability zone
func regionFromAZ(az string) string {
	// Availability zones are like "us-east-1a", we want "us-east-1"
	parts := strings.Split(az, "-")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "-")
	}
	return az // Return as-is if parsing fails
}
