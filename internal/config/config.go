package config

import (
	"os"
	"path/filepath"
)

var DATA_DIR = getDataDir()

var LOG_DIR = getLogDir()

func getLogDir() string {
	if dir := os.Getenv("BELLDB_LOG_DIR"); dir != "" {
		return dir
	}

	return filepath.Join("data", "wal")
}

func getDataDir() string {
	if dir := os.Getenv("BELLDB_DATA_DIR"); dir != "" {
		return dir
	}

	return filepath.Join("data", "chunks")
}
