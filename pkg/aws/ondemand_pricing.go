package aws

import (
	"context"
	"fmt"
	"sync"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	truffle "github.com/spore-host/truffle/pkg/aws"
)

// On-demand pricing via truffle (spawn adoption, follows the spot path #659).
// The on-demand hourly rate is the baseline compute cost prism records on every
// instance (Instance.HourlyRate). Pricing is delegated to spore.host's truffle
// (github.com/spore-host/truffle) — its HourlyRate(ctx, type, region, "on-demand")
// wraps the AWS Price List API with a 24h cache AND a static fallback, and is
// region-aware (prism's former static getHourlyRate table was us-east-1-only),
// so this both removes hand-rolled Price List code and improves per-region
// accuracy. Truffle already handles spot here too (see SpotPricingClient); this
// is the on-demand twin.

// onDemandRateFetcher is the narrow slice of truffle used here, extracted so
// tests can inject a fake without live AWS. *truffle.Client satisfies it. It is
// the same signature spotRateFetcher uses — only the model string differs.
type onDemandRateFetcher interface {
	HourlyRate(ctx context.Context, instanceType, region, model string) (float64, error)
}

// OnDemandPricingClient resolves and caches per-(instanceType,region) on-demand
// rates. On-demand list prices change rarely, so a captured value is cached for
// the process lifetime (truffle also caches internally with a 24h TTL).
type OnDemandPricingClient struct {
	fetcher onDemandRateFetcher
	mu      sync.RWMutex
	cache   map[string]float64
}

// NewOnDemandPricingClient builds an on-demand pricing client over truffle from
// the given AWS config.
func NewOnDemandPricingClient(cfg awssdk.Config) *OnDemandPricingClient {
	return &OnDemandPricingClient{
		fetcher: truffle.NewClientFromConfig(cfg),
		cache:   map[string]float64{},
	}
}

// GetOnDemandHourlyRate returns the current on-demand $/hr for an instance type
// in a region. It errors (rather than falling back) so callers decide the
// fallback — BuildInstance treats an error as "use the static getHourlyRate
// estimate + log", never leaving the rate at 0.
func (c *OnDemandPricingClient) GetOnDemandHourlyRate(ctx context.Context, instanceType, region string) (float64, error) {
	if c == nil || c.fetcher == nil {
		return 0, fmt.Errorf("on-demand pricing client not configured")
	}
	key := region + "/" + instanceType

	c.mu.RLock()
	cached, ok := c.cache[key]
	c.mu.RUnlock()
	if ok {
		return cached, nil
	}

	rate, err := c.fetcher.HourlyRate(ctx, instanceType, region, "on-demand")
	if err != nil {
		return 0, err
	}
	if rate <= 0 {
		return 0, fmt.Errorf("no on-demand price available for %s in %s", instanceType, region)
	}

	c.mu.Lock()
	c.cache[key] = rate
	c.mu.Unlock()
	return rate, nil
}
