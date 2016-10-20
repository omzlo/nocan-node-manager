package nocan

import (
    "fmt"
	"io"
	"net"
	// "errors"
)

type CanId uint32

type CanFrame struct {
	CanId
	CanDlc  uint8
	CanData [8]uint8
}

const (
    CANID_MASK_EXTENDED = (1<<31)
    CANID_MASK_REMOTE = (1<<30)
    CANID_MASK_ERROR = (1<<29)
    CANID_MASK_FIRST = (1<<28)
    CANID_MASK_LAST = (1<<20)
    CANID_MASK_SYSTEM = (1<<19)
    CANID_MASK_MESSAGE = ^(CanId((1<<28)|(1<<20)))
)

func ReadCanFrame(r io.Reader, frame *CanFrame) error {
	var buf [13]byte

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	frame.CanId = (CanId(buf[0]) << 24) | (CanId(buf[1]) << 16) | (CanId(buf[2]) << 8) | CanId(buf[3])
	frame.CanDlc = buf[4]
    copy(frame.CanData[:], buf[5:])
	return nil
}

func WriteCanFrame(w io.Writer, frame *CanFrame) error {
	var buf [13]byte

	buf[0] = byte(frame.CanId >> 24)
	buf[1] = byte(frame.CanId >> 16)
	buf[2] = byte(frame.CanId >> 8)
	buf[3] = byte(frame.CanId)
	buf[4] = frame.CanDlc
	copy(buf[5:], frame.CanData[:])

	_, err := w.Write(buf[:])
	// write must return non nil err if it does not write len(buf) bytes
	return err
}

func (canid CanId) IsFirst() bool {
    return (canid&CANID_MASK_FIRST)!=0
}

func (canid CanId) IsLast() bool {
    return (canid&CANID_MASK_LAST)!=0
}

func (canid CanId) IsExtended() bool {
    return (canid&CANID_MASK_EXTENDED)!=0
}

func (canid CanId) IsError() bool {
    return ((canid>>29)&0x1)!=0
}

func (canid CanId) IsRemote() bool {
    return ((canid>>30)&0x1)!=0
}

func (canid CanId) IsControl() bool {
    return (canid&0xE0000000)==0xE0000000
}

func (canid CanId) IsSystem() bool {
    return (canid&CANID_MASK_SYSTEM)!=0
}

func (canid CanId) GetNode() Node {
    return Node((canid>>21)&0x7F)
}

func (canid CanId) SetNode(node Node) CanId {
    canid &= ^(CanId(0x7F<<21))
    canid |= CanId(node)<<21
    return canid
}

func (canid CanId) GetSysFunc() uint8 {
    return uint8(canid>>8)
}

func (canid CanId) SetSysFunc(fn uint8) CanId {
    canid &= CanId(0xFFFF00FF)
    canid |= CanId(fn)<<8
    return canid
}

func (canid CanId) GetSysParam() uint8 {
    return uint8(canid)
}

func (canid CanId) SetSysParam(pm uint8) CanId {
    canid &= CanId(0xFFFFFF00)
    canid |= CanId(pm)
    return canid
}

func (canid CanId) GetTopic() Topic {
    var i uint8
    var base uint8 = uint8(((canid>>16)&0x03)<<4);
    for i=0; i<16; i++ {
        if (canid&(1<<i))!=0 {
            return Topic(base+i)
        }
    }
    return -1
}

func (canid CanId) SetTopic(topic Topic) CanId {
    t := ((uint32(topic)>>4)<<16)|(1<<(uint32(topic)&0xF))
    canid &= CanId(0xFFFC0000)
    canid |= CanId(t)
    return canid
}

func (canid CanId) String() string {
    s := fmt.Sprintf("%04x<n=%d",uint(canid),canid.GetNode())
    if canid.IsFirst() {
        s+=",first"
    }
    if canid.IsLast() {
        s+=",last"
    }
    if canid.IsSystem() {
        s+=fmt.Sprintf(",sys,fn=%d,pm=%d",canid.GetSysFunc(),canid.GetSysParam()) 
    } else {
        s+=fmt.Sprintf(",pub,topic=%d",canid.GetTopic())
    }
    return s+">"
}

type CanDriver struct {
	Conn         net.Conn
	InputBuffer  [128]*Message
	OutputBuffer Message
	PowerStatus  struct {
		PowerOn    bool
		PowerLevel uint16
		SenseOn    bool
		SenseLevel uint16
	}
}

func (cd *CanDriver) ControlReset() bool {
	// TODO:
	return true
}

func (cd *CanDriver) ControlPower(on bool) bool {
	// TODO:
	return true
}

func (cd *CanDriver) ControlUpdatePowerStatus() bool {
	// TODO:
	return true
}
func (cd *CanDriver) ConstrolResistor(on bool) bool {
	// TODO:
	return true
}

func NewCanDriver(network, address string) (*CanDriver, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return &CanDriver{Conn: conn}, nil
}

func (cd *CanDriver) ProcessInput() (*Message, error) {
	var frame CanFrame

	if err := ReadCanFrame(cd.Conn, &frame); err != nil {
		return nil, err
	}

	node := frame.CanId.GetNode()
	switch {
	case !frame.CanId.IsExtended(), frame.CanId.IsRemote():
		Log(WARNING, "Got malformed frame, discarding.")
		return nil, nil
	case frame.CanId.IsControl():

	case frame.CanId.IsError():

	default:
		if frame.CanId.IsFirst() {
			if cd.InputBuffer[node] != nil {
				Log(WARNING, "Got frame with inconsistent first bit indicator, discarding.")
				return nil, nil
			}
			cd.InputBuffer[node] = NewMessageFromFrame(&frame)
		} else {
			if cd.InputBuffer[node] == nil {
				Log(WARNING, "Got first frame with missing first bit indicator, discarding.")
				return nil, nil
			}
			cd.InputBuffer[node].AppendData(frame.CanData[:frame.CanDlc])
		}
		if frame.CanId.IsLast() {
			retval := cd.InputBuffer[node]
			cd.InputBuffer[node] = nil
			return retval, nil
		}
	}
	return nil, nil
}

func (cd *CanDriver) ProcessOutput(m *Message) error {
	cd.OutputBuffer = *m
    return nil
}

func (cd *CanDriver) Close() {
    cd.Conn.Close()
}

type CanDriverModel struct {
   // TODO 
}
