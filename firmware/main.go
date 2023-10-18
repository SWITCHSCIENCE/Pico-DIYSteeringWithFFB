package main

import (
	"log"
	"machine"
	"machine/usb/joystick"
	"time"

	"tinygo.org/x/drivers/mcp2515"

	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/motor"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/pid"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/utils"
)

const (
	Lock2Lock     = 540
	HalfLock2Lock = Lock2Lock / 2
	MaxAngle      = 32768*HalfLock2Lock/360 - 1
)

var (
	spi   = machine.SPI0
	csPin = machine.GP28
)

var (
	js *joystick.Joystick
	ph *pid.PIDHandler
)

func init() {
	// USB initialize
	ph = pid.NewPIDHandler()
	// should be matched joystick.Definitions and pid.Descriptor.
	js = joystick.Enable(joystick.Definitions{
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
	}, ph.RxHandler, ph.SetupHandler, pid.Descriptor)
}

func main() {
	log.SetFlags(log.Lmicroseconds)

	// spi initialize
	if err := spi.Configure(
		machine.SPIConfig{
			Frequency: 500000,
			SCK:       machine.GP2,
			SDO:       machine.GP3,
			SDI:       machine.GP4,
			Mode:      0,
		},
	); err != nil {
		log.Print(err)
	}

	// can initialize
	can := mcp2515.New(spi, csPin)
	can.Configure()
	if err := can.Begin(mcp2515.CAN500kBps, mcp2515.Clock8MHz); err != nil {
		log.Fatal(err)
	}

	// motor setup
	if err := motor.Setup(can); err != nil {
		log.Fatal(err)
	}
	// loop for 10 ms cycle
	ticker := time.NewTicker(10 * time.Millisecond)
	fit := utils.Map(-MaxAngle, MaxAngle, -32767, 32767)
	limitInt16 := utils.Limit(-32767, 32767)
	centeringForceLimit := utils.Limit(-500, 500)
	cnt := 0
	for range ticker.C {
		state, err := motor.GetState(can)
		if err != nil {
			log.Print(err)
		}
		angle := fit(state.Angle)
		output := centeringForceLimit(-angle) + int32(state.Verocity)*128
		force := ph.CalcForces()
		switch {
		case angle > 32767:
			output -= 8 * (angle - 32767)
		case angle < -32767:
			output -= 8 * (angle + 32767)
		}
		output -= force[0]
		cnt++
		if cnt < 300 {
			output = output * int32(cnt) / 300
		}
		if err := motor.Output(can, int16(limitInt16(output))); err != nil {
			log.Print(err)
		}
		js.SetButton(2, angle > 32767)
		js.SetButton(3, angle < -32767)
		js.SendState()
	}
}
