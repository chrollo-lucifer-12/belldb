package db

type Point struct {
	Timestamp int64
	Value     float64
}

type Series struct {
	Points []Point
}

type KV struct {
	Series map[string]*Series
}

func NewKV() *KV {
	return &KV{Series: make(map[string]*Series)}
}

func (kv *KV) Put(metric string, timestamp int64, value float64) {
	series, ok := kv.Series[metric]

	if !ok {
		series = &Series{}
		kv.Series[metric] = series
	}

	series.Points = append(series.Points, Point{Timestamp: timestamp, Value: value})
}

func (kv *KV) Get(metric string, timestamp int64) (float64, error) {
	series, ok := kv.Series[metric]
	if !ok {
		return -1, errMetricNotFound(metric)
	}

	idx := kv.lowerBound(series, timestamp, 0, len(series.Points))

	if idx == len(series.Points) || series.Points[idx].Timestamp != timestamp {
		return -1, errTimestampNotFound(timestamp)
	}

	return series.Points[idx].Value, nil
}

func (kv *KV) Range(metric string, start, end int64) []Point {
	series, ok := kv.Series[metric]
	if !ok {
		return nil
	}

	startIdx := kv.lowerBound(series, start, 0, len(series.Points))
	endIdx := kv.lowerBound(series, end, 0, len(series.Points))

	return series.Points[startIdx:endIdx]
}

func (kv *KV) lowerBound(series *Series, timestamp int64, low, high int) int {
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
