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

func TestResources_SaveLoadState_NoopWithoutBinaryMarshaler(t *testing.T) {
	r := NewResources(&GameProps{}, spatial.Config{}, testPlainState{N: 7}, testTelemetry{})

	var buf bytes.Buffer
	if err := r.SaveState(&buf); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no bytes written for a non-BinaryMarshaler State, got %d", buf.Len())
	}

	if err := r.LoadState(&buf); err != nil {
		t.Fatalf("LoadState: %v", err)
	}
}
