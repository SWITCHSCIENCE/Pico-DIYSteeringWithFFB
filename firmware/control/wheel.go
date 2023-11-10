package control

import (
	"context"
	"machine/usb/hid/joystick"
	"time"

	"tinygo.org/x/drivers/mcp2515"

	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/motor"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/pid"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/settings"
	"github.com/SWITCHSCIENCE/Pico-DIYSteeringWithFFB/utils"
)

var (
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
	}, ph.RxHandler, ph.SetupHandler, pid.Descriptor)
)

type Joystick interface {
	SetHat(index int, dir joystick.HatDirection)
	SetButton(index int, push bool)
	SetAxis(index int, v int)
	SendState()
}

type Wheel struct {
	Joystick
	calc func() []int32
	can  *mcp2515.Device
}

func NewWheel(can *mcp2515.Device) *Wheel {
	w := &Wheel{
		Joystick: js,
		calc:     ph.CalcForces,
		can:      can,
	}
	return w
}

func (w *Wheel) Loop(ctx context.Context) error {
	if err := motor.Setup(w.can); err != nil {
		return err
	}
	CoggingTorqueCancel := int32(0)
	Viscosity := int32(0)
	SoftLockForceMagnitude := int32(0)
	var fit = func(x int32) int32 { return x }
	var limitForce = func(x int32) int32 { return x }
	settings.SubscribeClear()
	settings.SubscribeAdd(func(s settings.Settings) error {
		CoggingTorqueCancel = s.CoggingTorqueCancel
		Viscosity = s.Viscosity
		SoftLockForceMagnitude = s.SoftLockForceMagnitude
		HalfLock2Lock := s.Lock2Lock / 2
		MaxAngle := 32768*HalfLock2Lock/360 - 1
		fit = utils.Map(-MaxAngle, MaxAngle, -32767, 32767)
		limitForce = utils.Limit(-s.MaxCenteringForce, s.MaxCenteringForce)
		motor.SetNeutralAdjust(s.NeutralAdjust)
		return nil
	})
	if err := settings.Restore(); err != nil {
		return err
	}
	limit1 := utils.Limit(-32767, 32767)
	cnt := 0
	tick := time.NewTicker(1 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
			state, err := motor.GetState(w.can)
			if err != nil {
				return err
			}
			verocity := 256 * int32(state.Verocity) / 220
			angle := fit(state.Angle)
			output := limitForce(-angle)          // Centering
			cog := CoggingTorqueCancel * verocity // Cogging Torque Cancel
			decel := -Viscosity * pow3(verocity)  // Viscosity
			output += int32(cog + decel)          // Sum
			force := w.calc()
			switch {
			case angle > 32767:
				output -= SoftLockForceMagnitude * (angle - 32767)
			case angle < -32767:
				output -= SoftLockForceMagnitude * (angle + 32767)
			}
			output -= force[0]
			cnt++
			if cnt < 300 {
				output = output * int32(cnt) / 300
			}
			if err := motor.Output(w.can, int16(limit1(output))); err != nil {
				return err
			}
			limitAngle := int(limit1(angle))
			w.SetAxis(0, limitAngle)
			w.SetAxis(5, limitAngle)
			if cnt%10 == 0 {
				w.SendState()
			}
		}
	}
}
