package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestQuantityToStringConverter(t *testing.T) {
	assert.Equal(t, qtyAsString(resource.Quantity{}), "")
	assert.Equal(t, qtyAsString(resource.MustParse("1m")), "1m")
	assert.Equal(t, qtyAsString(resource.MustParse("2000G")), "2T")
}
