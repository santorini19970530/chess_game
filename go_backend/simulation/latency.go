package simulation

import "sort"

// PercentileMs returns the nearest-rank percentile of duration values (ms).
// p is 0..100. Empty input returns 0.
func PercentileMs(values []int64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	cp := append([]int64(nil), values...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	if p == 0 {
		return cp[0]
	}
	idx := (p * len(cp) + 99) / 100 // ceil(p/100 * n) - 1 style nearest up
	if idx < 1 {
		idx = 1
	}
	if idx > len(cp) {
		idx = len(cp)
	}
	return cp[idx-1]
}

// MeanMs returns the arithmetic mean of duration values (ms).
func MeanMs(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}
