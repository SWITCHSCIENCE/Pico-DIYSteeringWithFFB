package main

import (
	"log"
	"machine"
	"machine/usb/hid/joystick"
	"time"

	"tinygo.org/x/drivers/mcp2515"

	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/motor"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/pid"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/utils"
)

const (
	// for picossci-can pins
	LED1      machine.Pin = 25
	LED2      machine.Pin = 14
	LED3      machine.Pin = 15
	SW1       machine.Pin = 24
	SW2       machine.Pin = 23
	SW3       machine.Pin = 22
	CAN_INT   machine.Pin = 16
	CAN_RESET machine.Pin = 17
	CAN_SCK   machine.Pin = 18
	CAN_TX    machine.Pin = 19
	CAN_RX    machine.Pin = 20
	CAN_CS    machine.Pin = 21

	// Device Specific Configuration
	NeutralAdjust       = -600 // unit:360*n/32767 deg
	Lock2Lock           = 540  // unit:deg
	CoggingTorqueCancel = 128  // unit:100*n/32767 %
	CenteringForce      = 8    // unit:100*n/32767 %

	HalfLock2Lock = Lock2Lock / 2
	MaxAngle      = 32768*HalfLock2Lock/360 - 1
)

var (
	spi       = machine.SPI0
	js        = joystick.Port()
	ph        *pid.PIDHandler
	dummyMode = false
)

func init() {
	ph = pid.NewPIDHandler()
	js = joystick.UseSettings(joystick.Definitions{
		ReportID:     1,
		ButtonCnt:    24,
		HatSwitchCnt: 0,
		AxisDefs: []joystick.Constraint{
			{MinIn: -32767, MaxIn: 32767, MinOut: -32767, MaxOut: 32767},
			{MinIn: 0, MaxIn: 32767, MinOut: 0, MaxOut: 32767},
			{MinIn: 0, MaxIn: 32767, MinOut: 0, MaxOut: 32767},
			{MinIn: 0, MaxIn: 32767, MinOut: 0, MaxOut: 32767},
			{MinIn: 0, MaxIn: 32767, MinOut: 0, MaxOut: 32767},
			{MinIn: -32767, MaxIn: 32767, MinOut: -32767, MaxOut: 32767},
		},
	}, ph.RxHandler, ph.SetupHandler, pid.JoystickPidHIDReport)
	LED1.Configure(machine.PinConfig{Mode: machine.PinOutput})
	LED2.Configure(machine.PinConfig{Mode: machine.PinOutput})
	LED3.Configure(machine.PinConfig{Mode: machine.PinOutput})
	LED1.High()
	LED2.High()
	LED3.High()
	SW1.Configure(machine.PinConfig{Mode: machine.PinInput})
	SW2.Configure(machine.PinConfig{Mode: machine.PinInput})
	SW3.Configure(machine.PinConfig{Mode: machine.PinInput})
	CAN_INT.Configure(machine.PinConfig{Mode: machine.PinInput})
	CAN_RESET.Configure(machine.PinConfig{Mode: machine.PinOutput})
	CAN_RESET.Low()
	time.Sleep(10 * time.Millisecond)
	CAN_RESET.High()
	time.Sleep(10 * time.Millisecond)
}

var (
	fitx   = utils.Map(-32767, 32767, 0, 4)
	limitx = utils.Limit(0, 3)
	fity   = utils.Map(-32767, 32767, 0, 3)
	limity = utils.Limit(0, 2)
	prev   = 0
)

func main() {
	LED1.Low()
	log.SetFlags(log.Lmicroseconds)
	if err := spi.Configure(
		machine.SPIConfig{
			Frequency: 500000,
			SCK:       CAN_SCK,
			SDO:       CAN_TX,
			SDI:       CAN_RX,
			Mode:      0,
		},
	); err != nil {
		log.Print(err)
	}
	can := mcp2515.New(spi, CAN_CS)
	can.Configure()
	if err := can.Begin(mcp2515.CAN500kBps, mcp2515.Clock8MHz); err != nil {
		log.Fatal(err)
	}
	if err := motor.Setup(can); err != nil {
		log.Fatal(err)
	}
	motor.SetNeutralAdjust(NeutralAdjust)
	ticker := time.NewTicker(1 * time.Millisecond)
	fit := utils.Map(-MaxAngle, MaxAngle, -32767, 32767)
	limit1 := utils.Limit(-32767, 32767)
	limit2 := utils.Limit(-500, 500)
	cnt := 0
	for range ticker.C {
		state, err := motor.GetState(can)
		if err != nil {
			log.Print(err)
		}
		angle := fit(state.Angle)
		output := limit2(-angle) + int32(state.Verocity)*CoggingTorqueCancel
		force := ph.CalcForces()
		switch {
		case angle > 32767:
			output -= CenteringForce * (angle - 32767)
		case angle < -32767:
			output -= CenteringForce * (angle + 32767)
		}
		output -= force[0]
		cnt++
		// for slow start
		if cnt < 300 {
			output = output * int32(cnt) / 300
		}
		if err := motor.Output(can, int16(limit1(output))); err != nil {
			log.Print(err)
		}
		js.SetAxis(0, int(limit1(angle)))
		if cnt%10 == 0 {
			js.SendState()
		}
	}
}
