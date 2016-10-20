package nocan

import (
    "fmt"
)

type Message struct {
    Port
    Id CanId
    Data []byte
}

func NewMessage(id CanId, data []byte) *Message {
    m := &Message{Port: -1, Id: id, Data: make([]byte,0,64)}
    m.Data = m.Data[:len(data)]
    copy(m.Data, data)
    return m
}

func NewMessageFromFrame(frame *CanFrame) *Message {
    return NewMessage(frame.CanId&CANID_MASK_MESSAGE, frame.CanData[:frame.CanDlc])
}

func (m* Message) String() string {
    s := fmt.Sprintf("{port:%d, %s, [",int(m.Port),m.Id)
    for i:=0; i<len(m.Data); i++ {
        if i>0 {
            s+=" "
        }
        s += fmt.Sprintf("%02x",m.Data[i])
    }
    if !m.Id.IsSystem() {
        s+=` - "`
        for i:=0; i<len(m.Data); i++ {
            if m.Data[i]>=32 && m.Data[i]<127 {
                s+=string(m.Data[i])
            } else {
                s+=`.`
            }
        }
        s+=`"`
    }

    return s + "]}"
}

func (m* Message) AppendData(data []byte) bool {
    if len(m.Data)+len(data)>64 {
        return false
    }
    m.Data = append(m.Data, data...)
    return true
}

func (m *Message) MatchSystemMessage(node Node, fn uint8) bool {
    return m.Id.IsSystem() && m.Id.GetNode()==node && m.Id.GetSysFunc()==fn
}

func NewSystemMessage(node Node, fn uint8, param uint8, value []byte) *Message {
    id := CanId(CANID_MASK_SYSTEM)
    id.SetNode(node).SetSysFunc(fn).SetSysParam(param)
    return NewMessage(id, value)
}

func NewPublishMessage(node Node, topic Topic, value []byte) *Message {
    id := CanId(0).SetNode(node).SetTopic(topic)
    return NewMessage(id, value)
}
