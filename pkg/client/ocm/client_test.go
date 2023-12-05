package ocm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchPages(t *testing.T) {
	testPager := NewMoqPager(make([]testDataType, 21))
	n := 0
	// TODO: go 1.21 can infer the generic parameters from testPager.
	testFetcher := fetchPages[
		*paginatedRequestMoqImpl,
		*paginatedResponseMoqImpl,
		*paginatedResponseMoqImpl,
		testDataType]
	err := testFetcher(testPager, 5, 30, func(_ testDataType) bool {
		n++
		return true
	})
	assert.NoError(t, err)
	assert.Equal(t, 21, n, "Should fetch data from all pages")

	n = 0
	err = testFetcher(testPager, 5, 30, func(_ testDataType) bool {
		n++
		return n < 11
	})
	assert.NoError(t, err)
	assert.Equal(t, 11, n, "Should stop if the functor returns false")

	n = 0
	err = testFetcher(testPager, 5, 2, func(_ testDataType) bool {
		n++
		return true
	})
	assert.Error(t, err, "Should return error if too many pages")
	assert.Equal(t, 5*2, n, "Should iterate over all pages until error")

	n = 0
	err = testFetcher(testPager, 5, 5, func(_ testDataType) bool {
		n++
		return true
	})
	assert.NoError(t, err, "Should not return error if page[maxPage] has < pageSize elements")
	assert.Equal(t, 21, n, "Should iterate over all elements")
}
