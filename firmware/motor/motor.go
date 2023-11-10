//go:build !dummy

package motor

import (
	"encoding/binary"
	"runtime"

	"tinygo.org/x/drivers/mcp2515"
)

func ReadFrame(can *mcp2515.Device) (*mcp2515.CANMsg, error) {
	for !can.Received() {
		runtime.Gosched()
	}
	return can.Rx()
}

type MotorState struct {
	Verocity  int16 // -220 .. 220 rpm
	Current   int16 // -32767 .. 32767 = -33 .. 33 A
	Angle     int32 // -49151 .. 49151 = -540 .. 540 deg
	Custom    byte
	Reserve   byte
	lastAngle uint16
	offset    int32
	angle     uint16 // 0 .. 32767 = 0 .. 360 deg
	adjust    int32
}

func (ms *MotorState) UnmarshalBinary(b []byte) error {
	ms.Verocity = -int16(binary.BigEndian.Uint16(b[0:2]))
	ms.Current = -int16(binary.BigEndian.Uint16(b[2:4]))
	ms.angle = binary.BigEndian.Uint16(b[4:6]) & 0x7fff
	ms.Custom = b[6]
	ms.Reserve = b[7]
	switch {
	case ms.lastAngle < 8192 && ms.angle > 24576:
		ms.offset -= 32767
	case ms.lastAngle > 24576 && ms.angle < 8192:
		ms.offset += 32767
	}
	ms.Angle = -(int32(ms.angle) + ms.offset + ms.adjust)
	ms.lastAngle = ms.angle
	return nil
}

func Setup(can *mcp2515.Device) error {
	if err := can.Tx(0x109, 8, []byte{0, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}
	_, err := ReadFrame(can)
	if err != nil {
		return err
	}
	if err := can.Tx(0x106, 8, []byte{0x80, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}
	_, err = ReadFrame(can)
	if err != nil {
		return err
	}
	if err := can.Tx(0x105, 8, []byte{0x00, 0, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}
	_, err = ReadFrame(can)
	if err != nil {
		return err
	}
	return nil
}

var state = MotorState{adjust: 0}

func SetNeutralAdjust(adjDeg float32) {
	state.adjust = int32(adjDeg * 32767 / 360)
}

func GetState(can *mcp2515.Device) (*MotorState, error) {
	if err := can.Tx(0x107, 8, []byte{0x01, 0x01, 0x02, 0x04, 0x55, 0, 0, 0}); err != nil {
		return nil, err
	}
	msg, err := ReadFrame(can)
	if err != nil {
		return nil, err
	}
	state.UnmarshalBinary(msg.Data)
	return &state, nil
}

var buf = make([]byte, 8)

func Output(can *mcp2515.Device, pow int16) error {
	binary.BigEndian.PutUint16(buf[0:2], uint16(-pow))
	return can.Tx(0x32, uint8(len(buf)), buf)
}
