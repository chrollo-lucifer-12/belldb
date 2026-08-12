package flag

import (
	"os"
	"path/filepath"
)

var DATA_DIR = getDataDir()

func getDataDir() string {
	if dir := os.Getenv("BELLDB_DATA_DIR"); dir != "" {
		return dir
	}

	return filepath.Join("data", "chunks")
}
