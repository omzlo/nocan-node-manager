package nocan

import (
	"pannetrat.com/nocan/clog"
	"pannetrat.com/nocan/model"
	"pannetrat.com/nocan/serialcan"
	"time"
)

const (
	SERIAL_CAN_CTRL_RESET              = 1
	SERIAL_CAN_CTRL_GET_REGISTERS      = 2
	SERIAL_CAN_CTRL_SET_POWER          = 3
	SERIAL_CAN_CTRL_GET_POWER_STATUS   = 4
	SERIAL_CAN_CTRL_RESISTOR_CONFIGURE = 5
)

type SerialCanRequest struct {
	Status uint8
}

const (
	SERIAL_CAN_REQUEST_STATUS_UNKNOWN   = 0
	SERIAL_CAN_REQUEST_STATUS_SUBMITTED = 1
	SERIAL_CAN_REQUEST_STATUS_SUCCESS   = 2
	SERIAL_CAN_REQUEST_STATUS_FAILURE   = 2
)

type SerialTask struct {
	BaseTask
	Serial        *serialcan.SerialCan
	Device        string
	InputBuffer   [128]*model.Message
	OutputBuffer  model.Message
	Ticker        *time.Ticker
	RequestStatus [6]SerialCanRequest
	PowerStatus   struct {
		PowerOn      bool
		PowerLevel   float32
		SenseOn      bool
		SenseLevel   float32
		UsbReference float32
	}
	RescueMode bool
}

func NewSerialTask(pm *model.PortManager, device string) *SerialTask {
	serial, err := serialcan.SerialCanOpen(device)
	if err != nil {
		clog.Error("Could not open %s: %s", device, err.Error())
		return nil
	}
	clog.Debug("Opened device %s", device)

	se := &SerialTask{BaseTask: BaseTask{pm, pm.CreatePort("serial")}, Serial: serial, Device: device}

	return se
}

func (se *SerialTask) Close() {
	se.Serial.Close()
}

func (se *SerialTask) Rescue() bool {
	se.Close()
	for {
		serial, err := serialcan.SerialCanOpen(se.Device)
		if err == nil {
			se.Serial = serial
			clog.Info("Reopened device %s", se.Device)
			se.SendReset()
			return true
		} else {
			clog.Warning("Failed to reopen device %s: %s", se.Device, err.Error())
		}
		time.Sleep(10 * time.Second)
	}
}

/*
func (se *SerialTask) GetAttributes() interface{} {
	return &se.PowerStatus
}
*/

func (se *SerialTask) ProcessSend(port *model.Port) {
	for {
		var frame model.CanFrame

		//clog.Debug("Waiting for serial input")
		if se.Serial.Recv(&frame) == false {
			clog.Error("Failed to receive frame from %s", se.Device)
			if se.Rescue() {
				continue
			} else {
				break
			}
		}
		//clog.Debug("Got serial input")

		node := frame.CanId.GetNode()

		switch {
		case !frame.CanId.IsExtended(), frame.CanId.IsRemote():
			clog.Warning("Got malformed frame, discarding.")
		case frame.CanId.IsControl():
			switch frame.CanId.GetSysFunc() {
			case SERIAL_CAN_CTRL_RESET:
				if frame.CanId.GetSysParam() == 0 {
					se.RequestStatus[SERIAL_CAN_CTRL_RESET].Status = SERIAL_CAN_REQUEST_STATUS_SUCCESS
				} else {
					se.RequestStatus[SERIAL_CAN_CTRL_RESET].Status = SERIAL_CAN_REQUEST_STATUS_FAILURE
				}
			case SERIAL_CAN_CTRL_GET_POWER_STATUS:
				if frame.CanId.GetSysParam() == 0 {
					se.RequestStatus[SERIAL_CAN_CTRL_GET_POWER_STATUS].Status = SERIAL_CAN_REQUEST_STATUS_SUCCESS
					usbref := (uint16(frame.CanData[6]) << 8) | uint16(frame.CanData[7])
					powerlevel := (uint16(frame.CanData[1]) << 8) | uint16(frame.CanData[2])
					senselevel := (uint16(frame.CanData[4]) << 8) | uint16(frame.CanData[5])
					se.PowerStatus.PowerOn = frame.CanData[0] != 0
					se.PowerStatus.PowerLevel = float32(powerlevel) / float32(usbref) * 1.1 * 9.2
					se.PowerStatus.SenseOn = frame.CanData[3] != 0
					se.PowerStatus.SenseLevel = 100 * float32(senselevel) / 1023
					se.PowerStatus.UsbReference = 1023 * 1.1 / float32(usbref)
					clog.Info("Power stat estimates: power=%t power_level=%.1fV sense=%t sense_level=%.3f%% usb_power=%.1fV",
						se.PowerStatus.PowerOn, se.PowerStatus.PowerLevel,
						se.PowerStatus.SenseOn, se.PowerStatus.SenseLevel,
						se.PowerStatus.UsbReference)
				} else {
					se.RequestStatus[SERIAL_CAN_CTRL_GET_POWER_STATUS].Status = SERIAL_CAN_REQUEST_STATUS_FAILURE
				}
			}
		case frame.CanId.IsError():
			clog.Error("Recieved error frame on CAN controller")
		default:
			if frame.CanId.IsFirst() {
				if se.InputBuffer[node] != nil {
					clog.Warning("Got frame with inconsistent first bit indicator, discarding.")
					break
				}
				se.InputBuffer[node] = model.NewMessageFromFrame(&frame)
			} else {
				if se.InputBuffer[node] == nil {
					clog.Warning("Got first frame with missing first bit indicator, discarding.")
					break
				}
				se.InputBuffer[node].AppendData(frame.CanData[:frame.CanDlc])
			}
			if frame.CanId.IsLast() {
				port.SendMessage(se.InputBuffer[node])
				se.InputBuffer[node] = nil
			}
		}
	}
}

func (se *SerialTask) ProcessRecv(port *model.Port) {
	for {
		var frame model.CanFrame

		select {
		case m := <-port.Input:
			pos := 0
			for {
				frame.CanId = (m.Id & model.CANID_MASK_MESSAGE) | model.CANID_MASK_EXTENDED
				if pos == 0 {
					frame.CanId |= model.CANID_MASK_FIRST
				}
				if len(m.Data)-pos <= 8 {
					frame.CanId |= model.CANID_MASK_LAST
					frame.CanDlc = uint8(len(m.Data) - pos)
				} else {
					frame.CanDlc = 8
				}
				copy(frame.CanData[:], m.Data[pos:pos+int(frame.CanDlc)])
				if se.Serial.Send(&frame) == false {
					clog.Error("Failed to send frame to %s", se.Device)
					return
				}
				pos += int(frame.CanDlc)
				if pos >= len(m.Data) {
					break
				}
			}
		case <-se.Ticker.C:
			se.SendGetPowerStatus()
		}
	}
}

func (se *SerialTask) Run() {
	se.Ticker = time.NewTicker(10 * time.Second)
	go se.ProcessRecv(se.Port)
	se.SendReset()
	go func() {
		se.SendSetPower(false)
		time.Sleep(3 * time.Second)
		se.SendSetPower(true)
	}()
	se.SendGetPowerStatus()
	se.ProcessSend(se.Port)
}

func MakeControlFrame(fn uint8, param uint8, data []byte) *model.CanFrame {
	frame := &model.CanFrame{}
	frame.CanId = model.CanId(model.CANID_MASK_CONTROL).SetSysFunc(fn).SetSysParam(param)
	frame.CanDlc = uint8(len(data))
	if data != nil {
		copy(frame.CanData[:], data)
	}
	return frame
}

func (se *SerialTask) SendReset() {
	frame := MakeControlFrame(SERIAL_CAN_CTRL_RESET, 0, nil)
	se.RequestStatus[SERIAL_CAN_CTRL_RESET].Status = SERIAL_CAN_REQUEST_STATUS_SUBMITTED
	se.Serial.Send(frame)
}

func (se *SerialTask) SendGetPowerStatus() {
	frame := MakeControlFrame(SERIAL_CAN_CTRL_GET_POWER_STATUS, 0, nil)
	se.RequestStatus[SERIAL_CAN_CTRL_GET_POWER_STATUS].Status = SERIAL_CAN_REQUEST_STATUS_SUBMITTED
	se.Serial.Send(frame)
}

func (se *SerialTask) SendSetPower(powerOn bool) {
	var on_off uint8
	if powerOn {
		on_off = 1
	} else {
		on_off = 0
	}
	frame := MakeControlFrame(SERIAL_CAN_CTRL_SET_POWER, on_off, nil)
	//se.RequestStatus[SERIAL_CAN_CTRL_GET_POWER_STATUS].Status = SERIAL_CAN_REQUEST_STATUS_SUBMITTED
	se.Serial.Send(frame)
}
