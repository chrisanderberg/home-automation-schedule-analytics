package storage

type ControlType string

const (
	// ControlTypeDiscrete represents enumerated/radio-style controls.
	ControlTypeDiscrete ControlType = "discrete"
	// ControlTypeSlider represents discretized slider controls.
	ControlTypeSlider ControlType = "slider"
)

// Control stores metadata required to validate and index aggregate updates.
type Control struct {
	ControlID   string
	ControlType ControlType
	NumStates   int
	StateLabels []string
}

// AggregateKey identifies one dense aggregate payload row.
type AggregateKey struct {
	ControlID    string
	ModelID      string
	QuarterIndex int
}
