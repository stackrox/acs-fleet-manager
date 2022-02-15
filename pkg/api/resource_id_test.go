package api

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation"
)

func TestResourceIDGeneration(t *testing.T) {
	id := NewID()
	errors := validation.IsDNS1123Label(id)
	if len(errors) > 0 {
		for _, e := range errors {
			t.Fatalf(e)
		}
	}
}
