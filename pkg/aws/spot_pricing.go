package aws

import (
	"context"
	"fmt"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	truffle "github.com/spore-host/truffle/pkg/aws"
)

// Spot pricing (#659). A spot instance is billed at the spot rate — fixed once the instance is
// running, and typically 60–90% below the on-demand list price prism otherwise estimates with. This
// client fetches the real spot rate so the spend ledger estimate tracks spot to the penny, largely
// removing the need for Cost Explorer reconciliation on spot workloads.
//
// Pricing is delegated to spore.host's truffle (github.com/spore-host/truffle), the discovery/pricing
// client — its HourlyRate(ctx, type, region, "spot") returns the minimum current spot price across
// AZs (via DescribeSpotPriceHistory) — rather than hand-rolling the EC2 call. This is prism's first
// dependency on spore.host, following the same plain-require pattern as budgetengine.

// spotRateFetcher is the narrow slice of truffle used here, extracted so tests can inject a fake
// without live AWS. *truffle.Client satisfies it.
type spotRateFetcher interface {
	HourlyRate(ctx context.Context, instanceType, region, model string) (float64, error)
}

// SpotPricingClient resolves and caches per-(instanceType,region) spot rates. The spot rate is stable
// for a running instance's life, so a captured value is cached indefinitely — the observer/refresh
// path also persists it on the instance, making this a second-line guard against repeated lookups.
type SpotPricingClient struct {
	fetcher spotRateFetcher
	mu      sync.RWMutex
	cache   map[string]float64
}

// NewSpotPricingClient builds a spot pricing client over truffle from the given AWS config.
func NewSpotPricingClient(cfg awssdk.Config) *SpotPricingClient {
	return &SpotPricingClient{
		fetcher: truffle.NewClientFromConfig(cfg),
		cache:   map[string]float64{},
	}
}

// GetSpotHourlyRate returns the current minimum spot $/hr for an instance type in a region. It errors
// (rather than falling back) so callers can decide the fallback — the observer/refresh path treats an
// error as "use the on-demand rate + warn", never blocking accrual.
func (c *SpotPricingClient) GetSpotHourlyRate(ctx context.Context, instanceType, region string) (float64, error) {
	if c == nil || c.fetcher == nil {
		return 0, fmt.Errorf("spot pricing client not configured")
	}
	key := region + "/" + instanceType

	c.mu.RLock()
	cached, ok := c.cache[key]
	c.mu.RUnlock()
	if ok {
		return cached, nil
	}

	rate, err := c.fetcher.HourlyRate(ctx, instanceType, region, "spot")
	if err != nil {
		return 0, err
	}
	if rate <= 0 {
		return 0, fmt.Errorf("no spot price available for %s in %s", instanceType, region)
	}

	c.mu.Lock()
	c.cache[key] = rate
	c.mu.Unlock()
	return rate, nil
}
