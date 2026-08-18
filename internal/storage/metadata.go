package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ChunkMetaData struct {
	MinTs int64  `json:"min_ts"`
	MaxTs int64  `json:"max_ts"`
	Count int    `json:"count"`
	Path  string `json:"path"`
}

type Metadata struct {
	Chunks []ChunkMetaData `json:"chunks"`
}

func SaveMeta(metadata Metadata, dir string) error {
	metaFile := filepath.Join(dir, "meta.json")

	if err := AtomicWrite(metaFile, func(f *os.File) error {
		data, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("write metadata: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("save metadat: %w", err)
	}

	return nil
}

func LoadMeta(metaFilePath string) (Metadata, error) {
	file, err := OpenFile(metaFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, nil
		}

		return Metadata{}, err
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return Metadata{}, fmt.Errorf("read metadata: %w", err)
	}

	var metadata Metadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return metadata, nil
}
