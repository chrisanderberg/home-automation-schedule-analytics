package blob

const (
	// ClockUTC is wall-clock UTC week bucketing.
	ClockUTC = iota
	// ClockLocal is wall-clock local-time week bucketing.
	ClockLocal
	// ClockMeanSolar is longitude-adjusted mean solar time bucketing.
	ClockMeanSolar
	// ClockApparentSolar is mean solar plus equation-of-time correction.
	ClockApparentSolar
	// ClockUnequalHours maps day/night into unequal-hour bucket space.
	ClockUnequalHours
	clockCount
)

const (
	// BucketsPerDay is fixed at 5-minute resolution.
	BucketsPerDay = 288
	// BucketsPerWeek follows Monday..Sunday indexing across all days.
	BucketsPerWeek = 7 * BucketsPerDay
	// Clocks is the fixed number of parallel time coordinate systems.
	Clocks = clockCount
	// GroupSize is one full week of buckets across all clocks.
	GroupSize = BucketsPerWeek * Clocks
)

const (
	// MinStates/MaxStates are the allowed discrete control cardinalities.
	MinStates = 2
	MaxStates = 10
)
