package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/belldb/internal/config"
)

func LoadDODChunk(
	path string,
	timestamps []byte,
	values []byte,
) (*DODChunk, []byte, []byte, error) {
	file, err := OpenFile(path)
	if err != nil {
		return nil, timestamps, values, err
	}
	defer CloseFile(file)

	chunk, err := DecodeDODChunk(file, timestamps, values)
	if err != nil {
		return nil, timestamps, values,
			fmt.Errorf("decode chunk %s: %w", path, err)
	}

	return chunk, timestamps, values, nil
}

func FlushDODChunk(metric string, chunk *DODChunk) (ChunkMetaData, error) {
	dir := filepath.Join(config.DATA_DIR, metric)

	path := filepath.Join(dir, fmt.Sprintf("%d.chunk", chunk.firstTimestamp))

	if err := AtomicWrite(path, func(f *os.File) error {
		return EncodeDODChunk(f, *chunk)
	}); err != nil {
		return ChunkMetaData{}, fmt.Errorf("flush chunk: %w", err)
	}

	return ChunkMetaData{
		MinTs: chunk.firstTimestamp,
		MaxTs: chunk.lastTimestamp,
		Path:  path,
		Count: chunk.count,
	}, nil
}

func Flush(metric string, points []Point) (ChunkMetaData, error) {

	minTs := points[0].Timestamp
	maxTs := points[len(points)-1].Timestamp

	dir := filepath.Join(config.DATA_DIR, metric)

	timestampStr := strconv.Itoa(int(minTs))
	path := filepath.Join(dir, timestampStr)

	meta := ChunkMetaData{MinTs: minTs, MaxTs: maxTs, Path: path, Count: len(points)}

	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return ChunkMetaData{}, fmt.Errorf("flush :%w", err)
	}
	defer fp.Close()

	if err := EncodePoints(fp, points); err != nil {
		return ChunkMetaData{}, fmt.Errorf("flush :%w", err)
	}

	return meta, nil
}
