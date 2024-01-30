package impl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataPagerMoq(t *testing.T) {
	pager := NewMoqPager([]testDataType{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	cases := map[string]struct {
		pageSize     int
		pageNumber   int
		stopOnValue  testDataType
		expectedSize int
		expectedLast testDataType
	}{
		"first page size 3": {
			pageSize:     3,
			pageNumber:   1,
			expectedSize: 3,
			expectedLast: 3,
		},
		"first page size 6": {
			pageSize:     6,
			pageNumber:   1,
			expectedSize: 6,
			expectedLast: 6,
		},
		"second page": {
			pageSize:     3,
			pageNumber:   2,
			expectedSize: 3,
			expectedLast: 6,
		},
		"break on second page": {
			pageSize:     3,
			pageNumber:   2,
			stopOnValue:  5,
			expectedSize: 3,
			expectedLast: 5,
		},
		"last short page": {
			pageSize:     3,
			pageNumber:   4,
			expectedSize: 1,
			expectedLast: 10,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			response, err := pager.Size(c.pageSize).Page(c.pageNumber).Send()
			assert.NoError(t, err)
			assert.Equal(t, c.expectedSize, response.Size())
			var last testDataType = 0
			response.Items().Each(func(value testDataType) bool {
				last = value
				return value != c.stopOnValue
			})
			assert.Equal(t, c.expectedLast, last)
		})
	}
}
