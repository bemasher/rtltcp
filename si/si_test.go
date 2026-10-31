package si

import "testing"

func TestScientificNotationSet(t *testing.T) {
	tests := []struct {
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

	for _, test := range tests {
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
