package filter_test

import (
	"math"
	"sync"
	"testing"

	"github.com/adrianozp/gaardrail/pkg/control/filter"
)

func mustReference(t *testing.T, spec filter.Spec) *filter.Signal {
	t.Helper()
	s, err := filter.NewReference(spec)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNonePassthrough(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.None, Size: 5})
	if got := s.Filter(50); got != 50 {
		t.Fatalf("none must be a passthrough, got %v", got)
	}
}

func TestEmptySpecNormalizesToNone(t *testing.T) {
	s := mustReference(t, filter.Spec{})
	if got := s.Spec(); got != (filter.Spec{Kind: filter.None, Size: 1}) {
		t.Fatalf("got %+v", got)
	}
}

func TestMovingAverageRampsStepOverNSamples(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.MovingAverage, Size: 4})
	want := []float64{12.5, 25, 37.5, 50}
	for i := range want {
		if got := s.Filter(50); math.Abs(got-want[i]) > 1e-9 {
			t.Fatalf("sample %d: got %v want %v", i, got, want[i])
		}
	}
}

func TestExponentialConstantPerTick(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.Exponential, Size: 2})
	a := math.Exp(-0.5)
	if got, want := s.Filter(50), (1-a)*50; math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMeasurementSeedsWithFirstSample(t *testing.T) {
	s, err := filter.NewMeasurement(filter.Spec{Kind: filter.Exponential, Size: 4})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Filter(80); got != 80 {
		t.Fatalf("first sample must pass through, got %v", got)
	}
}

func TestMeasurementMovingAverageSeedsWithFirstSample(t *testing.T) {
	s, err := filter.NewMeasurement(filter.Spec{Kind: filter.MovingAverage, Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	s.Filter(40)
	if got := s.Filter(80); got != 60 {
		t.Fatalf("expected (40+80)/2, got %v", got)
	}
}

func TestInvalidKind(t *testing.T) {
	if _, err := filter.NewReference(filter.Spec{Kind: "median", Size: 3}); err == nil {
		t.Fatal("invalid kind must return an error")
	}
}

func TestSeedSetsState(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.Exponential, Size: 3})
	s.Seed(50)
	if got := s.Filter(50); math.Abs(got-50) > 1e-9 {
		t.Fatalf("after Seed(50), Filter(50) must hold 50, got %v", got)
	}
}

func TestResetClearsState(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.MovingAverage, Size: 2})
	s.Seed(100)
	s.Reset()
	if got := s.Filter(10); got != 5 {
		t.Fatalf("expected (10+0)/2 after reset, got %v", got)
	}
}

func TestConcurrentUse(t *testing.T) {
	s := mustReference(t, filter.Spec{Kind: filter.Exponential, Size: 3})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			s.Filter(1)
			s.Seed(2)
			s.Reset()
		})
	}
	wg.Wait()
}
