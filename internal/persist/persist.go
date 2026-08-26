// Package persist implements Game.Save/Game.Load: resource gob-framing, ECS snapshot splicing.
package persist

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

// ListSaves returns every save found for basePath, "" (quicksave) first.
func ListSaves(basePath string) ([]string, error) {
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

// Save pauses ecs and writes resources followed by the ECS snapshot to filePath(basePath, label).
func Save(ecs *goke.ECS, basePath, label string, resources ...any) error {
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

	if err := saveResources(out, resources...); err != nil {
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

// Load restores a snapshot written by Save into resources and a freshly constructed ecs.
func Load(ecs *goke.ECS, basePath, label string, comps []goke.CompToken, resources ...any) error {
	in, err := os.Open(filePath(basePath, label))
	if err != nil {
		return err
	}
	defer in.Close()

	if err := loadResources(in, resources...); err != nil {
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

// saveResources gob-encodes resources into a length-prefixed frame so loadResources can consume exactly its bytes.
func saveResources(w io.Writer, resources ...any) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, r := range resources {
		if err := enc.Encode(r); err != nil {
			return fmt.Errorf("gokebiten: encode resource %T: %w", r, err)
		}
	}
	if err := binary.Write(w, binary.BigEndian, uint32(buf.Len())); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// loadResources restores resources (pointers, same order as saveResources) from r.
func loadResources(r io.Reader, resources ...any) error {
	var n uint32
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return err
	}
	data := make([]byte, n)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewReader(data))
	for _, res := range resources {
		if err := dec.Decode(res); err != nil {
			return fmt.Errorf("gokebiten: decode resource %T: %w", res, err)
		}
	}
	return nil
}
