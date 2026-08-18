package config

import (
	"os"
	"path/filepath"
	"time"
)

var DATA_DIR = getDataDir()
var LOG_DIR = getLogDir()

var SegmentSize int64
var BufferSize int
var QueueSize int
var SyncInterval time.Duration
var BatchSize int

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
