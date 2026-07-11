package aws

import (
	"context"
	"errors"
	"testing"
)

// fakeOnDemandFetcher records calls and returns a scripted rate/error.
type fakeOnDemandFetcher struct {
	rate  float64
	err   error
	calls int
}

func (f *fakeOnDemandFetcher) HourlyRate(_ context.Context, _, _, model string) (float64, error) {
	f.calls++
	switch model {
	case "", "on-demand", "ondemand", "on_demand":
		// accepted on-demand model strings
	default:
		return 0, errors.New("expected on-demand model, got " + model)
	}
	return f.rate, f.err
}

func newTestOnDemandClient(f onDemandRateFetcher) *OnDemandPricingClient {
	return &OnDemandPricingClient{fetcher: f, cache: map[string]float64{}}
}

// TestOnDemandPricingClient_Caches: a successful lookup is cached, so a repeat
// call for the same (type,region) does not hit the fetcher again.
func TestOnDemandPricingClient_Caches(t *testing.T) {
	f := &fakeOnDemandFetcher{rate: 0.0928}
	c := newTestOnDemandClient(f)
	ctx := context.Background()

	got, err := c.GetOnDemandHourlyRate(ctx, "t3.large", "us-west-2")
	if err != nil || got != 0.0928 {
		t.Fatalf("first lookup = %v, %v; want 0.0928, nil", got, err)
	}
	got, err = c.GetOnDemandHourlyRate(ctx, "t3.large", "us-west-2")
	if err != nil || got != 0.0928 {
		t.Fatalf("second lookup = %v, %v; want 0.0928, nil", got, err)
	}
	if f.calls != 1 {
		t.Fatalf("fetcher called %d times, want 1 (second served from cache)", f.calls)
	}
}

// TestOnDemandPricingClient_Error propagates the fetcher error (caller decides
// fallback) and does not cache.
func TestOnDemandPricingClient_Error(t *testing.T) {
	f := &fakeOnDemandFetcher{err: errors.New("no on-demand price")}
	c := newTestOnDemandClient(f)
	if _, err := c.GetOnDemandHourlyRate(context.Background(), "t3.large", "us-west-2"); err == nil {
		t.Fatal("expected error to propagate")
	}
}

// TestOnDemandPricingClient_ZeroIsError: a non-positive rate is unavailable.
func TestOnDemandPricingClient_ZeroIsError(t *testing.T) {
	f := &fakeOnDemandFetcher{rate: 0}
	c := newTestOnDemandClient(f)
	if _, err := c.GetOnDemandHourlyRate(context.Background(), "t3.large", "us-west-2"); err == nil {
		t.Fatal("zero rate should be an error")
	}
}

// TestOnDemandPricingClient_NilSafe: a nil client fails soft, never panics.
func TestOnDemandPricingClient_NilSafe(t *testing.T) {
	var c *OnDemandPricingClient
	if _, err := c.GetOnDemandHourlyRate(context.Background(), "t3.large", "us-west-2"); err == nil {
		t.Fatal("nil client should return an error, not panic")
	}
}
