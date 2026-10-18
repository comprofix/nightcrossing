package config

// Mixed traffic (DESIGN §5.8, D65): from level 4 of every mode but Stroll,
// road lanes carry a mix of vehicles. Each lane keeps one speed, so nothing
// overtakes. Every mixed lane carries cars, bikes and vans; the big trucks
// stay on row 8, the slow heavy lane by the track. The patterns are fixed
// and repeat, so a lane can still be learned. Each keeps its Stroll lane's
// speed and about its share of road covered. Starting values, tuned by
// play.
var (
	mixedHeavy = Lane{Kind: Road, Dir: -1, Speed: 10, Pattern: []Spawn{
		{Entity: Truck, Gap: 150}, {Entity: Van, Gap: 90}, {Entity: CarRed, Gap: 110},
		{Entity: Truck, Gap: 230}, {Entity: Bike, Gap: 120}, {Entity: CarBlue, Gap: 160},
	}}
	mixedCars = Lane{Kind: Road, Dir: 1, Speed: 16, Pattern: []Spawn{
		{Entity: CarRed, Gap: 90}, {Entity: Van, Gap: 120}, {Entity: CarBlue, Gap: 90},
		{Entity: Bike, Gap: 150}, {Entity: CarYellow, Gap: 170},
	}}
	mixedFast = Lane{Kind: Road, Dir: -1, Speed: 28, Pattern: []Spawn{
		{Entity: Bike, Gap: 220}, {Entity: CarYellow, Gap: 260}, {Entity: Van, Gap: 280}, {Entity: Bike, Gap: 240},
	}}
	mixedLocal = Lane{Kind: Road, Dir: 1, Speed: 12, Pattern: []Spawn{
		{Entity: Van, Gap: 110}, {Entity: CarRed, Gap: 80}, {Entity: Van, Gap: 160},
		{Entity: CarBlue, Gap: 100}, {Entity: Bike, Gap: 150},
	}}
)

// The three mixed roads, one more lane mixed in each: the heavy lane on
// level 4, the local lane too on level 6, and all four on level 8, which
// the loops keep (D65).
var (
	mixedRoad1 = withLanes(strollRoad, map[int]Lane{8: mixedHeavy})
	mixedRoad2 = withLanes(strollRoad, map[int]Lane{8: mixedHeavy, 11: mixedLocal})
	mixedRoad3 = withLanes(strollRoad, map[int]Lane{8: mixedHeavy, 9: mixedCars, 10: mixedFast, 11: mixedLocal})
)

// MixedFrom is the first level with mixed traffic, and MixedHint its hint
// (D65).
const (
	MixedFrom = 4
	MixedHint = "MIXED TRAFFIC: MIND THE TRUCKS"
)

// withLanes is road with the given rows replaced.
func withLanes(road [Rows]Lane, rows map[int]Lane) *[Rows]Lane {
	for r, l := range rows {
		road[r] = l
	}
	return &road
}

// withMixedTraffic gives levels from MixedFrom on their mixed roads (D65).
func withMixedTraffic(levels []LevelDef) {
	for i := MixedFrom - 1; i < len(levels); i++ {
		switch n := i + 1; {
		case n >= 8:
			levels[i].Road = mixedRoad3
		case n >= 6:
			levels[i].Road = mixedRoad2
		default:
			levels[i].Road = mixedRoad1
		}
	}
}

// The fair-gap rules for mixed traffic, part of the crossing check (D21,
// D65): the gap behind a truck is at least TruckGapMin pixels once the gap
// % is applied, and no vehicle covers the hero's cell for longer than
// CoverMaxTicks: as long as a truck in Stroll's level 1 heavy lane does.
const TruckGapMin = 2 * Cell

// CoverMaxTicks is the longest a vehicle may cover the hero's cell (D65).
var CoverMaxTicks = CoverTicks(Truck.Length, strollRoad[8].Speed)

// CoverTicks is how long a vehicle length pixels long, in a lane moving at
// speed (SubPx units per tick), covers the hero's cell, with the vehicle
// trimmed by HitMargin and the hero by HeroMargin: the other side of
// GapOpenTicks.
func CoverTicks(length, speed int) int {
	covered := (length - 2*HitMargin) + (Cell - 2*HeroMargin)
	return (covered*SubPx + speed - 1) / speed
}
