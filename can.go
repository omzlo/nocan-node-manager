package nocan

import (
    "net"
    "io"
    "errors"
)

type CanFrame struct {
    CanId uint32
    CanDlc uint8
    CanData [8]uint8
}

func ReadCanFrame(r io.Reader, frame *CanFrame) error {
    var buf [13]byte

    if _, err := io.ReadFull(r,buf[:]); err!=nil {
        return err
    }
    frame.CanId = (uint32(buf[0])<<24) | (uint32(buf[1])<<16) | (uint32(buf[2])<<8) | uint32(buf[3]);
    frame.Dlc = buf[4]
    copy(frame.canData,buf[5:])
    return nil
}

func WriteCanFrame(w io.Writer, frame *CanFrame) error {
    var buf [13]byte
    
    buf[0] = byte(frame.CanId>>24);
    buf[1] = byte(frame.CanId>>16);
    buf[2] = byte(frame.CanId>>8);
    buf[3] = byte(frame.CanId);
    buf[4] = frame.Dlc;
    copy(buf[5:],frame.CanData[:])

    _, err := w.Write(buf[:])
    // write must return non nil err if it does not write len(buf) bytes
    return err
}

func (frame *CanFrame) IsFirst() bool {

}

func (frame *CanFrame) IsLast() bool {

}

func (frame *CanFrame) IsExtended() bool {

}

func (frame *CanFrame) IsError() bool {

}

func (frame *CanFrame) IsRemote() bool {

}
func (frame *CanFrame) IsControl() bool {

}
func (frame *CanFrame) Node() Node {

}

type CanDriver struct {
    Conn net.Conn
    InputBuffer [128]*Message
    OutputBuffer *Message
    PowerStatus struct {
        PowerOn bool
        PowerLevel uint16
        SenseOn bool
        SenseLevel uint16
    }
}

func (cd *CanDriver) ControlReset() bool {
    // TODO:
    return true;
}

func (cd *CanDriver) ControlPower(on bool) bool {
    // TODO:
    return true;
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
    if err!=nil {
       return err
    }
    return &CanDriver{Conn: conn}
}

func (cd *CanDriver)ProcessInput() (*Message, error) {
    var frame CanFrame

    if err:=ReadCanFrame(cd.Conn,&frame); err!=nil {
        return nil, err
    }

    node := frame.Node()
    switch {
        case !frame.IsExtended(), frame.IsRemote():
        // ERROR
        case frame.IsControl():

        case frame.IsError():

        default:
            if fame.IsFirst() {
                if cd.InputBuffer[node]!= nil {
                    Log(WARNING,"Got frame with inconsistent first bit indicator, discarding.")
                    return nil, nil
                } else {
                    cd.InputBuffer[node] = NewMessage(fame.CanId, frame.CanData[:frame.CanDlc])
                }
            } else {
                if cd.InputBuffer[node]==nil {
                    log(WARNING,"Got first frame with missing first bit indicator, discarding.);
                }
                cd.InputBuffer[node].Append(
            }
            if frame.IsLast() {
                retval := cd.InputBuffer[node]
                cd.InputBuffer[node] = nil
                return retval
            }
    }
    return nil, nil // should never happen
}

func (cd *CanDriver)ProcessOutput(m *Message) error {

}

func (cd *CanDriver)Close() {

}
