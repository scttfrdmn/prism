package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
)

// PricingClient provides EBS volume pricing. On-demand and spot instance
// compute rates are sourced from spore.host's truffle (see OnDemandPricingClient
// and SpotPricingClient); the former AWS Price List API path lived here but was
// retired when compute pricing moved to truffle (region-aware, cached, with a
// static fallback). EBS volume rates remain here — truffle has no storage
// pricing — as a small static table.
type PricingClient struct{}

// NewPricingClient creates a pricing client. The AWS config is accepted for
// signature stability with the other pricing clients; EBS rates are static so
// no AWS client is built.
func NewPricingClient(_ aws.Config) *PricingClient {
	return &PricingClient{}
}

// GetEBSVolumeHourlyRate calculates the hourly cost for an EBS volume.
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
