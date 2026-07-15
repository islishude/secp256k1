package cpufeat

import "testing"

func TestHasADXAndBMI2(t *testing.T) {
	tests := []struct {
		name string
		ebx  uint32
		want bool
	}{
		{name: "none"},
		{name: "bmi2 only", ebx: leaf7BMI2},
		{name: "adx only", ebx: leaf7ADX},
		{name: "both", ebx: leaf7ADX | leaf7BMI2, want: true},
		{name: "both and unrelated", ebx: leaf7ADX | leaf7BMI2 | 1<<5, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasADXAndBMI2(test.ebx); got != test.want {
				t.Fatalf("hasADXAndBMI2(%#x) = %t, want %t", test.ebx, got, test.want)
			}
		})
	}
}
