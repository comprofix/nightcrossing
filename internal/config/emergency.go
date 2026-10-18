package config

// Emergency vehicles (DESIGN §6.9, D71–D74): in Rush Hour, and carried on
// into Storm and School Run, a police car, an ambulance or a fire truck
// comes through one of the road's middle lanes. At its warning the lane's
// cars drive off and none come on, while the lanes either side bunch up
// and stop to let it through, trapping a toad still in the lane. Starting
// values, tuned by play.

// The levels each vehicle comes from (D73): the police car from
// EmergencyFrom, the ambulance too from AmbulanceFrom, and the fire truck
// too from FireTruckFrom.
const (
	EmergencyFrom = 3
	AmbulanceFrom = 4
	FireTruckFrom = 5
)

// EmergencyRows are the road rows an emergency vehicle can come through:
// the two middle lanes, never the one by the track or the one by the
// start row (D72).
var EmergencyRows = [...]int{9, 10}

// Responder is an emergency vehicle: its look and length, and how fast it
// comes through, in SubPx units per tick (D73).
type Responder struct {
	Entity
	Speed int
}

// The vehicles (D73).
var (
	Police    = Responder{Entity: Entity{ID: "police", Length: 2 * Cell}, Speed: 10 * SubPx}
	Ambulance = Responder{Entity: Entity{ID: "ambulance", Length: 3 * Cell}, Speed: 9 * SubPx}
	FireTruck = Responder{Entity: Entity{ID: "fire_truck", Length: 4 * Cell}, Speed: 8 * SubPx}
)

// Responders lists the vehicles, in the order they come in.
func Responders() []Responder { return []Responder{Police, Ambulance, FireTruck} }

// RespondersOn is how many of Responders come on level n: the first that
// many of them (D73).
func RespondersOn(n int) int {
	switch {
	case n >= FireTruckFrom:
		return 3
	case n >= AmbulanceFrom:
		return 2
	case n >= EmergencyFrom:
		return 1
	}
	return 0
}

// The trap (D72). At the warning the emergency lane's cars speed up to at
// least ClearPct of their speed, faster if they need to, to be off the
// field within CloseTicks. The lanes either side bunch up QueueGap pixels
// nose to tail, the cars closing at the speed that fills the field in
// about CloseTicks, with extra cars coming on from ExtrasTicks after the
// warning; never sooner than EscapeTicks: the time a toad in the
// emergency lane has to get out, which the crossing check holds it to.
// The vehicle comes once the lane is clear and the trap closed, or at the
// latest RunWaitTicks after the lane is clear. Once it has gone, the
// queues' brake lights go off after ReleaseTicks, and they move off
// MoveOffTicks after that; the emergency lane's traffic comes back
// ReturnTicks after they do.
const (
	ClearPct     = 150
	CloseTicks   = 3 * TPS
	ExtrasTicks  = TPS
	EscapeTicks  = 2 * TPS
	QueueGap     = 4
	RunWaitTicks = 3 * TPS
	ReleaseTicks = TPS
	MoveOffTicks = TPS / 2
	ReturnTicks  = TPS
)

// EmergencyMessage is the hint strip's warning (D74).
const EmergencyMessage = "EMERGENCY VEHICLE: CLEAR THE LANE"
