package gokebiten

import (
	"encoding"
	"encoding/binary"
	"io"

	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/spatial"
)

// Resources bundles what Game needs (GameProps, input, world config, and
// built-in TPS/EntityCount telemetry) with two game-owned shapes: S for
// control state, T for display telemetry.
type Resources[S, T any] struct {
	gameProps   *GameProps
	inputs      control.InputEvents
	spaceConfig spatial.Config
	measuredTPS int

	state     S
	telemetry T
}

var _ resources = (*Resources[struct{}, struct{}])(nil)

func NewResources[S, T any](gameProps *GameProps, spaceConfig spatial.Config, state S, telemetry T) *Resources[S, T] {
	return &Resources[S, T]{gameProps: gameProps, spaceConfig: spaceConfig, state: state, telemetry: telemetry}
}

func (r *Resources[S, T]) State() *S     { return &r.state }
func (r *Resources[S, T]) Telemetry() *T { return &r.telemetry }
func (r *Resources[S, T]) TPS() *int     { return &r.measuredTPS }

func (r *Resources[S, T]) GetGameProps() *GameProps             { return r.gameProps }
func (r *Resources[S, T]) GetSpaceConfig() spatial.Config       { return r.spaceConfig }
func (r *Resources[S, T]) GetInputEvents() *control.InputEvents { return &r.inputs }

// SaveState writes a uint32 length prefix followed by State to w — the
// prefix is 0 if S doesn't implement encoding.BinaryMarshaler, so games
// with nothing worth persisting beyond the ECS snapshot don't need to do
// anything special. Length-prefixed (rather than reading w to EOF) so
// SaveState/LoadState can share a stream with data written after them,
// e.g. Game.Save appending an ECS snapshot to the same file.
func (r *Resources[S, T]) SaveState(w io.Writer) error {
	bm, ok := any(&r.state).(encoding.BinaryMarshaler)
	if !ok {
		return binary.Write(w, binary.BigEndian, uint32(0))
	}
	data, err := bm.MarshalBinary()
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// LoadState restores State from rd, written by SaveState — a no-op (beyond
// consuming the length prefix, to leave rd positioned right after State for
// any data that follows) if S doesn't implement encoding.BinaryUnmarshaler.
func (r *Resources[S, T]) LoadState(rd io.Reader) error {
	var n uint32
	if err := binary.Read(rd, binary.BigEndian, &n); err != nil {
		return err
	}
	bu, ok := any(&r.state).(encoding.BinaryUnmarshaler)
	if !ok {
		_, err := io.CopyN(io.Discard, rd, int64(n))
		return err
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(rd, data); err != nil {
		return err
	}
	return bu.UnmarshalBinary(data)
}

// Resettable lets Telemetry hook into each stats interval — if T implements
// it, Reset is called right after TPS is refreshed.
type Resettable interface{ Reset() }

func (r *Resources[S, T]) Reset() {
	if resettable, ok := any(&r.telemetry).(Resettable); ok {
		resettable.Reset()
	}
}
