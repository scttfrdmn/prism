package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func ceResult(start, end, amount string, estimated bool) cetypes.ResultByTime {
	return cetypes.ResultByTime{
		Estimated: estimated,
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start),
			End:   aws.String(end),
		},
		Total: map[string]cetypes.MetricValue{
			"UnblendedCost": {Amount: aws.String(amount), Unit: aws.String("USD")},
		},
	}
}

func TestSummarizeBilled_SumsAcrossPeriods(t *testing.T) {
	// Mirrors the real bristol-workspace lifetime: Mar..Jun, last period estimated.
	results := []cetypes.ResultByTime{
		ceResult("2026-03-01", "2026-04-01", "11.5467688704", false),
		ceResult("2026-04-01", "2026-05-01", "45.8914960224", false),
		ceResult("2026-05-01", "2026-06-01", "70.1414001792", false),
		ceResult("2026-06-01", "2026-06-04", "8.8707360672", true),
	}

	got := summarizeBilled(results)

	const want = 136.4504011392
	if diff := got.total - want; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("total = %.10f, want %.10f", got.total, want)
	}
	if !got.estimated {
		t.Error("estimated = false, want true (last period is AWS-estimated)")
	}
	if len(got.periods) != 4 {
		t.Fatalf("periods = %d, want 4", len(got.periods))
	}
	if got.periods[3].Start != "2026-06-01" || !got.periods[3].Estimated {
		t.Errorf("last period = %+v, want start 2026-06-01 and estimated", got.periods[3])
	}
}

func TestSummarizeBilled_EmptyAndMissingMetric(t *testing.T) {
	// No results -> zero, not estimated.
	if got := summarizeBilled(nil); got.total != 0 || got.estimated || len(got.periods) != 0 {
		t.Errorf("empty summarize = %+v, want zero/false/0-periods", got)
	}

	// A period with no UnblendedCost metric contributes zero, not a panic.
	results := []cetypes.ResultByTime{
		{TimePeriod: &cetypes.DateInterval{Start: aws.String("2026-05-01"), End: aws.String("2026-06-01")}},
	}
	got := summarizeBilled(results)
	if got.total != 0 {
		t.Errorf("total = %v, want 0 when metric absent", got.total)
	}
	if len(got.periods) != 1 || got.periods[0].Amount != 0 {
		t.Errorf("periods = %+v, want one zero-amount period", got.periods)
	}
}

func TestWrapBilledCostErr_AccessDeniedHint(t *testing.T) {
	if wrapBilledCostErr(nil) != nil {
		t.Error("wrapBilledCostErr(nil) should be nil")
	}

	denied := wrapBilledCostErr(errString("AccessDeniedException: user is not authorized to perform ce:GetCostAndUsage"))
	if denied == nil || !containsAll(denied.Error(), "ce:GetCostAndUsage", "Billing console") {
		t.Errorf("access-denied error missing hint: %v", denied)
	}

	other := wrapBilledCostErr(errString("boom"))
	if other == nil || !containsAll(other.Error(), "cost explorer query failed") {
		t.Errorf("generic error not wrapped: %v", other)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
