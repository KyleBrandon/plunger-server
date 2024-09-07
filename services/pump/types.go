package pump

type PumpSensor interface {
	IsPumpOn() (bool, error)
	TurnPumpOn() error
	TurnPumpOff() error
}

type Handler struct {
	pump PumpSensor
}
