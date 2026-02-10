package blob

const (
	BucketsPerDay  = 288
	BucketsPerWeek = 7 * BucketsPerDay
	Clocks         = 5
	GroupSize      = BucketsPerWeek * Clocks
)

const (
	MinStates = 2
	MaxStates = 10
)
