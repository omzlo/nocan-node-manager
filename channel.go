package nocan

type Channel int

type ChannelState struct {
    Channel
    Driver *CanDriver
}

func (cs *ChannelState) GatherInput(input chan<- Message) {
	for {
		m, err := cs.Driver.ProcessInput()
        if err==nil {
            m.Channel = cs.channel
            input <- *m
        } else {
            return
        }
	}
}

func (cs *ChannelState) DispatchOutput(output <-chan Message) {
    for {
        m := <- output
        err := cs.Driver.ProcessOutput(&m)
        if err!=nil {
            return 
        }
    }
}

type ChannelManager struct {
    LastChannel Channel
    Input chan Message
    Output chan Message
	Channels []ChannelState
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{Input: make(chan Message,16), Output: make(chan Message, 16), Channels: make([]ChannelState,0,4)}
}

func (cm *ChannelManager) AddChannel(driver *CanDriver) Channel {
    retval := cm.LastChannel++
    append(cm.Channels, &ChannelState{retval,driver})
    return retval
}

func (cm *ChannelManager) ListenAndServe() {
    for i := 0; i < len(cm.Channels); i++ {
        go cm.Channels[i].GatherInput(cm.Input)
        go cm.Channels[i].DispatchOuput(cm.Ouput)
    }
    for {
        select {
            case m := <- cm.Input:
                for i := 0; i<len(cm.Channels); i++ {
                    if Channel(i) != m.GetChannel() {
                        cm.Channels[i].Output <- m
                    }
                }
            // add signal channel ? for errors and timers ?
        }
    }
}
