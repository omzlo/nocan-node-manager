package nocan

type Message struct {
    channel Channel
    id uint32
    length uint8
    data [64]byte
}

func NewMessage(id uint32, data []byte) {
    m := &Message{id: id}
    m.length = uint8(len(data))
    copy(m.data[:],data)
}

func (m* Message) AppendData(data []byte) bool {
    if m.length+len(data)>64 {
        return false
    }
    copy(m.data[m.length:],data)
    m.length+=len(data)
    return true
}

func (m* Message) Node() Node {
    return Node((m.id>>22)&0x7F)
}

func (m* Message) IsSystem() bool {
    return ((m.id>>19)&1)==1
}

func (m *Message) GetChannel() Channel {
    return m.channel
}

func (m* Message) GetTopic() Topic {
    var i uint8
    var base uint8 = uint8(((m.id>>16)&0x03)<<4);
    for i=0; i<16; i++ {
        if (m.id&(1<<i))!=0 {
            return Topic(base+i)
        }
    }
    return -1
}

func (m* Message) GetSysFunc() uint8 {
    return uint8(m.id>>8)
}

func (m* Message) GetSysParam() uint8 {
    return uint8(m.id)
}

func (m *Message) SetSysFunc(f uint8) {
    m.id &= 0xFFFF00FF
    m.id |= uint32(f)<<8
}

func (m *Message) SetSysParam(p uint8) {
    m.id &= 0xFFFFFF00;
    m.id |= uint32(p);
}

func (m *Message) Length() uint8 {
    return m.length;
}

func (m *Message) GetData() []byte {
    return m.data[:]
}

func (m *Message) SetData(d []byte) {
    if len(d)>64 {
        copy(m.data[:],d[:64])
        m.length = 64
    } else {
        copy(m.data[:],d)
        m.length = uint8(len(d))
    }
}

