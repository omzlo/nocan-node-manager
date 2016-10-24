package nocan

/*
#include "serial_can.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"
import "fmt"

import "pannetrat.com/nocan/clog"

type SerialCan struct {
	fd C.int
}

func SerialCanOpen(device string) (*SerialCan, error) {
	dev := C.CString(device)
	defer C.free(unsafe.Pointer(dev))

	fd := C.serial_can_open(dev)
	if fd < 0 {
		clog.Error("FAILED opening %s", device)
		return nil, fmt.Errorf("Could not open %s", device)
	}
	clog.Debug("SUCCESS opening %s", device)
	return &SerialCan{fd}, nil
}

func (sc *SerialCan) Close() {
	clog.Debug("Closing serial device")
	C.serial_can_close(sc.fd)
}

func (sc *SerialCan) Send(frame *CanFrame) bool {
	var block [13]C.uchar

	block[0] = C.uchar(frame.CanId >> 24)
	block[1] = C.uchar(frame.CanId >> 16)
	block[2] = C.uchar(frame.CanId >> 8)
	block[3] = C.uchar(frame.CanId)
	block[4] = C.uchar(frame.CanDlc)
	for i := 0; i < 8; i++ {
		block[5+i] = C.uchar(frame.CanData[i])
	}

	if C.serial_can_send(sc.fd, &block[0]) > 0 {
		clog.Debug("SUCCESS Sending serial frame %s", frame.String())
		return true
	}
	clog.Debug("FAILED Sending serial frame %s", frame.String())
	return false
}

func (sc *SerialCan) Recv(frame *CanFrame) bool {
	var data [13]C.uchar
	if C.serial_can_recv(sc.fd, &data[0]) == 0 {
		clog.Debug("FAILED Receiving serial frame %s", frame.String())
		return false
	}

	frame.CanId = (CanId(data[0]) << 24) | (CanId(data[1]) << 16) | (CanId(data[2]) << 8) | CanId(data[3])
	frame.CanDlc = uint8(data[4])
	for i := 0; i < 8; i++ {
		frame.CanData[i] = uint8(data[5+i])
	}
	clog.Debug("SUCCESS Receiving serial frame %s", frame.String())
	return true
}
