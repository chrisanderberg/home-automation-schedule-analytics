package storage

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"testing"

	"home-automation-analytics/aggregation/blob"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := InitSchema(context.Background(), db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return db
}

// TestControlCRUD verifies control metadata can be written and read back
// without mutation through the storage layer.
func TestControlCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	control := Control{ControlID: "c1", ControlType: ControlTypeDiscrete, NumStates: 3}
	if err := UpsertControl(context.Background(), db, control); err != nil {
		t.Fatalf("upsert control: %v", err)
	}

	got, err := GetControl(context.Background(), db, "c1")
	if err != nil {
		t.Fatalf("get control: %v", err)
	}
	if got.ControlID != control.ControlID || got.ControlType != control.ControlType || got.NumStates != control.NumStates {
		t.Fatalf("control mismatch: got %+v want %+v", got, control)
	}
}

// TestAggregateCreateUpdate verifies aggregate rows are lazily created with
// canonical blob size and that updates persist modified blob values.
func TestAggregateCreateUpdate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	key := AggregateKey{ControlID: "c1", ModelID: "m1", QuarterIndex: 1}
	blobBytes, err := GetOrCreateAggregate(context.Background(), db, key, 2)
	if err != nil {
		t.Fatalf("get or create aggregate: %v", err)
	}
	if len(blobBytes) != 2*2*blob.GroupSize*8 {
		t.Fatalf("blob size mismatch: %d", len(blobBytes))
	}

	if err := UpdateAggregate(context.Background(), db, key, 2, func(data []byte) error {
		b, err := blob.NewBlob(2)
		if err != nil {
			return err
		}
		copy(b.Data(), data)
		if err := b.SetU64(0, 7); err != nil {
			return err
		}
		copy(data, b.Data())
		return nil
	}); err != nil {
		t.Fatalf("update aggregate: %v", err)
	}

	updated, err := GetOrCreateAggregate(context.Background(), db, key, 2)
	if err != nil {
		t.Fatalf("get aggregate after update: %v", err)
	}
	b, err := blob.NewBlob(2)
	if err != nil {
		t.Fatalf("new blob: %v", err)
	}
	copy(b.Data(), updated)
	v, err := b.GetU64(0)
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	if v != 7 {
		t.Fatalf("updated value mismatch: got %d want %d", v, 7)
	}
}

// TestAggregateConcurrentUpdates exercises concurrent aggregate updates and
// verifies serialized update behavior prevents lost writes.
func TestAggregateConcurrentUpdates(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	key := AggregateKey{ControlID: "c1", ModelID: "m1", QuarterIndex: 1}
	_, err := GetOrCreateAggregate(context.Background(), db, key, 2)
	if err != nil {
		t.Fatalf("get or create aggregate: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	updateFn := func(delta uint64) {
		defer wg.Done()
		err := UpdateAggregate(context.Background(), db, key, 2, func(data []byte) error {
			b, err := blob.NewBlob(2)
			if err != nil {
				return err
			}
			copy(b.Data(), data)
			v, err := b.GetU64(0)
			if err != nil {
				return err
			}
			if err := b.SetU64(0, v+delta); err != nil {
				return err
			}
			copy(data, b.Data())
			return nil
		})
		if err != nil {
			t.Errorf("update aggregate: %v", err)
		}
	}

	go updateFn(3)
	go updateFn(4)
	wg.Wait()

	updated, err := GetOrCreateAggregate(context.Background(), db, key, 2)
	if err != nil {
		t.Fatalf("get aggregate after updates: %v", err)
	}
	b, err := blob.NewBlob(2)
	if err != nil {
		t.Fatalf("new blob: %v", err)
	}
	copy(b.Data(), updated)
	v, err := b.GetU64(0)
	if err != nil {
		t.Fatalf("get value: %v", err)
	}
	if v != 7 {
		t.Fatalf("concurrent update mismatch: got %d want %d", v, 7)
	}
}

// TestUpdateAggregateRejectsMismatchedBlobSize verifies updates fail fast when
// stored blob bytes do not match the expected shape for numStates.
func TestUpdateAggregateRejectsMismatchedBlobSize(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	key := AggregateKey{ControlID: "c1", ModelID: "m1", QuarterIndex: 1}
	_, err := db.ExecContext(
		context.Background(),
		`INSERT INTO aggregates (control_id, model_id, quarter_index, blob) VALUES (?, ?, ?, ?)`,
		key.ControlID,
		key.ModelID,
		key.QuarterIndex,
		[]byte{1, 2, 3},
	)
	if err != nil {
		t.Fatalf("seed malformed aggregate: %v", err)
	}

	err = UpdateAggregate(context.Background(), db, key, 2, func(data []byte) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected mismatched blob size error")
	}
	if !strings.Contains(err.Error(), "aggregate blob size mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
