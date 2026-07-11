package aws

import (
	"context"
	"errors"
	"testing"
)

// fakeSpotFetcher records calls and returns a scripted rate/error.
type fakeSpotFetcher struct {
	rate  float64
	err   error
	calls int
}

func (f *fakeSpotFetcher) HourlyRate(_ context.Context, _, _, model string) (float64, error) {
	f.calls++
	if model != "spot" {
		return 0, errors.New("expected spot model")
	}
	return f.rate, f.err
}

func newTestSpotClient(f spotRateFetcher) *SpotPricingClient {
	return &SpotPricingClient{fetcher: f, cache: map[string]float64{}}
}

// TestSpotPricingClient_Caches: a successful lookup is cached, so a repeat call for the same
// (type,region) does not hit the fetcher again.
func TestSpotPricingClient_Caches(t *testing.T) {
	f := &fakeSpotFetcher{rate: 0.42}
	c := newTestSpotClient(f)
	ctx := context.Background()

	got, err := c.GetSpotHourlyRate(ctx, "m7i.large", "us-west-2")
	if err != nil || got != 0.42 {
		t.Fatalf("first lookup = %v, %v; want 0.42, nil", got, err)
	}
	got, err = c.GetSpotHourlyRate(ctx, "m7i.large", "us-west-2")
	if err != nil || got != 0.42 {
		t.Fatalf("second lookup = %v, %v; want 0.42, nil", got, err)
	}
	if f.calls != 1 {
		t.Fatalf("fetcher called %d times, want 1 (second served from cache)", f.calls)
	}
}

// TestSpotPricingClient_Error propagates the fetcher error (caller decides fallback) and does not cache.
func TestSpotPricingClient_Error(t *testing.T) {
	f := &fakeSpotFetcher{err: errors.New("no spot price")}
	c := newTestSpotClient(f)
	if _, err := c.GetSpotHourlyRate(context.Background(), "m7i.large", "us-west-2"); err == nil {
		t.Fatal("expected error to propagate")
	}
}

// TestSpotPricingClient_ZeroIsError: a non-positive rate is treated as unavailable.
func TestSpotPricingClient_ZeroIsError(t *testing.T) {
	f := &fakeSpotFetcher{rate: 0}
	c := newTestSpotClient(f)
	if _, err := c.GetSpotHourlyRate(context.Background(), "m7i.large", "us-west-2"); err == nil {
		t.Fatal("zero rate should be an error")
	}
}

// TestSpotPricingClient_NilSafe: a nil client / unset fetcher fails soft with an error, never panics.
func TestSpotPricingClient_NilSafe(t *testing.T) {
	var c *SpotPricingClient
	if _, err := c.GetSpotHourlyRate(context.Background(), "m7i.large", "us-west-2"); err == nil {
		t.Fatal("nil client should return an error, not panic")
	}
}
