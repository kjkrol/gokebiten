package engine

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kjkrol/goke/v3"
)

// listSaves returns every save found for basePath, "" (quicksave) first.
func listSaves(basePath string) ([]string, error) {
	var labels []string
	if _, err := os.Stat(filePath(basePath, "")); err == nil {
		labels = append(labels, "")
	}

	matches, err := filepath.Glob(basePath + ".game.*.save")
	if err != nil {
		return nil, err
	}
	prefix, suffix := basePath+".game.", ".save"
	var named []string
	for _, m := range matches {
		named = append(named, strings.TrimSuffix(strings.TrimPrefix(m, prefix), suffix))
	}
	sort.Strings(named)

	return append(labels, named...), nil
}

// save pauses ecs and writes groups followed by the ECS snapshot to filePath(basePath, label).
func save(ecs *goke.ECS, basePath, label string, groups map[string][]any) error {
	ecs.Pause()
	defer ecs.Resume()

	tmp, err := os.CreateTemp("", "gokebiten-ecs-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := ecs.Save(tmpPath); err != nil {
		return err
	}

	out, err := os.Create(filePath(basePath, label))
	if err != nil {
		return err
	}
	defer out.Close()

	if err := saveResources(out, groups); err != nil {
		return err
	}

	ecsData, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer ecsData.Close()
	_, err = io.Copy(out, ecsData)
	return err
}

// load restores a snapshot written by save into groups and a freshly constructed ecs.
func load(ecs *goke.ECS, basePath, label string, comps []goke.CompToken, groups map[string][]any) error {
	in, err := os.Open(filePath(basePath, label))
	if err != nil {
		return err
	}
	defer in.Close()

	if err := loadResources(in, groups); err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "gokebiten-ecs-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return ecs.Load(tmpPath, comps...)
}

// filePath is the quicksave path when label is empty, else a named save path.
func filePath(basePath, label string) string {
	if label == "" {
		return basePath + ".game.save"
	}
	return basePath + ".game." + label + ".save"
}

// saveResources gob-encodes groups (each a name and its own gob sub-stream)
// into one length-prefixed frame so loadResources can match by name, not position.
func saveResources(w io.Writer, groups map[string][]any) error {
	encoded := make(map[string][]byte, len(groups))
	for name, targets := range groups {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		for _, t := range targets {
			if err := enc.Encode(t); err != nil {
				return fmt.Errorf("gokebiten: encode resource %q: %w", name, err)
			}
		}
		encoded[name] = buf.Bytes()
	}

	var outer bytes.Buffer
	if err := gob.NewEncoder(&outer).Encode(encoded); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint32(outer.Len())); err != nil {
		return err
	}
	_, err := w.Write(outer.Bytes())
	return err
}

// loadResources restores groups from r, matching each by name — a name
// absent from the saved data (e.g. a plugin added since the save was
// written) is skipped, leaving its targets at their current values.
func loadResources(r io.Reader, groups map[string][]any) error {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return err
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}

	var encoded map[string][]byte
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&encoded); err != nil {
		return err
	}

	for name, targets := range groups {
		blob, ok := encoded[name]
		if !ok {
			continue
		}
		dec := gob.NewDecoder(bytes.NewReader(blob))
		for _, t := range targets {
			if err := dec.Decode(t); err != nil {
				return fmt.Errorf("gokebiten: decode resource %q: %w", name, err)
			}
		}
	}
	return nil
}
