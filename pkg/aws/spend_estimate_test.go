package aws

import (
	"testing"
	"time"

	ctypes "github.com/scttfrdmn/prism/pkg/types"
)

// These tests cover the spend estimate prism shows as CurrentSpend. The case
// that motivated them: bristol-workspace-2 hibernated on 11 September 2026, no
// transition out of "running" was ever written to the state history, and the
// estimate went on charging the full m7i.xlarge rate for every hour that
// followed. It reached $129.86 against about $7.40 that AWS actually billed.
// The rule the tests encode is that AWS's observed state wins over prism's
// record of it, because the record can miss a transition and the observation
// cannot.

const (
	testComputeRate = 0.2016  // m7i.xlarge on-demand, us-east-2
	testStorageRate = 0.00877 // 80 GiB gp3 at $0.08/GB-month over 730 hours
)

// hoursAgo is the wall-clock instant n hours before now, which is how these
// tests express "the instance started two days ago" without freezing time.
func hoursAgo(n float64) time.Time {
	return time.Now().Add(-time.Duration(n * float64(time.Hour)))
}

func transition(from, to string, at time.Time) ctypes.StateTransition {
	return ctypes.StateTransition{FromState: from, ToState: to, Timestamp: at}
}

// closeTo keeps the assertions readable: every expected value here is derived
// from time.Now(), so exact equality would be flaky.
func closeTo(got, want, tol float64) bool {
	diff := got - want
	return diff < tol && diff > -tol
}

func TestCalculateRunningHoursFromHistory_StoppedAfterLastRecordedStart(t *testing.T) {
	// The bristol-workspace-2 case. Billing starts when the instance entered
	// running 100 hours ago, which is the last transition in the history. AWS
	// says the instance is stopped now and dates the stop to 40 hours ago, so
	// the billable window is the 60 hours between the two.
	start := hoursAgo(100)
	stoppedAt := hoursAgo(40)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(100)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateStopped, history, false, &stoppedAt)

	if !closeTo(got, 60, 0.01) {
		t.Errorf("running hours = %.4f, want 60 (start 100h ago to stop 40h ago)", got)
	}
}

func TestCalculateRunningHoursFromHistory_StoppedWithNoAWSStopTime(t *testing.T) {
	// Same shape, but AWS gave no parseable stop time. The instance is stopped,
	// so the hours after the last recorded start are not billable, and without a
	// stop time there is nothing to bill them up to. Billing none of them
	// under-counts by at most the unrecorded window. Billing all of them, which
	// is what the code did before, over-counts without bound.
	start := hoursAgo(100)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(100)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateStopped, history, false, nil)

	if !closeTo(got, 0, 0.01) {
		t.Errorf("running hours = %.4f, want 0 (no hours we can vouch for)", got)
	}
}

func TestCalculateRunningHoursFromHistory_HibernatedIsNotBillable(t *testing.T) {
	// Prism reports hibernated instances with its own state string rather than
	// AWS's "stopped", so the check has to recognize both.
	start := hoursAgo(50)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(48)),
	}

	got := calculateRunningHoursFromHistory(start, "hibernated", history, false, nil)

	if !closeTo(got, 2, 0.01) {
		t.Errorf("running hours = %.4f, want 2 (nothing billable after the recorded start)", got)
	}
}

func TestCalculateRunningHoursFromHistory_StillRunningBillsToNow(t *testing.T) {
	// The unchanged path: AWS agrees the instance is running, so the hours keep
	// accruing to now.
	start := hoursAgo(10)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(10)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateRunning, history, false, nil)

	if !closeTo(got, 10, 0.01) {
		t.Errorf("running hours = %.4f, want 10", got)
	}
}

func TestCalculateRunningHoursFromHistory_RecordedStopEndsBilling(t *testing.T) {
	// When the history does record the stop, the stop wins and the AWS
	// timestamp is not needed.
	start := hoursAgo(30)
	history := []ctypes.StateTransition{
		transition("running", "stopping", hoursAgo(25)),
		transition("stopping", "stopped", hoursAgo(25)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateStopped, history, false, nil)

	if !closeTo(got, 5, 0.01) {
		t.Errorf("running hours = %.4f, want 5 (billing start to the recorded stop)", got)
	}
}

func TestCalculateRunningHoursFromHistory_EmptyHistory(t *testing.T) {
	// A stopped instance with no history bills nothing, and a running one bills
	// the whole span.
	start := hoursAgo(12)

	if got := calculateRunningHoursFromHistory(start, instanceStateStopped, nil, false, nil); got != 0 {
		t.Errorf("stopped with no history = %.4f, want 0", got)
	}
	if got := calculateRunningHoursFromHistory(start, instanceStateRunning, nil, false, nil); !closeTo(got, 12, 0.01) {
		t.Errorf("running with no history = %.4f, want 12", got)
	}
}

func TestCalculateActualCosts_HibernatedChargesStorageNotCompute(t *testing.T) {
	// The whole point of hibernating: the disk goes on billing, the instance
	// does not. Here the workspace ran for 10 of the last 100 hours.
	start := hoursAgo(100)
	stoppedAt := hoursAgo(90)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(100)),
	}

	spend, rate := calculateActualCosts(testComputeRate, testStorageRate, start, instanceStateStopped, history, false, &stoppedAt)

	wantCompute := 10 * testComputeRate
	wantStorage := 100 * testStorageRate
	if !closeTo(spend, wantCompute+wantStorage, 0.01) {
		t.Errorf("spend = %.4f, want %.4f (10h compute + 100h storage)", spend, wantCompute+wantStorage)
	}
	// The effective rate is spend per elapsed hour, well under the compute rate
	// because the machine spent 90 of those hours stopped.
	if rate >= testComputeRate {
		t.Errorf("effective rate = %.5f, want well below the %.4f compute rate", rate, testComputeRate)
	}
}

func TestCalculateActualCosts_RunningChargesBothRates(t *testing.T) {
	// A machine that has run the whole time is charged both rates for every
	// hour, so the effective rate is their sum.
	start := hoursAgo(20)
	history := []ctypes.StateTransition{
		transition("pending", "running", hoursAgo(20)),
	}

	spend, rate := calculateActualCosts(testComputeRate, testStorageRate, start, instanceStateRunning, history, false, nil)

	if !closeTo(spend, 20*(testComputeRate+testStorageRate), 0.01) {
		t.Errorf("spend = %.4f, want %.4f", spend, 20*(testComputeRate+testStorageRate))
	}
	if !closeTo(rate, testComputeRate+testStorageRate, 0.001) {
		t.Errorf("effective rate = %.5f, want %.5f", rate, testComputeRate+testStorageRate)
	}
}

func TestCalculateRunningHoursFromHistory_FirstTransitionLeavesStopped(t *testing.T) {
	// The walk used to open by assuming the instance was running, so the hours
	// between the billing start and the first recorded transition were billed
	// as running hours even when that transition says the instance was leaving
	// "stopped". Here the workspace was stopped for 20 hours, started, and has
	// run for 10.
	start := hoursAgo(30)
	history := []ctypes.StateTransition{
		transition(instanceStateStopped, "pending", hoursAgo(10)),
		transition("pending", instanceStateRunning, hoursAgo(10)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateRunning, history, false, nil)

	if !closeTo(got, 10, 0.01) {
		t.Errorf("running hours = %.4f, want 10 (the 20 hours before the start were stopped)", got)
	}
}

func TestCalculateRunningHoursFromHistory_FirstTransitionLeavesRunning(t *testing.T) {
	// The other side of the same rule. The first transition says the instance
	// was leaving "running", so the hours before it were running hours and are
	// billed, as they always were.
	start := hoursAgo(30)
	history := []ctypes.StateTransition{
		transition(instanceStateRunning, instanceStateStopping, hoursAgo(10)),
		transition(instanceStateStopping, instanceStateStopped, hoursAgo(10)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateStopped, history, false, nil)

	if !closeTo(got, 20, 0.01) {
		t.Errorf("running hours = %.4f, want 20 (billing start to the recorded stop)", got)
	}
}

func TestCalculateRunningHoursFromHistory_LaunchEntryKeepsRunningAssumption(t *testing.T) {
	// A launch is recorded with an empty FromState, which names no earlier
	// state to read. That case keeps the old assumption, so behaviour for a
	// freshly launched instance does not change.
	start := hoursAgo(5)
	history := []ctypes.StateTransition{
		transition("", "pending", hoursAgo(4)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateRunning, history, false, nil)

	if !closeTo(got, 5, 0.01) {
		t.Errorf("running hours = %.4f, want 5 (pending is billable, as is the span before it)", got)
	}
}

func TestCalculateRunningHoursFromHistory_StoppedSpanAndUnrecordedStop(t *testing.T) {
	// Both corrections at once, which is the shape the bristol-workspace-2
	// history actually has: a stopped span before the recorded restart, and a
	// stop after it that nothing recorded. Billing starts 120 hours ago, the
	// workspace was stopped until it restarted 100 hours ago, and AWS dates its
	// stop to 40 hours ago. Only those 60 hours are billable.
	start := hoursAgo(120)
	stoppedAt := hoursAgo(40)
	history := []ctypes.StateTransition{
		transition(instanceStateStopped, "pending", hoursAgo(100)),
		transition("pending", instanceStateRunning, hoursAgo(100)),
	}

	got := calculateRunningHoursFromHistory(start, instanceStateStopped, history, false, &stoppedAt)

	if !closeTo(got, 60, 0.01) {
		t.Errorf("running hours = %.4f, want 60 (restart 100h ago to stop 40h ago)", got)
	}
}
