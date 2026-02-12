package blob

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidNumStates = errors.New("invalid number of states")
	ErrIndexOutOfRange  = errors.New("index out of range")
	ErrSelfTransition   = errors.New("self transition is not allowed")
)

type Blob struct {
	numStates int
	data      []byte
}

// NewBlob allocates a zeroed dense u64 blob for one (control, model, quarter)
// aggregate payload using the canonical N^2 * GroupSize layout.
func NewBlob(numStates int) (*Blob, error) {
	if numStates < MinStates || numStates > MaxStates {
		return nil, ErrInvalidNumStates
	}
	valueCount := numStates * numStates * GroupSize
	data := make([]byte, valueCount*8)
	return &Blob{numStates: numStates, data: data}, nil
}

// NumStates returns the state cardinality this blob was created for.
func (b *Blob) NumStates() int {
	return b.numStates
}

// Data exposes the raw little-endian bytes backing the blob.
func (b *Blob) Data() []byte {
	return b.data
}

// ValueCount returns the number of u64 values addressable in the blob.
func (b *Blob) ValueCount() int {
	return b.numStates * b.numStates * GroupSize
}

// GetU64 reads one u64 value by canonical value index.
func (b *Blob) GetU64(index int) (uint64, error) {
	if index < 0 || index >= b.ValueCount() {
		return 0, ErrIndexOutOfRange
	}
	offset := index * 8
	return binary.LittleEndian.Uint64(b.data[offset : offset+8]), nil
}

// SetU64 writes one u64 value by canonical value index.
func (b *Blob) SetU64(index int, value uint64) error {
	if index < 0 || index >= b.ValueCount() {
		return ErrIndexOutOfRange
	}
	offset := index * 8
	binary.LittleEndian.PutUint64(b.data[offset:offset+8], value)
	return nil
}

// HoldIndex maps (state, clock, bucket) to the holding region of the dense
// blob layout.
func HoldIndex(state, clock, bucket, numStates int) (int, error) {
	if numStates < MinStates || numStates > MaxStates {
		return 0, ErrInvalidNumStates
	}
	if state < 0 || state >= numStates {
		return 0, ErrIndexOutOfRange
	}
	if clock < 0 || clock >= Clocks {
		return 0, ErrIndexOutOfRange
	}
	if bucket < 0 || bucket >= BucketsPerWeek {
		return 0, ErrIndexOutOfRange
	}
	return (state * GroupSize) + (clock * BucketsPerWeek) + bucket, nil
}

// TransGroupIndex maps a directed non-self pair into its compact transition
// group index where each "from" block omits the diagonal entry.
func TransGroupIndex(fromState, toState, numStates int) (int, error) {
	if numStates < MinStates || numStates > MaxStates {
		return 0, ErrInvalidNumStates
	}
	if fromState < 0 || fromState >= numStates || toState < 0 || toState >= numStates {
		return 0, ErrIndexOutOfRange
	}
	if fromState == toState {
		return 0, ErrSelfTransition
	}
	offset := toState
	if toState > fromState {
		offset = toState - 1
	}
	return fromState*(numStates-1) + offset, nil
}

// TransIndex maps (from, to, clock, bucket) into the transition region that
// begins immediately after all holding groups.
func TransIndex(fromState, toState, clock, bucket, numStates int) (int, error) {
	if clock < 0 || clock >= Clocks {
		return 0, ErrIndexOutOfRange
	}
	if bucket < 0 || bucket >= BucketsPerWeek {
		return 0, ErrIndexOutOfRange
	}
	groupIndex, err := TransGroupIndex(fromState, toState, numStates)
	if err != nil {
		return 0, err
	}
	return (numStates * GroupSize) + (groupIndex * GroupSize) + (clock * BucketsPerWeek) + bucket, nil
}
