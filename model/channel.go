package model

import (
	"errors"
	"pannetrat.com/nocan/clog"
	"sync"
	"time"
)

type Channel int16

type ChannelState struct {
	ChannelId   Channel
	Name        string
	ValueLength int
	Value       [64]byte
	UpdatedAt   time.Time
}

type ChannelModel struct {
	Mutex  sync.RWMutex
	ById   map[Channel]*ChannelState
	ByName map[string]*ChannelState
	Port   *Port
	TopId  Channel
}

func NewChannelModel() *ChannelModel {
	tm := &ChannelModel{
		ById:   make(map[Channel]*ChannelState),
		ByName: make(map[string]*ChannelState),
		Port:   PortManager.CreatePort("channels"),
		TopId:  0,
	}
	return tm
}

func (tm *ChannelModel) Each(fn func(Channel, *ChannelState)) {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	for k, v := range tm.ById {
		fn(k, v)
	}
}

func (tm *ChannelModel) Register(channelName string) (Channel, error) {
	if len(channelName) == 0 {
		return Channel(-1), errors.New("Channel cannot be empty")
	}

	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	if state, ok := tm.ByName[channelName]; ok {
		return state.ChannelId, nil
	}

	for {
		if tm.TopId < 0 {
			tm.TopId = 0
		}
		if state, ok := tm.ById[tm.TopId]; !ok {
			state = &ChannelState{ChannelId: tm.TopId, Name: channelName, UpdatedAt: time.Now()}
			tm.ById[tm.TopId] = state
			tm.ByName[channelName] = state
			tm.TopId++
			return state.ChannelId, nil
		}
		tm.TopId++
	}
	// never reached
	return Channel(-1), errors.New("Maximum numver of channels has been reached")
}

func (tm *ChannelModel) Unregister(channel Channel) bool {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	ts := tm.getState(channel)
	if ts == nil {
		return false
	}
	delete(tm.ByName, ts.Name)
	delete(tm.ById, ts.ChannelId)
	ts.Name = ""
	return true
}

func (tm *ChannelModel) Lookup(channelName string) (Channel, bool) {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	if state, ok := tm.ByName[channelName]; ok {
		return state.ChannelId, true
	}
	return Channel(-1), false
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
	if state, ok := tm.ById[channel]; ok {
		return state
	}
	return nil
}

func ChannelToBytes(channel_id Channel, block []uint8) {
	block[0] = uint8(channel_id >> 8)
	block[1] = uint8(channel_id & 0xFF)
}

func BytesToChannel(block []uint8) Channel {
	return (Channel(block[0]) << 8) | Channel(block[1])
}

func (tm *ChannelModel) Run() {
	var channel_id Channel
	var channel_bytes [2]uint8
	var status uint8

	for {
		m := <-tm.Port.Input

		if m.Id.IsSystem() {
			switch m.Id.GetSysFunc() {
			case NOCAN_SYS_CHANNEL_REGISTER:
				var err error

				ChannelToBytes(Channel(-1), channel_bytes[:])
				status = 0xFF

				channel_expanded, ok := Nodes.ExpandKeywords(m.Id.GetNode(), string(m.Data))

				if ok {
					channel_id, err = tm.Register(channel_expanded)
					if err != nil {
						clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to register channel %s (expanded from %s) for node %d, %s", channel_expanded, string(m.Data), m.Id.GetNode(), err.Error())
					} else {
						clog.Info("NOCAN_SYS_CHANNEL_REGISTER: Registered channel %s for node %d as %d", channel_expanded, m.Id.GetNode(), channel_id)
						ChannelToBytes(channel_id, channel_bytes[:])
						status = 0x00
					}
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_REGISTER: Failed to expand channel name '%s' for node %d", string(m.Data), m.Id.GetNode())
				}
				msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_REGISTER_ACK, status, channel_bytes[:])
				tm.Port.SendMessage(msg)
			case NOCAN_SYS_CHANNEL_LOOKUP:
				if channel_id, ok := tm.Lookup(string(m.Data)); ok {
					clog.Info("NOCAN_SYS_CHANNEL_LOOKUP: Node %d succesfully found id %d for channel %s", m.Id.GetNode(), channel_id, string(m.Data))
					ChannelToBytes(channel_id, channel_bytes[:])
					msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0x00, channel_bytes[:])
					tm.Port.SendMessage(msg)
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_LOOKUP: Node %d failed to find bitmap for channel %s", m.Id.GetNode(), string(m.Data))
					ChannelToBytes(Channel(-1), channel_bytes[:])
					msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_LOOKUP_ACK, 0xFF, channel_bytes[:])
					tm.Port.SendMessage(msg)
				}
			case NOCAN_SYS_CHANNEL_UNREGISTER:
				channel_id = BytesToChannel(m.Data[:2])
				if tm.Unregister(channel_id) {
					clog.Info("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d successfully unregistered channel %d", m.Id.GetNode(), channel_id)
					status = 0x00
				} else {
					clog.Warning("NOCAN_SYS_CHANNEL_UNREGISTER: Node %d failed to unregister channel %d", m.Id.GetNode(), channel_id)
					status = 0xFF
				}
				msg := NewSystemMessage(m.Id.GetNode(), NOCAN_SYS_CHANNEL_UNREGISTER_ACK, status, nil)
				tm.Port.SendMessage(msg)
			}
		} else if m.Id.IsPublish() {
			tm.SetContent(m.Id.GetChannel(), m.Data)
		}
	}

}
