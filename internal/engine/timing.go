package engine

type LoadStatus int

const (
	StatusDisabled LoadStatus = iota
	StatusLoading
	StatusLoaded
	StatusFailed
	StatusNoData
	StatusRetrying
)

func (s LoadStatus) String() string {
	switch s {
	case StatusDisabled:
		return "Disabled"
	case StatusLoading:
		return "Loading"
	case StatusLoaded:
		return "Loaded"
	case StatusFailed:
		return "Failed"
	case StatusNoData:
		return "No Data"
	case StatusRetrying:
		return "Retrying"
	default:
		return "Unknown"
	}
}

type NavCommand int

const (
	NavFirstFrame NavCommand = iota
	NavLastFrame
	NavNext
	NavPrevious
)

type NavResponse int

const (
	NavRespNext NavResponse = iota
	NavRespPrevious
)

type ScreenDelay struct {
	Time        int
	ScreenIndex int
}

type Timing struct {
	TotalScreens int
	BaseDelayMS  int
	Delay        interface{}
	fullDelay    []int
	screenIndex  []int
}

func (t *Timing) CalcNavTiming() {
	if t == nil || t.Delay == nil {
		return
	}

	switch d := t.Delay.(type) {
	case int:
		t.TotalScreens = max(t.TotalScreens, 1)
		full := make([]int, t.TotalScreens)
		for i := range full {
			full[i] = (i + 1) * d
		}
		t.fullDelay = full
		t.screenIndex = make([]int, t.TotalScreens)
		for i := range t.screenIndex {
			t.screenIndex[i] = i
		}
	case []int:
		t.TotalScreens = len(d)
		sum := 0
		t.fullDelay = make([]int, len(d))
		for i, val := range d {
			sum += val
			t.fullDelay[i] = sum
		}
		t.screenIndex = make([]int, len(d))
		for i := range t.screenIndex {
			t.screenIndex[i] = i
		}
	case []ScreenDelay:
		t.TotalScreens = len(d)
		sum := 0
		t.fullDelay = make([]int, len(d))
		t.screenIndex = make([]int, len(d))
		for i, val := range d {
			sum += val.Time
			t.fullDelay[i] = sum
			t.screenIndex[i] = val.ScreenIndex
		}
	}
}

func (t *Timing) ScreenIndexFromBaseCount(baseCount int) (int, bool) {
	if t == nil || t.TotalScreens == 0 {
		return 0, false
	}
	if len(t.fullDelay) == 0 {
		t.CalcNavTiming()
	}
	for i, delay := range t.fullDelay {
		if delay > baseCount {
			return t.screenIndex[i], true
		}
	}
	return 0, false
}

func (t *Timing) NavNextBaseCount(current int) int {
	if len(t.fullDelay) == 0 {
		t.CalcNavTiming()
	}
	for _, delay := range t.fullDelay {
		if delay > current {
			return delay
		}
	}
	return current
}

func (t *Timing) NavPrevBaseCount(current int) int {
	if len(t.fullDelay) == 0 {
		t.CalcNavTiming()
	}
	result := 0
	for _, delay := range t.fullDelay {
		if delay < current {
			result = delay
		}
	}
	return result
}

func (t *Timing) LastFrameBaseCount() int {
	if len(t.fullDelay) == 0 {
		t.CalcNavTiming()
	}
	if len(t.fullDelay) == 0 {
		return 0
	}
	return t.fullDelay[len(t.fullDelay)-1] - 1
}
