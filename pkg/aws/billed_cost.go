package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// costAllocationTagKey is the AWS cost-allocation tag prism stamps on every
// managed instance (value == workspace name). For per-instance billed cost it
// must be ACTIVATED in the payer account's Billing console, and activation is
// not retroactive: cost before activation is not attributed to the tag.
const costAllocationTagKey = "prism:instance-id"

// GetBilledCost returns the AWS-billed cost for one workspace over [since, now],
// read from AWS Cost Explorer using the UnblendedCost metric. This is the
// metered "billed so far" amount AWS has charged, not prism's local estimate
// (Instance.CurrentSpend).
//
// Cost Explorer is a global service reached through a single endpoint, so the
// client is pinned to us-east-1 regardless of the workspace's region; the
// region is applied as a filter dimension instead.
//
// Isolation strategy: query first by the prism cost-allocation tag (exact, when
// the tag is activated). If that returns nothing -- the common case when the
// tag has not been activated in Billing -- fall back to the region's EC2 total
// and label the result so the caller can warn that it may include other
// instances.
func (m *Manager) GetBilledCost(ctx context.Context, name, region string, since time.Time) (*ctypes.BilledCostResult, error) {
	ce := costexplorer.NewFromConfig(m.cfg, func(o *costexplorer.Options) {
		o.Region = "us-east-1"
	})

	// Cost Explorer's end date is exclusive; end tomorrow to include today.
	start := since.UTC().Format("2006-01-02")
	if since.IsZero() {
		start = time.Now().UTC().AddDate(-1, 0, 0).Format("2006-01-02")
	}
	end := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")

	result := &ctypes.BilledCostResult{
		Name:     name,
		Currency: "USD",
		Start:    start,
		End:      end,
		Region:   region,
		Source:   "AWS Cost Explorer (UnblendedCost)",
	}

	// Primary: isolate this instance by its prism cost-allocation tag.
	tagged, err := m.queryBilled(ctx, ce, start, end, &cetypes.Expression{
		Tags: &cetypes.TagValues{
			Key:    aws.String(costAllocationTagKey),
			Values: []string{name},
		},
	})
	if err != nil {
		return nil, wrapBilledCostErr(err)
	}
	if tagged.total > 0 {
		result.TagActive = true
		result.BilledTotal = tagged.total
		result.Monthly = tagged.periods
		result.Estimated = tagged.estimated
		return result, nil
	}

	// Fallback: tag not activated (or no spend yet). Report the region's EC2
	// total, clearly labeled as possibly including other instances.
	regionRes, err := m.queryBilled(ctx, ce, start, end, &cetypes.Expression{
		And: []cetypes.Expression{
			{Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionRegion,
				Values: []string{region},
			}},
			{Dimensions: &cetypes.DimensionValues{
				Key: cetypes.DimensionService,
				Values: []string{
					"Amazon Elastic Compute Cloud - Compute",
					"EC2 - Other",
				},
			}},
		},
	})
	if err != nil {
		return nil, wrapBilledCostErr(err)
	}
	result.TagActive = false
	result.BilledTotal = regionRes.total
	result.FallbackRegionTotal = regionRes.total
	result.Monthly = regionRes.periods
	result.Estimated = regionRes.estimated
	result.Note = fmt.Sprintf(
		"cost-allocation tag %q is not active, so this is the EC2 total for region %s and may include other instances. Activate the tag in the AWS Billing console (Cost Allocation Tags) for exact per-instance cost; activation is not retroactive.",
		costAllocationTagKey, region)
	return result, nil
}

type billedQueryResult struct {
	total     float64
	estimated bool
	periods   []ctypes.BilledCostPeriod
}

func (m *Manager) queryBilled(ctx context.Context, ce *costexplorer.Client, start, end string, filter *cetypes.Expression) (*billedQueryResult, error) {
	out, err := ce.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Granularity: cetypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		Filter:      filter,
	})
	if err != nil {
		return nil, err
	}
	return summarizeBilled(out.ResultsByTime), nil
}

// summarizeBilled reduces Cost Explorer's per-period results into a single
// billedQueryResult: the UnblendedCost summed across periods, each period
// preserved, and estimated set if any period is still AWS-estimated. It is a
// pure function of the API response so it can be unit-tested without AWS.
func summarizeBilled(results []cetypes.ResultByTime) *billedQueryResult {
	res := &billedQueryResult{}
	for _, period := range results {
		amount := 0.0
		if metric, ok := period.Total["UnblendedCost"]; ok && metric.Amount != nil {
			// Amount arrives as a decimal string, e.g. "70.1414001792".
			fmt.Sscanf(*metric.Amount, "%f", &amount)
		}
		res.total += amount
		if period.Estimated {
			res.estimated = true
		}
		p := ctypes.BilledCostPeriod{Amount: amount, Estimated: period.Estimated}
		if period.TimePeriod != nil {
			if period.TimePeriod.Start != nil {
				p.Start = *period.TimePeriod.Start
			}
			if period.TimePeriod.End != nil {
				p.End = *period.TimePeriod.End
			}
		}
		res.periods = append(res.periods, p)
	}
	return res
}

// wrapBilledCostErr adds an actionable hint when Cost Explorer denies access,
// the most common failure (the daemon's AWS identity lacks ce:GetCostAndUsage).
func wrapBilledCostErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "AccessDenied") || strings.Contains(err.Error(), "not authorized") {
		return fmt.Errorf("AWS Cost Explorer access denied -- the daemon's AWS identity needs the ce:GetCostAndUsage permission (and Cost Explorer must be enabled once in the Billing console): %w", err)
	}
	return fmt.Errorf("cost explorer query failed: %w", err)
}
