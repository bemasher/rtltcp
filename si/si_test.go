package si

import (
	"math"
	"testing"
)

var setTests = []struct {
	in      string
	want    float64
	wantErr bool
}{
	{"100M", 100e6, false},
	{"2.4M", 2.4e6, false},
	{"100MHz", 100e6, false},
	{"1k", 1e3, false},
	{"1m", 1e-3, false},
	{"-3", -3, false},
	{"1e6", 1e6, false},
	{"1E6", 1e6, false},
	{"1.5e+9", 1.5e9, false},
	{"1.5e-6", 1.5e-6, false},
	{"not a number", 0, true},
}

func TestScientificNotationSet(t *testing.T) {
	for _, test := range setTests {
		var got ScientificNotation
		err := got.Set(test.in)
		if test.wantErr {
			if err == nil {
				t.Errorf("Set(%q) = %v, want error", test.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Set(%q): %v", test.in, err)
			continue
		}
		if float64(got) != test.want {
			t.Errorf("Set(%q) = %v, want %v", test.in, got, test.want)
		}
	}
}

func FuzzScientificNotationSet(f *testing.F) {
	for _, test := range setTests {
		f.Add(test.in)
	}

	f.Fuzz(func(t *testing.T, in string) {
		var got ScientificNotation
		if err := got.Set(in); err != nil {
			return
		}

		// Any parsable value must survive a String -> Set round trip.
		if v := float64(got); math.IsInf(v, 0) || math.IsNaN(v) {
			return
		}

		var back ScientificNotation
		if err := back.Set(got.String()); err != nil {
			t.Fatalf("Set(%q) = %v, String() = %q: %v", in, float64(got), got.String(), err)
		}
		if back != got {
			t.Errorf("Set(%q) = %v, round trip through %q = %v", in, float64(got), got.String(), float64(back))
		}
	})
}
