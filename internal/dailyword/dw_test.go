package dailyword

import "testing"

func Test_trimColons(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			"success trim",
			"test (test)",
			"test",
		},
		{
			"success trim with two brackets",
			"test (test) (test)",
			"test (test)",
		},
		{
			"no trim",
			"test (test) test",
			"test (test) test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimBrackets(tt.s); got != tt.want {
				t.Errorf("trimBrackets() = %v, want %v", got, tt.want)
			}
		})
	}
}
