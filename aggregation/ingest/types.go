package ingest

type Config struct {
	TimeZone  string
	Latitude  float64
	Longitude float64
}

type HoldingInput struct {
	ControlID   string
	ModelID     string
	State       int
	StartTimeMs int64
	EndTimeMs   int64
}
