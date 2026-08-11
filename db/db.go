package db

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type Point struct {
	Timestamp int64
	Value     float64
}

type Series struct {
	Points []Point
}

type DB struct {
	Series map[string]*Series
	log    *Log
}

func errMetricNotFound(metric string) error {
	return fmt.Errorf("metric not found: %s", metric)
}

func errTimestampNotFound(timestamp int64) error {
	return fmt.Errorf("timestamp not found: %d", timestamp)
}

func NewDB() *DB {
	return &DB{
		Series: make(map[string]*Series),
	}
}

func (db *DB) Open(filepath string) error {
	fp, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Join(errors.New("db open"), err)
	}

	db.log = NewLog(fp)

	for {
		sp, err := DecodeRecord(fp)

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		series, ok := db.Series[sp.metric]
		if !ok {
			series = &Series{}
			db.Series[sp.metric] = series
		}

		series.Points = append(series.Points, Point{
			Timestamp: sp.timestamp,
			Value:     sp.value,
		})
	}

	return nil
}

func (db *DB) Close() {
	db.log.aof.Close()
}

func (db *DB) Put(metric string, timestamp int64, value float64) error {

	err := db.log.Write(EncodeRecord(SavePoint{metric: metric, timestamp: timestamp, value: value}))
	if err != nil {
		return err
	}

	series, ok := db.Series[metric]

	if !ok {
		series = &Series{}
		db.Series[metric] = series
	}

	series.Points = append(series.Points, Point{Timestamp: timestamp, Value: value})

	return nil
}

func (db *DB) Get(metric string, timestamp int64) (float64, error) {
	series, ok := db.Series[metric]
	if !ok {
		return -1, errMetricNotFound(metric)
	}

	idx := db.lowerBound(series, timestamp, 0, len(series.Points))

	if idx == len(series.Points) || series.Points[idx].Timestamp != timestamp {
		return -1, errTimestampNotFound(timestamp)
	}

	return series.Points[idx].Value, nil
}

func (db *DB) Range(metric string, start, end int64) []Point {
	series, ok := db.Series[metric]
	if !ok {
		return nil
	}

	startIdx := db.lowerBound(series, start, 0, len(series.Points))
	endIdx := db.lowerBound(series, end, 0, len(series.Points))

	return series.Points[startIdx:endIdx]
}

func (db *DB) lowerBound(series *Series, timestamp int64, low, high int) int {
	for low < high {
		mid := low + (high-low)/2

		if series.Points[mid].Timestamp < timestamp {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return low
}
