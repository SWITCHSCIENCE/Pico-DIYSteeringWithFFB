package pid

import (
	"fmt"
	"machine"
	"machine/usb"
	"machine/usb/hid"
	"time"
)

type PIDHandler struct {
	effectStates []*TEffectState
	pidBlockLoad PIDBlockLoadFeatureData
	pidPool      PIDPoolFeatureData
	gains        Gains
	params       EffectParams
	nextEID      uint8
	enabled      bool
	paused       bool
	gain         uint8
}

func NewPIDHandler() *PIDHandler {
	effects := make([]*TEffectState, MAX_EFFECTS)
	for i := range effects[:] {
		effects[i] = &TEffectState{}
	}
	return &PIDHandler{
		effectStates: effects,
		pidBlockLoad: PIDBlockLoadFeatureData{b: make([]byte, 5)},
		pidPool:      PIDPoolFeatureData{b: make([]byte, 5)},
		gains: Gains{
			TotalGain:    255,
			ConstantGain: 255,
		},
		params: EffectParams{},
	}
}

func (m *PIDHandler) SetGains(gains Gains) {
	m.gains = gains
}

func (m *PIDHandler) SetEffectParams(params EffectParams) {
	m.params = params
}

// from InterruptOut
func (m *PIDHandler) RxHandler(b []byte) {
	if len(b) == 0 {
		return
	}
	reportId := ReportID(b[0])
	switch reportId {
	case ReportSetEffect: // 0x01
		m.SetEffect(b)
	case ReportSetEnvelope: // 0x02
		m.SetEnvelope(b)
	case ReportSetCondition: // 0x03
		m.SetCondition(b)
	case ReportSetPeriodic: // 0x04
		m.SetPeriodic(b)
	case ReportSetConstantForce: // 0x05
		m.SetConstantForce(b)
	case ReportSetRampForce: // 0x06
		m.SetRampForce(b)
	case ReportSetCustomForceData: // 0x07
		m.SetCustomForceData(b)
	case ReportSetDownloadForceSample: // 0x08
		m.SetDownloadForceSample(b)
	case ReportEffectOperation: // 0x0a
		m.EffectOperation(b)
	case ReportBlockFree: // 0x0b
		m.BlockFree(b)
	case ReportDeviceControl: // 0x0c
		m.DeviceControl(b)
	case ReportDeviceGain: // 0x0d
		m.DeviceGain(b)
	case ReportSetCustomForce: // 0x0e
		m.SetCustomForce(b)
	}
}

func (m *PIDHandler) CreateNewEffect(data *CreateNewEffectFeatureData) error {
	m.pidBlockLoad.ReportID = 6
	m.pidBlockLoad.EffectBlockIndex = m.GetNextFreeEffect()
	if m.pidBlockLoad.EffectBlockIndex == 0 {
		m.pidBlockLoad.LoadStatus = 2 // 1=Success,2=Full,3=Error
		return fmt.Errorf("effect not allocated")
	}
	m.pidBlockLoad.LoadStatus = 1 // 1=Success,2=Full,3=Error
	effect := TEffectState{}
	effect.State = MEFFECTSTATE_ALLOCATED
	m.pidBlockLoad.RamPoolAvailable -= SIZE_EFFECT
	*m.effectStates[m.pidBlockLoad.EffectBlockIndex] = effect
	return nil
}

func (m *PIDHandler) GetReport(setup usb.Setup) bool {
	reportId := setup.WValueL
	switch setup.WValueH {
	case hid.REPORT_TYPE_INPUT:
	case hid.REPORT_TYPE_OUTPUT:
	case hid.REPORT_TYPE_FEATURE:
		switch reportId {
		case 6:
			b, _ := m.pidBlockLoad.MarshalBinary()
			machine.SendUSBInPacket(0, b)
			return true
		case 7:
			b, _ := PIDPoolFeatureData{
				ReportID:               7,
				RamPoolSize:            MEMORY_SIZE,
				MaxSimultaneousEffects: MAX_EFFECTS,
				MemoryManagement:       3,
			}.MarshalBinary()
			machine.SendUSBInPacket(0, b)
			return true
		}
	}
	return false
}

func (m *PIDHandler) GetIdle(setup usb.Setup) bool {
	machine.SendZlp()
	return true
}

func (m *PIDHandler) GetProtocol(setup usb.Setup) bool {
	machine.SendZlp()
	return true
}

func (m *PIDHandler) SetReport(setup usb.Setup) bool {
	reportId := setup.WValueL
	switch setup.WValueH {
	case hid.REPORT_TYPE_INPUT:
		machine.SendZlp()
		return true
	case hid.REPORT_TYPE_OUTPUT:
		machine.SendZlp()
		return true
	case hid.REPORT_TYPE_FEATURE:
		if setup.WLength == 0 {
			machine.ReceiveUSBControlPacket()
			machine.SendZlp()
			return true
		}
		if reportId == 5 {
			b, err := machine.ReceiveUSBControlPacket()
			if err != nil {
				return false
			}
			v := &CreateNewEffectFeatureData{}
			v.UnmarshalBinary(b[:])
			if err := m.CreateNewEffect(v); err != nil {
				return false
			}
			machine.SendZlp()
			return true
		}
	}
	return false
}

func (m *PIDHandler) SetIdle(setup usb.Setup) bool {
	machine.SendZlp()
	return true
}

func (m *PIDHandler) SetProtocol(setup usb.Setup) bool {
	machine.SendZlp()
	return true
}

func (m *PIDHandler) SetupHandler(setup usb.Setup) bool {
	switch setup.BmRequestType {
	case usb.REQUEST_DEVICETOHOST_CLASS_INTERFACE:
		switch setup.BRequest {
		case usb.GET_REPORT:
			return m.GetReport(setup)
		case usb.GET_IDLE:
			return m.GetIdle(setup)
		case usb.GET_PROTOCOL:
			return m.GetProtocol(setup)
		}
	case usb.REQUEST_HOSTTODEVICE_CLASS_INTERFACE:
		switch setup.BRequest {
		case usb.SET_REPORT:
			return m.SetReport(setup)
		case usb.SET_IDLE:
			return m.SetIdle(setup)
		case usb.SET_PROTOCOL:
			return m.SetProtocol(setup)
		}
	}
	return false
}

func (m *PIDHandler) GetNextFreeEffect() uint8 {
	if m.nextEID == MAX_EFFECTS {
		return 0
	}
	id := m.nextEID
	m.nextEID++

	for m.effectStates[m.nextEID].State != MEFFECTSTATE_FREE {
		if m.nextEID >= MAX_EFFECTS {
			break
		}
		m.nextEID++
	}
	effect := m.effectStates[id]
	effect.State = MEFFECTSTATE_ALLOCATED
	return id
}

func (m *PIDHandler) StopAllEffects() {
	for id := uint8(0); id < MAX_EFFECTS; id++ {
		m.StopEffect(id)
	}
}

func (m *PIDHandler) StartEffect(id uint8) {
	if id >= MAX_EFFECTS {
		// unknown id
		return
	}
	effect := m.effectStates[id]
	effect.State = MEFFECTSTATE_PLAYING
	effect.ElapsedTime = 0
	effect.StartTime = uint64(time.Now().UnixMilli())
}

func (m *PIDHandler) StopEffect(id uint8) {
	if id >= MAX_EFFECTS {
		// unknown id
		return
	}
	effect := m.effectStates[id]
	effect.State &= ^MEFFECTSTATE_PLAYING
	m.pidBlockLoad.RamPoolAvailable += SIZE_EFFECT
}

func (m *PIDHandler) FreeAllEffects() {
	m.nextEID = 1
	for id := uint8(0); id < MAX_EFFECTS; id++ {
		m.effectStates[id].Clear()
	}
	m.pidBlockLoad.RamPoolAvailable = MEMORY_SIZE
}

func (m *PIDHandler) FreeEffect(id uint8) {
	if id >= MAX_EFFECTS {
		// unknown id
		return
	}
	state := m.effectStates[id]
	state.State = MEFFECTSTATE_FREE
	if id < m.nextEID {
		m.nextEID = id
	}
}

// SetEffect reportId == 0x01
func (m *PIDHandler) SetEffect(b []byte) {
	var v SetEffectOutputData
	_ = v.UnmarshalBinary(b)
	effect := m.effectStates[v.EffectBlockIndex]
	effect.Duration = v.Duration
	effect.DirectionX = v.DirectionX
	effect.DirectionY = v.DirectionY
	effect.EffectType = v.EffectType
	effect.Gain = v.Gain
	effect.EnableAxis = v.EnableAxis
}

// SetEnvelope reportId == 0x02
func (m *PIDHandler) SetEnvelope(b []byte) {
	var v SetEnvelopeOutputData
	_ = v.UnmarshalBinary(b)
	effect := m.effectStates[v.EffectBlockIndex]
	effect.AttackLevel = int16(v.AttackLevel)
	effect.FadeLevel = v.FadeLevel
	effect.AttackTime = uint16(v.AttackTime)
	effect.FadeTime = uint16(v.FadeTime)
}

// SetCondition reportId == 0x03
func (m *PIDHandler) SetCondition(b []byte) {
	var v SetConditionOutputData
	_ = v.UnmarshalBinary(b)
	axis := v.ParameterBlockOffset
	effect := m.effectStates[v.EffectBlockIndex]
	condition := effect.Conditions[axis]
	condition.CpOffset = v.CpOffset
	condition.PositiveCoefficient = v.PositiveCoefficient
	condition.NegativeCoefficient = v.NegativeCoefficient
	condition.PositiveSaturation = v.PositiveSaturation
	condition.NegativeSaturation = v.NegativeSaturation
	condition.DeadBand = v.DeadBand
	effect.Conditions[axis] = condition
	if effect.ConditionBlocksCount < axis {
		effect.ConditionBlocksCount++
	}
}

// SetPeriodic reportId == 0x04
func (m *PIDHandler) SetPeriodic(b []byte) {
	var v SetPeriodicOutputData
	_ = v.UnmarshalBinary(b)
	effect := m.effectStates[v.EffectBlockIndex]
	effect.Magnitude = v.Magnitude
	effect.Offset = v.Offset
	effect.Phase = v.Phase
	effect.Period = uint16(v.Period)
}

// SetConstantForce reportId == 0x05
func (m *PIDHandler) SetConstantForce(b []byte) {
	var v SetConstantForceOutputData
	_ = v.UnmarshalBinary(b)
	effect := m.effectStates[v.EffectBlockIndex]
	effect.Magnitude = v.Magnitude
}

// SetRampForce reportId == 0x06
func (m *PIDHandler) SetRampForce(b []byte) {
	var v SetRampForceOutputData
	_ = v.UnmarshalBinary(b)
	effect := m.effectStates[v.EffectBlockIndex]
	effect.StartMagnitude = v.StartMagnitude
	effect.EndMagnitude = v.EndMagnitude
}

// SetCustomForceData reportId == 0x07
func (m *PIDHandler) SetCustomForceData(b []byte) {
	var v SetCustomForceDataOutputData
	_ = v.UnmarshalBinary(b)
	// TODO: implement
}

// SetDownloadForceSample reportId == 0x08
func (m *PIDHandler) SetDownloadForceSample(b []byte) {
	var v SetDownloadForceSampleOutputData
	_ = v.UnmarshalBinary(b)
	// TODO: implement
}

// EffectOperation reportId == 0x0a
func (m *PIDHandler) EffectOperation(b []byte) {
	var v EffectOperationOutputData
	_ = v.UnmarshalBinary(b)
	switch v.Operation {
	case EOStart:
		effect := m.effectStates[v.EffectBlockIndex]
		switch v.LoopCount {
		case 0xff:
			effect.Duration = USB_DURATION_INFINITE
		default:
			effect.Duration *= uint16(v.LoopCount)
		}
		m.StartEffect(v.EffectBlockIndex)
	case EOStartSolo:
		m.StopAllEffects()
		m.StartEffect(v.EffectBlockIndex)
	case EOStop:
		m.StopEffect(v.EffectBlockIndex)
	}
}

// BlockFree reportId == 0x0b
func (m *PIDHandler) BlockFree(b []byte) {
	var v BlockFreeOutputData
	_ = v.UnmarshalBinary(b)
	if v.EffectBlockIndex == 0xff {
		m.FreeAllEffects()
		return
	}
	m.FreeEffect(v.EffectBlockIndex)
}

// DeviceControl reportId == 0x0c
func (m *PIDHandler) DeviceControl(b []byte) {
	var v DeviceControlOutputData
	_ = v.UnmarshalBinary(b)
	switch v.Control {
	case ControlEnableActuators:
		m.enabled = true
	case ControlDisableActuators:
		m.enabled = false
	case ControlStopAllEffects:
		m.StopAllEffects()
	case ControlReset:
		m.FreeAllEffects()
	case ControlPause:
		m.paused = true
	case ControlContinue:
		m.paused = false
	}
}

// DeviceGain reportId == 0x0d
func (m *PIDHandler) DeviceGain(b []byte) {
	var v DeviceGainOutputData
	_ = v.UnmarshalBinary(b)
	m.gain = v.Gain
}

// SetCustomForce reportId == 0x0e
func (m *PIDHandler) SetCustomForce(b []byte) {
	var v SetCustomForceOutputData
	_ = v.UnmarshalBinary(b)
	// TODO: implement
}

func (m *PIDHandler) CalcForces() []int32 {
	forces := []int32{0, 0}
	for _, ef := range m.effectStates {
		if ef.State == MEFFECTSTATE_PLAYING &&
			(ef.Duration == USB_DURATION_INFINITE ||
				ef.ElapsedTime <= ef.Duration) &&
			!m.paused {
			forces[0] += ef.Force(m.gains, m.params, 0)
			forces[1] += ef.Force(m.gains, m.params, 1)
		}
	}
	return forces
}

func (m *PIDHandler) GetCurrentEffect() *TEffectState {
	return m.effectStates[m.pidBlockLoad.EffectBlockIndex]
}
