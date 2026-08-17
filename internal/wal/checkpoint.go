package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/belldb/internal/config"
)

func (log *Log) Checkpoint(lsn uint64) error {
	dir := config.LOG_DIR

	checkpointPath := filepath.Join(dir, "checkpoint")

	fp, err := os.OpenFile(checkpointPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()

	return binary.Write(fp, binary.LittleEndian, lsn)
}

func (log *Log) rotate() error {
	dir := config.LOG_DIR

	log.segmentID++

	checkpointIDStr := strconv.Itoa(log.segmentID)

	walPath := filepath.Join(dir, checkpointIDStr)

	aof, err := os.OpenFile(
		walPath,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)

	if err != nil {
		return err
	}

	log.aof = aof
	log.buf = bufio.NewWriterSize(aof, 1024*1024)

	return nil
}

func (log *Log) Recover() ([]SavePoint, error) {
	lsn, err := lastCheckpoint()

	if err != nil {
		return nil, err
	}

	return getRecords(lsn)
}

func lastCheckpoint() (uint64, error) {
	path := filepath.Join(config.LOG_DIR, "checkpoint")

	fp, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer fp.Close()

	info, err := fp.Stat()
	if err != nil {
		return 0, err
	}

	if info.Size() < 8 {
		return 0, nil
	}

	_, err = fp.Seek(-8, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	var lsn uint64
	err = binary.Read(fp, binary.LittleEndian, &lsn)

	return lsn, err
}

func getRecords(lsn uint64) ([]SavePoint, error) {
	segmentID := lsn/WalLimit + 1

	var res []SavePoint

	for {
		walPath := filepath.Join(config.LOG_DIR, strconv.Itoa(int(segmentID)))

		fp, err := os.Open(walPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return nil, err
		}

		for {
			sp, err := DecodeRecord(fp)

			if sp.LSN < lsn {
				continue
			}

			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}

			if err != nil {
				return nil, err
			}

			res = append(res, sp)
		}

		fp.Close()

		segmentID++
	}

	return res, nil
}
