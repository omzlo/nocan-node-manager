package model

import (
	"errors"
	"pannetrat.com/nocan/bitmap"
	"sync"
	"time"
)

type Channel int8

type ChannelState struct {
	Name        string
	ValueLength int
	Value       [64]byte
	UpdatedAt   time.Time
}

type ChannelModel struct {
	Mutex  sync.RWMutex
	States [64]*ChannelState
	Names  map[string]Channel
}

func NewChannelModel() *ChannelModel {
	tm := &ChannelModel{}
	tm.Names = make(map[string]Channel)
	return tm
}

func (tm *ChannelModel) Each(fn func(Channel, *ChannelState, interface{}), data interface{}) {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	for i := 0; i < 64; i++ {
		if tm.States[i] != nil {
			fn(Channel(i), tm.States[i], data)
		}
	}
}

func (tm *ChannelModel) Register(channelName string) (Channel, error) {
	var i Channel

	if len(channelName) == 0 {
		return Channel(-1), errors.New("Channel cannot be empty")
	}

	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	if i, ok := tm.Names[channelName]; ok {
		return i, nil
	}
	for i = 0; i < 64; i++ {
		if tm.States[i] == nil {
			tm.States[i] = &ChannelState{Name: channelName, UpdatedAt: time.Now()}
			tm.Names[channelName] = i
			return i, nil
		}
	}
	return Channel(-1), errors.New("Maximum numver of channels has been reached")
}

func (tm *ChannelModel) Unregister(channel Channel) bool {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	ts := tm.getState(channel)
	if ts == nil {
		return false
	}
	delete(tm.Names, ts.Name)
	ts.Name = ""
	return true
}

func (tm *ChannelModel) Lookup(channelName string, channel_bitmap []byte) bool {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	// TODO: extend with '+',attributes, etc.
	bitmap.Bitmap64Fill(channel_bitmap, 0)
	if i, ok := tm.Names[channelName]; ok {
		bitmap.Bitmap64Set(channel_bitmap, uint(i))
		return true
	}
	return false
}

func (tm *ChannelModel) FindByName(channelName string) Channel {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	if i, ok := tm.Names[channelName]; ok {
		return i
	}
	return Channel(-1)
}

func (tm *ChannelModel) GetContent(channel Channel) ([]byte, bool) {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	ts := tm.getState(channel)
	if ts == nil {
		return nil, false
	}
	return ts.Value[:ts.ValueLength], true
}

func (tm *ChannelModel) SetContent(channel Channel, content []byte) bool {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	ts := tm.getState(channel)
	if ts == nil || len(content) > 64 {
		return false
	}
	copy(ts.Value[:], content)
	ts.ValueLength = len(content)
	return true
}

func (tm *ChannelModel) getState(channel Channel) *ChannelState {
	if channel < 0 || channel > 63 {
		return nil
	}
	return tm.States[channel]
}
