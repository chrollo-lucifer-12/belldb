package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ChunkMetaData struct {
	MinTs int64  `json:"min_ts"`
	MaxTs int64  `json:"max_ts"`
	Path  string `json:"path"`
}

type Metadata struct {
	Chunks []ChunkMetaData `json:"chunks"`
}

func SaveMeta(metadata Metadata, dir string) error {
	metaFile := filepath.Join(dir, "meta.json")

	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

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

func LoadMeta(metaFilePath string) (Metadata, error) {

	data, err := os.ReadFile(metaFilePath)
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
