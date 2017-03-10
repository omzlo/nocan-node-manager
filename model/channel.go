package model

import (
	"errors"
	"pannetrat.com/nocan/bitmap"
	"pannetrat.com/nocan/clog"
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
	Port   *Port
}

func NewChannelModel() *ChannelModel {
	tm := &ChannelModel{
		Names: make(map[string]Channel),
		Port:  PortManager.CreatePort("channels"),
	}
	return tm
}

func (tm *ChannelModel) Each(fn func(Channel, *ChannelState)) {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	for i := 0; i < 64; i++ {
		if tm.States[i] != nil {
			fn(Channel(i), tm.States[i])
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

func (tm *ChannelModel) Publish(channel Channel, content []byte) bool {
	if tm.SetContent(channel, content) {
		tm.Port.SendMessage(NewPublishMessage(0, channel, content))
		return true
	}
	return false
}

func (tm *ChannelModel) getState(channel Channel) *ChannelState {
	if channel < 0 || channel > 63 {
		return nil
	}
	return tm.States[channel]
}

func (tm *ChannelModel) Run() {
	for {
		m := <-tm.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_CHANNEL_REGISTER:
				var channel_id Channel
				var err error

				channel_expanded, ok := Nodes.ExpandKeywords(m.Id.GetNode(), string(m.Data))
				if ok {
					channel_id, err = tm.Register(channel_expanded)
					if err != nil {
						clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to register channel %s (expanded from %s) for node %d, %s", channel_expanded, string(m.Data), m.Id.GetNode(), err.Error())
					} else {
						clog.Info("NOCAN_SYS_CHANNEL_REGISTER: Registered channel %s for node %d as %d", channel_expanded, m.Id.GetNode(), channel_id)
					}
				} else {
					channel_id = -1
					clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to expand channel name '%s' for node %d", string(m.Data), m.Id.GetNode())
				}
				msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_REGISTER_ACK, uint8(channel_id), nil)
				tm.Port.SendMessage(msg)
			case NOCAN_SYS_CHANNEL_LOOKUP:
				var bitmap [8]byte
				if tm.Lookup(string(m.Data), bitmap[:]) {
					clog.Info("NOCAN_SYS_CHANNEL_LOOKUP: Node %d succesfully found bitmap for channel %s", m.Id.GetNode(), string(m.Data))
					msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0, bitmap[:])
					tm.Port.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_LOOKUP: Node %d failed to find bitmap for channel %s", m.Id.GetNode(), string(m.Data))
					msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0xFF, nil)
					tm.Port.SendMessage(msg)
				}
			case NOCAN_SYS_CHANNEL_UNREGISTER:
				var rval uint8
				if tm.Unregister(Channel(m.Id.GetSysParam())) {
					clog.Info("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d successfully unregistered channel %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d failed to unregister channel %d", m.Id.GetNode(), m.Id.GetSysParam())
					rval = 0xFF
				}
				msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_UNREGISTER_ACK, rval, nil)
				tm.Port.SendMessage(msg)
			}
		} else if m.Id.IsPublish() {
			tm.SetContent(m.Id.GetChannel(), m.Data)
		}
	}

}
