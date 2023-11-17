package ocm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchPages(t *testing.T) {
	testPager := NewMoqPager(make([]byte, 21))
	n := 0
	// TODO: go 1.21 can infer the generic parameters from testPager.
	testFetcher := fetchPages[
		*dataPagerMoqImpl[byte],
		*paginatedResponseMoqImpl[byte],
		*paginatedResponseMoqImpl[byte],
		byte]
	err := testFetcher(&testPager, 5, 30, func(c byte) bool {
		n++
		return true
	})
	assert.NoError(t, err)
	assert.Equal(t, 21, n, "Should fetch data from all pages")

	n = 0
	err = testFetcher(&testPager, 5, 30, func(c byte) bool {
		n++
		return n < 11
	})
	assert.NoError(t, err)
	assert.Equal(t, 11, n, "Should stop if the functor returns false")

	n = 0
	err = testFetcher(&testPager, 5, 2, func(c byte) bool {
		n++
		return true
	})
	assert.Error(t, err, "Should return error if too many pages")
	assert.Equal(t, 5*2, n, "Should iterate over all pages until error")
}
