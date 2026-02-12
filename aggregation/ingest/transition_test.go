package ingest

import (
	"context"
	"testing"
	"time"

	"home-automation-analytics/aggregation/blob"
	"home-automation-analytics/aggregation/bucketing"
	"home-automation-analytics/aggregation/quarter"
	"home-automation-analytics/aggregation/storage"
)

// TestTransitionIngestSingleBucketUTC verifies transition ingestion increments
// the correct directed edge in both UTC and Local clocks and leaves other
// transition cells unchanged.
func TestTransitionIngestSingleBucketUTC(t *testing.T) {
	ctx := context.Background()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := storage.InitSchema(ctx, db); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	control := storage.Control{ControlID: "c1", ControlType: storage.ControlTypeDiscrete, NumStates: 3}
	if err := storage.UpsertControl(ctx, db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	cfg := Config{TimeZone: "UTC", Latitude: 0, Longitude: 0}

	timestamp := time.Date(2020, 1, 6, 0, 7, 0, 0, time.UTC)
	input := TransitionInput{
		ControlID:   "c1",
		ModelID:     "m1",
		FromState:   0,
		ToState:     2,
		TimestampMs: timestamp.UnixMilli(),
	}

	if err := IngestTransition(ctx, db, cfg, input); err != nil {
		t.Fatalf("ingest transition: %v", err)
	}

	key := storage.AggregateKey{
		ControlID:    "c1",
		ModelID:      "m1",
		QuarterIndex: quarter.QuarterIndexUTC(timestamp.UnixMilli()),
	}
	blobBytes, err := storage.GetOrCreateAggregate(ctx, db, key, 3)
	if err != nil {
		t.Fatalf("get aggregate: %v", err)
	}
	b, err := blob.NewBlob(3)
	if err != nil {
		t.Fatalf("new blob: %v", err)
	}
	copy(b.Data(), blobBytes)

	bucketUTC, err := bucketing.BucketAtUTC(timestamp.UnixMilli())
	if err != nil {
		t.Fatalf("bucketAtUTC: %v", err)
	}
	idxUTC, err := blob.TransIndex(0, 2, blob.ClockUTC, bucketUTC, 3)
	if err != nil {
		t.Fatalf("trans index utc: %v", err)
	}
	bucketLocal, err := bucketing.BucketAtLocal(timestamp.UnixMilli(), time.UTC)
	if err != nil {
		t.Fatalf("bucketAtLocal: %v", err)
	}
	idxLocal, err := blob.TransIndex(0, 2, blob.ClockLocal, bucketLocal, 3)
	if err != nil {
		t.Fatalf("trans index local: %v", err)
	}

	vUTC, err := b.GetU64(idxUTC)
	if err != nil {
		t.Fatalf("get utc value: %v", err)
	}
	vLocal, err := b.GetU64(idxLocal)
	if err != nil {
		t.Fatalf("get local value: %v", err)
	}
	if vUTC != 1 {
		t.Fatalf("utc transition mismatch: got %d want %d", vUTC, 1)
	}
	if vLocal != 1 {
		t.Fatalf("local transition mismatch: got %d want %d", vLocal, 1)
	}

	idxOther, err := blob.TransIndex(1, 2, 0, bucketUTC, 3)
	if err != nil {
		t.Fatalf("trans index other: %v", err)
	}
	vOther, err := b.GetU64(idxOther)
	if err != nil {
		t.Fatalf("get other value: %v", err)
	}
	if vOther != 0 {
		t.Fatalf("unexpected other transition value: %d", vOther)
	}
}
