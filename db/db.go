package db

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type DB struct {
	kv  *KV
	log *Log
}

func errMetricNotFound(metric string) error {
	return fmt.Errorf("metric not found: %s", metric)
}

func errTimestampNotFound(timestamp int64) error {
	return fmt.Errorf("timestamp not found: %d", timestamp)
}

func NewDB() *DB {
	return &DB{}
}

func (db *DB) Open(filepath string) error {

	db.kv = NewKV()

	fp, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Join(errors.New("db open"), err)
	}

	db.log = NewLog(fp)

	return db.recover()
}

func (db *DB) Close() error {
	return db.log.aof.Close()
}

func (db *DB) Put(metric string, timestamp int64, value float64) error {

	err := db.log.Write(EncodeRecord(SavePoint{metric: metric, timestamp: timestamp, value: value}))
	if err != nil {
		return err
	}

	db.kv.Put(metric, timestamp, value)

	return nil
}

func (db *DB) Get(metric string, timestamp int64) (float64, error) {
	return db.kv.Get(metric, timestamp)
}

func (db *DB) recover() error {
	for {
		sp, err := DecodeRecord(db.log.aof)

		if err == io.EOF {
			break
		}

		if errors.Is(err, io.ErrUnexpectedEOF) {
			break
		}

		if err != nil {
			return err
		}

		db.kv.Put(sp.metric, sp.timestamp, sp.value)
	}

	return nil
}
