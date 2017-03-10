package model

/*
#include "serial_can.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"
import "fmt"
import "encoding/hex"

import "pannetrat.com/nocan/clog"

const (
	SERIAL_HEADER_PACKET               = 0x0F
	SERIAL_HEADER_SUCCESS              = 0x10
	SERIAL_HEADER_REQUEST_SOFT_RESET   = 0x20
	SERIAL_HEADER_REQUEST_HARD_RESET   = 0x30
	SERIAL_HEADER_REQUEST_POWER_STATUS = 0x40
	SERIAL_HEADER_SET_POWER            = 0x51
	SERIAL_HEADER_SET_CAN_RES          = 0x61
	SERIAL_HEADER_REQUEST_DEBUG        = 0x72
	SERIAL_HEADER_VERSION              = 0x80
	SERIAL_HEADER_FAIL                 = 0xE0
	SERIAL_HEADER_COMMAND_UNKNOWN      = 0xF0
)

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

func (sc *SerialCan) Write(p []byte) (int, error) {
	var block [16]C.uchar
	if len(p) > 16 {
		return 0, fmt.Errorf("Serial write: serial frame is too long (%d>16)", len(p))
	}
	if len(p) == 0 {
		return 0, fmt.Errorf("Serial write: frame is empty")
	}

	for i := 0; i < len(p); i++ {
		block[i] = C.uchar(p[i])
	}
	if C.serial_can_send(sc.fd, &block[0]) > 0 {
		clog.Debug("SUCCESS Sending serial frame [%s]", hex.EncodeToString(p))
		return len(p), nil
	}
	clog.Debug("FAILED Sending serial frame [%s]", hex.EncodeToString(p))
	return 0, fmt.Errorf("Serial write: failed")
}

func (sc *SerialCan) Read(p []byte) (int, error) {
	var data [16]C.uchar
	var i int

	if C.serial_can_recv(sc.fd, &data[0]) == 0 {
		clog.Warning("FAILED Receiving serial frame (first byte=%x)", data[0])
		return 0, fmt.Errorf("Serial read: failed")
	}
	p[0] = byte(data[0])
	for i = 1; (i < 1+int(data[0]&0xF)) && (i < len(p)); i++ {
		p[i] = byte(data[i])
	}

	clog.Debug("SUCCESS Receiving serial frame [%s]", hex.EncodeToString(p[0:i]))
	return i, nil
}

func (sc *SerialCan) Status() (int, bool) {
	var status C.int
	if C.serial_can_status(sc.fd, &status) != 0 {
		clog.Warning("FAILED to get serial status")
		return 0, false
	}
	return int(status), true
}
