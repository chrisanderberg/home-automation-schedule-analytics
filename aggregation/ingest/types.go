package ingest

// Config holds service-wide clock configuration shared by all ingestion paths.
type Config struct {
	TimeZone  string  `json:"timeZone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// HoldingInput represents one half-open [start,end) holding interval in UTC ms.
type HoldingInput struct {
	ControlID   string `json:"controlId"`
	ModelID     string `json:"modelId"`
	State       int    `json:"state"`
	StartTimeMs int64  `json:"startTimeMs"`
	EndTimeMs   int64  `json:"endTimeMs"`
}
