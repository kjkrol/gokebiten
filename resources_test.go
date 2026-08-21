package gokebiten

import (
	"bytes"
	"testing"

	"github.com/kjkrol/gokebiten/spatial"
)

type testMarshaledState struct{ N int }

func (s testMarshaledState) MarshalBinary() ([]byte, error) { return []byte{byte(s.N)}, nil }
func (s *testMarshaledState) UnmarshalBinary(data []byte) error {
	s.N = int(data[0])
	return nil
}

type testPlainState struct{ N int }
type testTelemetry struct{}

func TestResources_SaveLoadState_RoundTrip(t *testing.T) {
	r := NewResources(&GameProps{}, spatial.Config{}, testMarshaledState{N: 7}, testTelemetry{})

	var buf bytes.Buffer
	if err := r.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	r2 := NewResources(&GameProps{}, spatial.Config{}, testMarshaledState{}, testTelemetry{})
	if err := r2.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if r2.State().N != 7 {
		t.Errorf("State().N = %d, want 7", r2.State().N)
	}
}

func TestResources_SaveLoadState_ZeroLengthWithoutBinaryMarshaler(t *testing.T) {
	r := NewResources(&GameProps{}, spatial.Config{}, testPlainState{N: 7}, testTelemetry{})

	var buf bytes.Buffer
	if err := r.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	// Only the 4-byte zero length prefix — no state payload — but a prefix
	// is always written, even here, so LoadState can share a stream with
	// data written after it (see Game.Save, which appends an ECS snapshot).
	if buf.Len() != 4 {
		t.Errorf("expected only a 4-byte length prefix for a non-BinaryMarshaler State, got %d bytes", buf.Len())
	}

	if err := r.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected LoadState to consume the length prefix, %d bytes left over", buf.Len())
	}
}

// TestResources_SaveLoadState_SharesStreamWithTrailingData guards the whole
// reason SaveState/LoadState are length-prefixed: Game.Save appends an ECS
// snapshot right after State in the same file, and Game.Load must be able
// to read State off the front without consuming any of what follows.
func TestResources_SaveLoadState_SharesStreamWithTrailingData(t *testing.T) {
	r := NewResources(&GameProps{}, spatial.Config{}, testMarshaledState{N: 7}, testTelemetry{})

	var buf bytes.Buffer
	if err := r.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	trailing := []byte("trailing data, e.g. a gzip ECS snapshot")
	buf.Write(trailing)

	r2 := NewResources(&GameProps{}, spatial.Config{}, testMarshaledState{}, testTelemetry{})
	if err := r2.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if r2.State().N != 7 {
		t.Errorf("State().N = %d, want 7", r2.State().N)
	}
	if buf.String() != string(trailing) {
		t.Errorf("LoadState consumed into the trailing data: left %q, want %q", buf.String(), trailing)
	}
}
