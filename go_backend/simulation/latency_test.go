package simulation

import "testing"

func TestComputeAvgMoveMs(t *testing.T) {
	if got := ComputeAvgMoveMs(1000, 10); got != 100 {
		t.Fatalf("got %d", got)
	}
	if got := ComputeAvgMoveMs(1000, 0); got != 0 {
		t.Fatalf("got %d", got)
	}
}

func TestPercentileMs_P95(t *testing.T) {
	vals := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	got := PercentileMs(vals, 95)
	if got != 100 {
		t.Fatalf("p95 got %d want 100", got)
	}
}

func TestMeanMs(t *testing.T) {
	if got := MeanMs([]int64{100, 200, 300}); got != 200 {
		t.Fatalf("got %v", got)
	}
}
