package ingest

// Config holds service-wide clock configuration shared by all ingestion paths.
type Config struct {
	TimeZone  string
	Latitude  float64
	Longitude float64
}

// HoldingInput represents one half-open [start,end) holding interval in UTC ms.
type HoldingInput struct {
	ControlID   string
	ModelID     string
	State       int
	StartTimeMs int64
	EndTimeMs   int64
}
