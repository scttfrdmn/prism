package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// Tests for #720: a stop prism never recorded (#717) must not keep the budget ledger, or the cost
// analytics, billing compute up to now.

var unrecordedStopLaunch = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// hibernatedWithUnrecordedStop is a workspace whose history ends at "pending → running" 10h after
// launch, but which AWS now reports hibernated: the stop never made it into the history.
func hibernatedWithUnrecordedStop(stop *time.Time) types.Instance {
	return types.Instance{
		ID:         "i-1",
		ProjectID:  "proj-1",
		State:      "hibernated",
		HourlyRate: 1.0,
		LaunchTime: unrecordedStopLaunch,
		StateHistory: []types.StateTransition{
			{FromState: "", ToState: "running", Timestamp: unrecordedStopLaunch},
			{FromState: "running", ToState: "stopping", Timestamp: unrecordedStopLaunch.Add(2 * time.Hour)},
			{FromState: "stopping", ToState: "hibernated", Timestamp: unrecordedStopLaunch.Add(2 * time.Hour)},
			{FromState: "hibernated", ToState: "pending", Timestamp: unrecordedStopLaunch.Add(10 * time.Hour)},
			{FromState: "pending", ToState: "running", Timestamp: unrecordedStopLaunch.Add(10 * time.Hour)},
		},
		ObservedStopTime: stop,
	}
}

func TestRunningHours_UnrecordedStop(t *testing.T) {
	now := unrecordedStopLaunch.Add(100 * time.Hour)
	stop := unrecordedStopLaunch.Add(15 * time.Hour)
	early := unrecordedStopLaunch.Add(5 * time.Hour) // before the last recorded start
	late := now.Add(time.Hour)                       // after now (clock skew)

	tests := []struct {
		name string
		inst types.Instance
		want float64
	}{
		{"AWS stop time bounds the trailing segment", hibernatedWithUnrecordedStop(&stop), 2 + 5},
		{"no AWS stop time: nothing billed past the last start", hibernatedWithUnrecordedStop(nil), 2},
		{"stop time before the last start is ignored", hibernatedWithUnrecordedStop(&early), 2},
		{"stop time after now (clock skew) clips at now", hibernatedWithUnrecordedStop(&late), 2 + 90},
		{"still running: bills to now (unchanged)", func() types.Instance {
			inst := hibernatedWithUnrecordedStop(nil)
			inst.State = "running"
			return inst
		}(), 2 + 90},
		{"stopping is not cut short (billable while hibernating)", func() types.Instance {
			inst := hibernatedWithUnrecordedStop(&stop)
			inst.State = "stopping"
			return inst
		}(), 2 + 90},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runningHours(tt.inst, now); got < tt.want-0.01 || got > tt.want+0.01 {
				t.Fatalf("runningHours = %.2f, want %.2f", got, tt.want)
			}
		})
	}
}

func TestRunningHoursInWindow_UnrecordedStop(t *testing.T) {
	stop := unrecordedStopLaunch.Add(15 * time.Hour)
	inst := hibernatedWithUnrecordedStop(&stop)
	// Window [launch+8h, launch+100h): only the 10h→15h run falls inside it.
	got := runningHoursInWindow(inst, unrecordedStopLaunch.Add(8*time.Hour), unrecordedStopLaunch.Add(100*time.Hour))
	if got < 4.99 || got > 5.01 {
		t.Fatalf("runningHoursInWindow = %.2f, want 5", got)
	}
}

// TestObserver_UnrecordedStopStopsAccruingCompute: once the instance is observed hibernated, later
// accrual passes add storage only -- no compute for the hours it has been off.
func TestObserver_UnrecordedStopStopsAccruingCompute(t *testing.T) {
	s := newTestSpendStore(t)
	stop := unrecordedStopLaunch.Add(15 * time.Hour)
	inst := hibernatedWithUnrecordedStop(&stop)
	inst.StorageGB = 30 // keeps the hibernated instance in the accrual loop (EBS still bills)
	insts := []types.Instance{inst}
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true, spendObserverOptions{})

	obs.clock = func() time.Time { return unrecordedStopLaunch.Add(20 * time.Hour) }
	obs.Observe(context.Background())
	obs.clock = func() time.Time { return unrecordedStopLaunch.Add(200 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	var compute float64
	for _, e := range evs {
		compute += e.Compute
	}
	if want := 7.0; compute < want-0.01 || compute > want+0.01 { // 2h + 5h at $1/hr
		t.Fatalf("ledger compute = $%.2f, want $%.2f (pre-fix: $%.2f, billed to t+200h)", compute, want, 2.0+190.0)
	}
}

func TestRecordObservedTransition(t *testing.T) {
	lastStart := time.Now().Add(-48 * time.Hour)
	cachedRunning := types.Instance{
		Name:  "ws",
		State: "running",
		StateHistory: []types.StateTransition{
			{FromState: "pending", ToState: "running", Timestamp: lastStart},
		},
	}

	t.Run("stop is recorded at AWS's stop time", func(t *testing.T) {
		stop := lastStart.Add(3 * time.Hour)
		live := &types.Instance{Name: "ws", State: "stopped", ObservedStopTime: &stop}
		recordObservedTransition(cachedRunning, live, "aws_observed", "system")
		if len(live.StateHistory) != 2 {
			t.Fatalf("history len = %d, want 2", len(live.StateHistory))
		}
		got := live.StateHistory[1]
		if got.FromState != "running" || got.ToState != "stopped" || !got.Timestamp.Equal(stop) || got.Reason != "aws_observed" {
			t.Fatalf("transition = %+v, want running→stopped at %s", got, stop)
		}
	})

	t.Run("stop time not after the last transition falls back to now", func(t *testing.T) {
		stop := lastStart.Add(-time.Hour)
		live := &types.Instance{Name: "ws", State: "stopped", ObservedStopTime: &stop}
		before := time.Now()
		recordObservedTransition(cachedRunning, live, "aws_observed", "system")
		if ts := live.StateHistory[len(live.StateHistory)-1].Timestamp; ts.Before(before) {
			t.Fatalf("timestamp = %s, want ~now", ts)
		}
	})

	t.Run("unchanged state carries history, appends nothing", func(t *testing.T) {
		live := &types.Instance{Name: "ws", State: "running"}
		recordObservedTransition(cachedRunning, live, "aws_observed", "system")
		if len(live.StateHistory) != 1 {
			t.Fatalf("history len = %d, want 1 (carried from cache)", len(live.StateHistory))
		}
	})

	t.Run("unpropagated AWS state keeps the optimistic cached state", func(t *testing.T) {
		cached := cachedRunning
		cached.State = "stopping"                             // written optimistically by applyExpectedTransitionalState
		live := &types.Instance{Name: "ws", State: "running"} // AWS has not caught up yet
		recordObservedTransition(cached, live, "aws_observed", "system")
		if live.State != "stopping" || len(live.StateHistory) != 1 {
			t.Fatalf("state=%s history=%d, want stopping with no new transition", live.State, len(live.StateHistory))
		}
	})
}

func TestInstancesToCheck_SweepIncludesRunning(t *testing.T) {
	instances := map[string]types.Instance{
		"a": {ID: "i-a", State: "running"},
		"b": {ID: "i-b", State: "stopping"},
		"c": {ID: "i-c", State: "stopped"},
		"d": {ID: "", State: "running"}, // never launched (no AWS ID): nothing to ask AWS about
	}
	if got := instancesToCheck(instances, false); len(got) != 1 || got[0].ID != "i-b" {
		t.Fatalf("non-sweep tick = %+v, want only the transitional instance", got)
	}
	got := instancesToCheck(instances, true)
	ids := map[string]bool{}
	for _, inst := range got {
		ids[inst.ID] = true
	}
	// Stopped is swept too: a start prism didn't initiate would otherwise under-bill the ledger.
	if len(got) != 3 || !ids["i-a"] || !ids["i-b"] || !ids["i-c"] {
		t.Fatalf("sweep tick = %+v, want running, transitional and stopped (not the ID-less one)", got)
	}
	if !isTransitionalState("hibernating") {
		t.Fatal("hibernating must be polled, or an instance caught mid-hibernation is never re-checked")
	}
}

// A stop time past the end of the window means the run covered the whole rest of the window: clip,
// don't drop.
func TestTrailingRunEnd_StopAfterWindowClips(t *testing.T) {
	stop := unrecordedStopLaunch.Add(300 * time.Hour)
	inst := hibernatedWithUnrecordedStop(&stop)
	got := runningHoursInWindow(inst, unrecordedStopLaunch, unrecordedStopLaunch.Add(100*time.Hour))
	if want := 2.0 + 90.0; got < want-0.01 || got > want+0.01 {
		t.Fatalf("runningHoursInWindow = %.2f, want %.2f (clipped at the window end)", got, want)
	}
}

func TestObservedTransitionTime_StartUsesAWSRunningStart(t *testing.T) {
	stoppedAt := time.Now().Add(-10 * time.Hour)
	cached := types.Instance{State: "stopped", StateHistory: []types.StateTransition{
		{FromState: "running", ToState: "stopped", Timestamp: stoppedAt},
	}}

	started := stoppedAt.Add(4 * time.Hour)
	live := &types.Instance{State: "running", RunningStateStartTime: &started}
	if got := observedTransitionTime(cached, live); !got.Equal(started) {
		t.Fatalf("start timestamp = %s, want AWS running start %s", got, started)
	}

	stale := stoppedAt.Add(-time.Hour) // e.g. a cached value from the previous run
	live = &types.Instance{State: "running", RunningStateStartTime: &stale}
	if got := observedTransitionTime(cached, live); got.Before(stoppedAt) {
		t.Fatalf("stale running start %s was used; want ~now", got)
	}
}
