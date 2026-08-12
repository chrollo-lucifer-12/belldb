package db

type KV struct {
	Series map[string]*Series
}

func NewKV() *KV {
	return &KV{Series: make(map[string]*Series)}
}

func (kv *KV) Put(metric string, timestamp int64, value float64) error {
	series, ok := kv.Series[metric]

	if !ok {
		series = &Series{name: metric}
		kv.Series[metric] = series
	}

	return series.Append(Point{Timestamp: timestamp, Value: value})
}

func (kv *KV) Get(metric string, timestamp int64) (float64, error) {
	series, ok := kv.Series[metric]
	if !ok {
		return -1, errMetricNotFound(metric)
	}

	return series.Get(timestamp)
}

func (kv *KV) Range(metric string, start, end int64) []Point {
	series, ok := kv.Series[metric]
	if !ok {
		return nil
	}

	return series.Range(start, end)
}
