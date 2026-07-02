package engine

import "testing"

func TestSilentRefreshKeepsLoadedStatus(t *testing.T) {
	d := NewBaseDisplay(1, "test", "Test", true)

	// initial fetch: loading then loaded
	if !d.BeginFetch(&WeatherParams{}) {
		t.Fatal("BeginFetch should return true for enabled display")
	}
	if d.Status() != StatusLoading {
		t.Fatalf("expected Loading, got %v", d.Status())
	}
	d.SetStatus(StatusLoaded)

	// refresh: status must stay Loaded while re-fetching
	d.BeginFetch(&WeatherParams{})
	if d.Status() != StatusLoaded {
		t.Fatalf("silent refresh should keep Loaded, got %v", d.Status())
	}

	// a failed refresh must not drop the display out of rotation
	d.SetStatus(StatusFailed)
	if d.Status() != StatusLoaded {
		t.Fatalf("failed refresh should keep Loaded, got %v", d.Status())
	}

	// but NoData is legitimate (e.g. hazards cleared) and passes through
	d.SetStatus(StatusNoData)
	if d.Status() != StatusNoData {
		t.Fatalf("expected NoData, got %v", d.Status())
	}
}

func TestFirstFetchFailureIsFailed(t *testing.T) {
	d := NewBaseDisplay(1, "test", "Test", true)
	d.BeginFetch(&WeatherParams{})
	d.SetStatus(StatusFailed)
	if d.Status() != StatusFailed {
		t.Fatalf("first-load failure should be Failed, got %v", d.Status())
	}
}

func TestTimingScreenIndexSequence(t *testing.T) {
	// radar-style {time, si} sequence
	tm := Timing{
		BaseDelayMS: 350,
		Delay: []ScreenDelay{
			{Time: 4, ScreenIndex: 5},
			{Time: 1, ScreenIndex: 0},
			{Time: 1, ScreenIndex: 1},
		},
	}
	tm.CalcNavTiming()
	if tm.TotalScreens != 3 {
		t.Fatalf("expected 3 screens, got %d", tm.TotalScreens)
	}
	if si, ok := tm.ScreenIndexFromBaseCount(0); !ok || si != 5 {
		t.Fatalf("baseCount 0: expected si 5, got %d (ok=%v)", si, ok)
	}
	if si, ok := tm.ScreenIndexFromBaseCount(4); !ok || si != 0 {
		t.Fatalf("baseCount 4: expected si 0, got %d (ok=%v)", si, ok)
	}
	if _, ok := tm.ScreenIndexFromBaseCount(6); ok {
		t.Fatal("baseCount past end should report not-ok (advance to next display)")
	}
}
