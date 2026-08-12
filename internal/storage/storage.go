package storage

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/belldb/internal/config"
)

func LoadChunk(path string) ([]Point, error) {

	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	return DecodePoints(fp)

}

func Flush(metric string, points []Point) error {

	minTs := points[0].Timestamp
	maxTs := points[len(points)-1].Timestamp

	dir := filepath.Join(config.DATA_DIR, metric)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := saveMeta(Metadata{MinTs: minTs, MaxTs: maxTs}, dir); err != nil {
		return err
	}

	startTimestamp := points[0].Timestamp
	timestampStr := strconv.Itoa(int(startTimestamp))

	path := filepath.Join(dir, timestampStr)

	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	if err := EncodePoints(fp, points); err != nil {
		return err
	}

	return fp.Sync()
}
