package nocan

type Channel int

type ChannelState struct {
}

func (cs *ChannelState) Send(m *Message) error {
	return nil
}

func (cs *ChannelState) ProcessInput(Input chan<- Message) {
	for {
		// ...
	}
}

type ChannelManager struct {
    Input chan Message
	Channels []ChannelState
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{}
}

func (cm *ChannelManager) Dispatch(m *Message) {
    for i := 0; i < len(cm.Channels); i++ {
		if Channel(i) != m.GetChannel() {
			// TODO: manage errors
			cm.Channels[i].Send(m)
		}
	}
}

func (cm *ChannelManager) runListenAndServe() {
    for i := 0; i < len(cm.Channels); i++ {
        go cm.Channels[i].ProcessInput(cm.Input)
    }
}
