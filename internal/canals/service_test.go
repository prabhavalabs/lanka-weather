package canals

import "testing"

func TestInArea(t *testing.T) {
	cases := []struct {
		name     string
		lon, lat float64
		want     bool
	}{
		{"Colombo Fort", 79.8428, 6.9344, true},
		{"Wellampitiya", 79.8921, 6.9389, true},
		{"Kandy", 80.6337, 7.2906, false},
		{"Negombo", 79.8358, 7.2083, false},
	}
	for _, c := range cases {
		if got := InArea(c.lon, c.lat); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}
