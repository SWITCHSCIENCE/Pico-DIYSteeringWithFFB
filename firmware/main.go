package main

import (
	"context"
	"log"
	"machine"
	"time"

	"tinygo.org/x/drivers/mcp2515"

	"github.com/SWITCHSCIENCE/ffb_steering_controller/control"
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
)

var (
	spi = machine.SPI0
)

func init() {
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
	js := control.NewWheel(can)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		if err := js.Loop(ctx); err != nil {
			log.Print(err)
			time.Sleep(3 * time.Second)
		}
	}
}
