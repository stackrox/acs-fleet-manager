package shared

import (
	"reflect"
	"testing"
)

func TestRoundUp(t *testing.T) {
	cases := []struct {
		name     string
		number   int
		multiple int
		expected int
	}{
		{
			name:     "round up 6 to nearest multiple of 4",
			number:   6,
			multiple: 4,
			expected: 8,
		},
		{
			name:     "round up 7 to nearest multiple of 4",
			number:   7,
			multiple: 4,
			expected: 8,
		},
		{
			name:     "round up 8 to nearest multiple of 4",
			number:   8,
			multiple: 4,
			expected: 8,
		},
		{
			name:     "round up 11 to nearest multiple of 2",
			number:   11,
			multiple: 2,
			expected: 12,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := RoundUp(tt.number, tt.multiple)
			if !reflect.DeepEqual(res, tt.expected) {
				t.Errorf("RoundUp() got = %+v, expected %+v", res, tt.expected)
			}
		})
	}
}

func TestRoundDown(t *testing.T) {
	cases := []struct {
		name     string
		number   int
		multiple int
		expected int
	}{
		{
			name:     "round down 6 to nearest multiple of 4",
			number:   6,
			multiple: 4,
			expected: 4,
		},
		{
			name:     "round down 7 to nearest multiple of 4",
			number:   7,
			multiple: 4,
			expected: 4,
		},
		{
			name:     "round down 8 to nearest multiple of 4",
			number:   8,
			multiple: 4,
			expected: 8,
		},
		{
			name:     "round down 11 to nearest multiple of 2",
			number:   11,
			multiple: 2,
			expected: 10,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := RoundDown(tt.number, tt.multiple)
			if !reflect.DeepEqual(res, tt.expected) {
				t.Errorf("RoundDown() got = %+v, expected %+v", res, tt.expected)
			}
		})
	}
}
