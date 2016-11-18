package model

import (
	"errors"
	"pannetrat.com/nocan/bitmap"
	"sync"
	"time"
)

type Topic int8

type TopicState struct {
	Name        string
	ValueLength int
	Value       [64]byte
	UpdatedAt   time.Time
}

type TopicModel struct {
	Mutex  sync.RWMutex
	States [64]*TopicState
	Names  map[string]Topic
}

func NewTopicModel() *TopicModel {
	tm := &TopicModel{}
	tm.Names = make(map[string]Topic)
	return tm
}

func (tm *TopicModel) Each(fn func(Topic, *TopicState, interface{}), data interface{}) {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	for i := 0; i < 64; i++ {
		if tm.States[i] != nil {
			fn(Topic(i), tm.States[i], data)
		}
	}
}

func (tm *TopicModel) Register(topicName string) (Topic, error) {
	var i Topic

	if len(topicName) == 0 {
		return Topic(-1), errors.New("Topic cannot be empty")
	}

	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	if i, ok := tm.Names[topicName]; ok {
		return i, nil
	}
	for i = 0; i < 64; i++ {
		if tm.States[i] == nil {
			tm.States[i] = &TopicState{Name: topicName, UpdatedAt: time.Now()}
			tm.Names[topicName] = i
			return i, nil
		}
	}
	return Topic(-1), errors.New("Maximum numver of topics has been reached")
}

func (tm *TopicModel) Unregister(topic Topic) bool {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	ts := tm.getState(topic)
	if ts == nil {
		return false
	}
	delete(tm.Names, ts.Name)
	ts.Name = ""
	return true
}

func (tm *TopicModel) Lookup(topicName string, topic_bitmap []byte) bool {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	// TODO: extend with '+',attributes, etc.
	bitmap.Bitmap64Fill(topic_bitmap, 0)
	if i, ok := tm.Names[topicName]; ok {
		bitmap.Bitmap64Set(topic_bitmap, uint(i))
		return true
	}
	return false
}

func (tm *TopicModel) FindByName(topicName string) Topic {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	if i, ok := tm.Names[topicName]; ok {
		return i
	}
	return Topic(-1)
}

func (tm *TopicModel) GetContent(topic Topic) ([]byte, bool) {
	tm.Mutex.RLock()
	defer tm.Mutex.RUnlock()

	ts := tm.getState(topic)
	if ts == nil {
		return nil, false
	}
	return ts.Value[:ts.ValueLength], true
}

func (tm *TopicModel) SetContent(topic Topic, content []byte) bool {
	tm.Mutex.Lock()
	defer tm.Mutex.Unlock()

	ts := tm.getState(topic)
	if ts == nil || len(content) > 64 {
		return false
	}
	copy(ts.Value[:], content)
	ts.ValueLength = len(content)
	return true
}

func (tm *TopicModel) getState(topic Topic) *TopicState {
	if topic < 0 || topic > 63 {
		return nil
	}
	return tm.States[topic]
}
