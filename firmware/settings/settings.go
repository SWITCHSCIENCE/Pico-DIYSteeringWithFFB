package settings

import "fmt"

type Settings struct {
	NeutralAdjust          float32 // unit:deg
	Lock2Lock              int32   // unit:deg
	CoggingTorqueCancel    int32   // 32768 // unit:100*n/256 %
	Viscosity              int32   // 30000 // unit:100*n/256 %
	MaxCenteringForce      int32   // unit:100*n/32767 %
	SoftLockForceMagnitude int32   // unit:100*n %
}

var (
	defaultSettings = Settings{
		NeutralAdjust:          -6.5, // unit:deg
		Lock2Lock:              540,  // unit:deg
		CoggingTorqueCancel:    128,  // unit:100*n/256 %
		Viscosity:              128,  // unit:100*n/256 %
		MaxCenteringForce:      500,  // unit:100*n/32767 %
		SoftLockForceMagnitude: 8,    // unit:100*n %
	}
	currentSettings = defaultSettings
	subscribe       []func(s Settings) error
)

func Validate(s Settings) error {
	if s.NeutralAdjust < -180 || s.NeutralAdjust > +180 {
		return fmt.Errorf("invalid neutral adjust: %f", s.NeutralAdjust)
	}
	if s.Lock2Lock < 180 || s.Lock2Lock > 1440 {
		return fmt.Errorf("invalid lock to lock: %d", s.Lock2Lock)
	}
	if s.CoggingTorqueCancel < 0 || s.CoggingTorqueCancel > 256 {
		return fmt.Errorf("invalid cogging torque cancel: %d", s.CoggingTorqueCancel)
	}
	if s.Viscosity < 0 || s.Viscosity > 1024 {
		return fmt.Errorf("invalid viscosity: %d", s.Viscosity)
	}
	if s.MaxCenteringForce < 0 || s.MaxCenteringForce > 2048 {
		return fmt.Errorf("invalid max centering force: %d", s.MaxCenteringForce)
	}
	if s.SoftLockForceMagnitude < 0 || s.SoftLockForceMagnitude > 16 {
		return fmt.Errorf("invalid soft lock force magnitude: %d", s.SoftLockForceMagnitude)
	}
	return nil
}

func SubscribeClear() {
	subscribe = nil
}

func SubscribeAdd(f func(s Settings) error) {
	subscribe = append(subscribe, f)
}

func Restore() error {
	s := defaultSettings
	// TODO: implement to read from flash memory
	if err := Update(s); err != nil {
		currentSettings = defaultSettings
		Update(currentSettings)
		return err
	}
	return nil
}

func Save(s Settings) error {
	if err := Validate(s); err != nil {
		return err
	}
	// TODO: implement to write to flash memory
	return nil
}

func Update(s Settings) error {
	if err := Validate(s); err != nil {
		return err
	}
	// notify all subscribers
	for _, l := range subscribe {
		if err := l(s); err != nil {
			return err
		}
	}
	currentSettings = s
	return nil
}

func Get() Settings {
	return currentSettings
}
