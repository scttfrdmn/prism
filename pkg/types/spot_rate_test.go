package types

import "testing"

// TestEffectiveComputeRate covers the spot-vs-on-demand rate selection (#659).
func TestEffectiveComputeRate(t *testing.T) {
	cases := []struct {
		name string
		inst Instance
		want float64
	}{
		{"on-demand uses HourlyRate", Instance{InstanceLifecycle: "on-demand", HourlyRate: 2.0}, 2.0},
		{"spot with captured rate uses spot", Instance{InstanceLifecycle: "spot", HourlyRate: 2.0, SpotHourlyRate: 0.6}, 0.6},
		{"spot without captured rate falls back to on-demand", Instance{InstanceLifecycle: "spot", HourlyRate: 2.0}, 2.0},
		{"IsSpot flag also counts as spot", Instance{IsSpot: true, HourlyRate: 2.0, SpotHourlyRate: 0.5}, 0.5},
		{"on-demand ignores a stray SpotHourlyRate", Instance{InstanceLifecycle: "on-demand", HourlyRate: 2.0, SpotHourlyRate: 0.6}, 2.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.inst.EffectiveComputeRate(); got != c.want {
				t.Fatalf("EffectiveComputeRate() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsSpotInstance(t *testing.T) {
	if !(Instance{InstanceLifecycle: "spot"}).IsSpotInstance() {
		t.Fatal("InstanceLifecycle=spot should be spot")
	}
	if !(Instance{IsSpot: true}).IsSpotInstance() {
		t.Fatal("IsSpot=true should be spot")
	}
	if (Instance{InstanceLifecycle: "on-demand"}).IsSpotInstance() {
		t.Fatal("on-demand should not be spot")
	}
}
