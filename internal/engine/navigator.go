package engine

import (
	"sort"
	"sync"
)

type Navigator struct {
	mu       sync.Mutex
	displays []Display
	playing  bool
	current  int
	speed    float64
	onStatus func()
}

func NewNavigator(speed float64) *Navigator {
	if speed <= 0 {
		speed = 1.0
	}
	return &Navigator{speed: speed, current: -1}
}

func (n *Navigator) Register(d Display) {
	d.SetNavCallback(func(id string, resp NavResponse) {
		n.handleNav(id, resp)
	})
	n.displays = append(n.displays, d)
	sort.Slice(n.displays, func(i, j int) bool {
		return n.displays[i].NavID() < n.displays[j].NavID()
	})
}

func (n *Navigator) Displays() []Display {
	return n.displays
}

func (n *Navigator) SetStatusCallback(fn func()) {
	n.onStatus = fn
}

func (n *Navigator) SetSpeed(speed float64) {
	if speed <= 0 {
		speed = 1.0
	}
	n.speed = speed
}

func (n *Navigator) IsPlaying() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.playing
}

func (n *Navigator) SetPlaying(v bool) {
	n.mu.Lock()
	n.playing = v
	n.mu.Unlock()
	if v {
		n.NavToFirst()
	} else {
		n.stopAllNav()
	}
}

func (n *Navigator) CurrentDisplay() Display {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.current < 0 || n.current >= len(n.displays) {
		return nil
	}
	return n.displays[n.current]
}

func (n *Navigator) HideAll() {
	for _, d := range n.displays {
		d.OnHide()
	}
}

func (n *Navigator) stopAllNav() {
	for _, d := range n.displays {
		d.StopNav()
	}
}

func (n *Navigator) firstEnabledIndex() int {
	for i, d := range n.displays {
		if d.Enabled() && d.Timing().TotalScreens > 0 && d.Status() == StatusLoaded {
			return i
		}
	}
	return -1
}

func (n *Navigator) nextEnabled(from int) int {
	for i := from + 1; i < len(n.displays); i++ {
		if n.isPlayable(n.displays[i]) {
			return i
		}
	}
	for i := 0; i < from; i++ {
		if n.isPlayable(n.displays[i]) {
			return i
		}
	}
	return -1
}

func (n *Navigator) prevEnabled(from int) int {
	for i := from - 1; i >= 0; i-- {
		if n.isPlayable(n.displays[i]) {
			return i
		}
	}
	for i := len(n.displays) - 1; i > from; i-- {
		if n.isPlayable(n.displays[i]) {
			return i
		}
	}
	return -1
}

func (n *Navigator) isPlayable(d Display) bool {
	if !d.Enabled() {
		return false
	}
	if d.Status() != StatusLoaded && d.Status() != StatusRetrying {
		return false
	}
	if d.Timing().TotalScreens == 0 {
		return false
	}
	return true
}

func (n *Navigator) handleNav(id string, resp NavResponse) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.playing {
		return
	}
	idx := n.current
	switch resp {
	case NavRespNext:
		next := n.nextEnabled(idx)
		if next >= 0 {
			n.showIndex(next, NavFirstFrame)
		}
	case NavRespPrevious:
		prev := n.prevEnabled(idx)
		if prev >= 0 {
			n.showIndex(prev, NavLastFrame)
		}
	}
}

func (n *Navigator) showIndex(idx int, cmd NavCommand) {
	if idx < 0 || idx >= len(n.displays) {
		return
	}
	if n.current >= 0 && n.current < len(n.displays) {
		n.displays[n.current].OnHide()
	}
	n.current = idx
	d := n.displays[idx]
	d.OnShow()
	if cmd == NavFirstFrame {
		d.NavNext(NavFirstFrame)
	} else {
		d.NavPrev(NavLastFrame)
	}
	d.StartNav(n.speed)
}

// ShowDisplayByID jumps directly to a display by id (used by screenshot mode).
func (n *Navigator) ShowDisplayByID(id string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, d := range n.displays {
		if d.ID() == id {
			n.showIndex(i, NavFirstFrame)
			return true
		}
	}
	return false
}

func (n *Navigator) NavToFirst() {
	idx := n.firstEnabledIndex()
	if idx < 0 {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.showIndex(idx, NavFirstFrame)
}

func (n *Navigator) NavNextManual() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.current < 0 {
		n.showIndex(n.firstEnabledIndex(), NavFirstFrame)
		return
	}
	next := n.nextEnabled(n.current)
	if next >= 0 {
		n.showIndex(next, NavFirstFrame)
	}
}

func (n *Navigator) NavPrevManual() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.current < 0 {
		return
	}
	prev := n.prevEnabled(n.current)
	if prev >= 0 {
		n.showIndex(prev, NavLastFrame)
	}
}

func (n *Navigator) LoadedCount() int {
	count := 0
	for _, d := range n.displays {
		if d.Status() != StatusLoading {
			count++
		}
	}
	return count
}

func (n *Navigator) NotifyStatus() {
	if n.onStatus != nil {
		n.onStatus()
	}
	if n.playing {
		first := n.firstEnabledIndex()
		if first >= 0 {
			d := n.displays[first]
			if d.Status() == StatusLoaded && n.current < 0 {
				n.NavToFirst()
			}
		}
	}
}

func (n *Navigator) FetchAll(params *WeatherParams) {
	for _, d := range n.displays {
		go func(display Display) {
			_ = display.Fetch(params)
			n.NotifyStatus()
		}(d)
	}
}
