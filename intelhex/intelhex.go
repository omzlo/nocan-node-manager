package intelhex

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"pannetrat.com/nocan/clog"
)

type IntelHexMemBlock struct {
	Type    uint8
	Address uint32
	Data    []byte
}

type IntelHex struct {
	Blocks []*IntelHexMemBlock
}

func New() *IntelHex {
	return &IntelHex{make([]*IntelHexMemBlock, 0, 8)}
}

func (ihex *IntelHex) Add(btype uint8, address uint32, data []byte) {

	//clog.Debug("ADD %d @%x len=%d",btype,address,len(data))
	for i, block := range ihex.Blocks {
		if btype == block.Type && address == block.Address+uint32(len(block.Data)) {
			ihex.Blocks[i].Data = append(block.Data, data...)
			return
		}
	}
	block := &IntelHexMemBlock{btype, address, make([]byte, len(data))}
	copy(block.Data, data)
	ihex.Blocks = append(ihex.Blocks, block)
}

func (ihex *IntelHex) Load(r io.Reader) error {
	var (
		byte_count       uint8
		address          uint32
		btype            uint8
		extended_address uint32 = 0
	)
	line_count := 0

	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line_count++
		line := scanner.Text()
		if line[0] != ':' {
			return fmt.Errorf("Missing ':' at the beginning of line %d", line_count)
		}
		data, err := hex.DecodeString(line[1:])
		if err != nil {
			return err
		}
		if len(data) < 5 || len(data) != (5+int(data[0])) {
			return fmt.Errorf("Missing data in hexfile on line %d", line_count)
		}
		byte_count = data[0]
		address = (uint32(data[1]) << 8) | uint32(data[2])
		btype = data[3]
		checksum := data[0]
		for i := 1; i < len(data)-1; i++ {
			checksum += data[i]
		}
		checksum = (^checksum) + 1
		if checksum != data[len(data)-1] {
			return fmt.Errorf("Checksum error on line %d, expected %02x but got %02x", line_count, data[len(data)-1], checksum)
		}
		switch btype {
		case 0:
			ihex.Add(btype, extended_address+address, data[4:4+byte_count])
		case 1:
			if byte_count != 0 {
				return fmt.Errorf("End of file marker has non zero length on line %d", line_count)
			}
			return nil
		case 2:
			if byte_count != 2 {
				return fmt.Errorf("Extended segment address record should be of length 2 on line %d", line_count)
			}
			extended_address = (uint32(data[4]) << 12) | (uint32(data[5]) << 4)
		case 3:
			if byte_count != 4 {
				return fmt.Errorf("Start segment address directive is of incorrect length on line %d", line_count)
			} else {
				clog.Debug("Ignoring Start Segment Address directive with value %04x on line %d in firmware",
					(uint32(data[4])<<24)|(uint32(data[5])<<16)|(uint32(data[6])<<8)|uint32(data[7]), line_count)
			}
		case 4:
			if byte_count != 2 {
				return fmt.Errorf("Extended linear address record should be of length 2 on line %d", line_count)
			}
			extended_address = (uint32(data[4]) << 24) | (uint32(data[5]) << 16)
		case 5:
			if byte_count != 4 {
				return fmt.Errorf("Start linear address directive is of incorrect length on line %d", line_count)
			} else {
				clog.Debug("Ignoring Start Linear Address directive with value %04x on line %d in firmware",
					(uint32(data[4])<<24)|(uint32(data[5])<<16)|(uint32(data[6])<<8)|uint32(data[7]), line_count)
			}
		default:
			clog.Warning("Firmware contains a block of unknown type %02x on line %d, whihc will be ignored.", btype, line_count)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("Unexpected end of file on line %d", line_count)
}

func (hex *IntelHex) Save(w io.Writer) error {
	return nil
}

func (hex *IntelHex) IterateBlocks(fn func(uint8, uint32, []byte, interface{}) error, extra interface{}) {

}
