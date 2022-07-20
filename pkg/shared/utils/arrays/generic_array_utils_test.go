package arrays

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_FindFirst(t *testing.T) {
	type args struct {
		predicate func(x interface{}) bool
		values    []interface{}
	}
	tests := []struct {
		name          string
		args          args
		expectedIndex int
		expectedValue interface{}
	}{
		{
			name: "Return -1 index and nil if no match",
			args: args{
				predicate: func(x interface{}) bool { return x.(int) > 5 },
				values:    []interface{}{1, 2, 3, 4},
			},
			expectedIndex: -1,
			expectedValue: nil,
		},
		{
			name: "Return with index and matching integer",
			args: args{
				predicate: func(x interface{}) bool { return x.(int) > 5 },
				values:    []interface{}{1, 2, 3, 4, 5, 6, 7, 8},
			},
			expectedIndex: 5,
			expectedValue: 6,
		},
		{
			name: "Return with index and matching string",
			args: args{
				predicate: func(x interface{}) bool { return x.(string) == "Red" },
				values:    []interface{}{"This", "Is", "Red", "Hat"},
			},
			expectedIndex: 2,
			expectedValue: "Red",
		},
		{
			name: "Return with index and matching incompatible types",
			args: args{
				predicate: func(x interface{}) bool {
					if val, ok := x.(int); ok {
						return val > 5
					}
					return false
				},
				values: []interface{}{1, 2, "Hello", 4, 5, 6, 7, 8},
			},
			expectedIndex: 5,
			expectedValue: 6,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotVal := FindFirst(tt.args.predicate, tt.args.values...)
			Expect(gotIndex).To(Equal(tt.expectedIndex))
			if gotVal != nil {
				Expect(gotVal).To(Equal(tt.expectedValue))
			}
		})
	}
}
