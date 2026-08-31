package world

import "testing"

func TestClampSpeed(t *testing.T) {
	cases := []struct {
		name    string
		v, max  int32
		wantVal int32
	}{
		{"max<=0 means no limit, unchanged", 1000, 0, 1000},
		{"v within [0,max] unchanged", 3, 10, 3},
		{"v above max clamps to max", 15, 10, 10},
		{"v exactly at max unchanged", 10, 10, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := clampSpeed(c.v, c.max); got != c.wantVal {
				t.Errorf("clampSpeed(%d, %d) = %d, want %d", c.v, c.max, got, c.wantVal)
			}
		})
	}
}
