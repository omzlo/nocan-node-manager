package nocan

import (
	"errors"
	"time"
)

type Topic int8

type TopicState struct {
	Name        string
	ValueLength int
	Value       [64]byte
	UpdatedAt   time.Time
}

type TopicManager struct {
	States [64]TopicState
	Names  map[string]Topic
}

func NewTopicManager() *TopicManager {
	tm := &TopicManager{}
	tm.Names = make(map[string]Topic)
	return tm
}

func (tm *TopicManager) Register(topicName string) (Topic, error) {
	var i Topic

	if i, ok := tm.Names[topicName]; ok {
		return i, nil
	}
	for i = 0; i < 64; i++ {
		if tm.States[i].Name == "" {
			tm.States[i].Name = topicName
			tm.States[i].UpdatedAt = time.Now()
			tm.Names[topicName] = i
			return i, nil
		}
	}
	return Topic(-1), errors.New("Maximum numver of topics has been reached")
}

func (tm *TopicManager) Unregister(topic Topic) bool {
	ts := tm.getState(topic)
	if ts == nil {
		return false
	}
	delete(tm.Names, ts.Name)
	ts.Name = ""
	return true
}

func (tm *TopicManager) FindByName(topicName string) Topic {
	if i, ok := tm.Names[topicName]; ok {
		return i
	}
	return Topic(-1)
}

func (tm *TopicManager) GetContent(topic Topic) ([]byte, bool) {
	ts := tm.getState(topic)
	if ts == nil {
		return nil, false
	}
	return ts.Value[:ts.ValueLength], true
}

func (tm *TopicManager) SetContent(topic Topic, content []byte) bool {
	ts := tm.getState(topic)
	if ts == nil || len(content) > 64 {
		return false
	}
	copy(ts.Value[:], content)
	ts.ValueLength = len(content)
	return true
}

func (tm *TopicManager) getState(topic Topic) *TopicState {
	if topic < 0 || topic > 63 {
		return nil
	}
	if len(tm.States[topic].Name) > 0 {
		return &tm.States[topic]
	}
	return nil
}
