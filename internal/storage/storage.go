package storage

import (
	"fmt"
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

func LoadDODChunk(path string) (*DODChunk, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	return DecodeDODChunk(fp)
}

func FlushDODChunk(metric string, chunk *DODChunk) (ChunkMetaData, error) {
	dir := filepath.Join(config.DATA_DIR, metric)

	filename := fmt.Sprintf("%d.chunk", chunk.firstTimestamp)
	path := filepath.Join(dir, filename)

	file, err := os.Create(path)
	if err != nil {
		return ChunkMetaData{}, err
	}
	defer file.Close()

	if err := EncodeDODChunk(file, *chunk); err != nil {
		return ChunkMetaData{}, nil
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
		return ChunkMetaData{}, err
	}
	defer fp.Close()

	if err := EncodePoints(fp, points); err != nil {
		return ChunkMetaData{}, err
	}

	return meta, nil
}
