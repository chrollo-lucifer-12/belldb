package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ChunkMetaData struct {
	Metadata
	Path string
}

type Metadata struct {
	MinTs int64 `json:"minTs"`
	MaxTs int64 `json:"maxTs"`
}

func saveMeta(metaData Metadata, dir string) error {
	data, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	metaFile := filepath.Join(dir, "meta.json")

	tmpFile := metaFile + ".tmp"

	fp, err := os.OpenFile(
		tmpFile,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return err
	}

	if _, err := fp.Write(data); err != nil {
		fp.Close()
		return err
	}

	if err := fp.Sync(); err != nil {
		fp.Close()
		return err
	}

	if err := fp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile, metaFile)
}

func loadMeta(dir string) (Metadata, error) {

	metaFile := filepath.Join(dir, "meta.json")

	data, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, nil
		}
		return Metadata{}, err
	}

	var metadata Metadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, err
	}

	return metadata, nil

}
