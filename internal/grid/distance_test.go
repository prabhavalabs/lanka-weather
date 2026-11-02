package grid

import (
	"math"
	"testing"
)

func TestCoastDistance(t *testing.T) {
	if got := CoastDistanceKm(80.7, 7.3); got != 0 {
		t.Fatalf("on land: %v", got)
	}
	// Point Pedro, Sri Lanka's northern tip, is near 9.83° N 80.21° E, about
	// 550 km from a storm at 14.7° N 80.7° E.
	if got := CoastDistanceKm(80.7, 14.7); math.Abs(got-545) > 20 {
		t.Fatalf("from the Bay of Bengal: %v", got)
	}
}
