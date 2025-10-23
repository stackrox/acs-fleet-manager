package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOIDCIssuers_GetURIs(t *testing.T) {
	tests := []struct {
		name     string
		issuers  *OIDCIssuers
		expected []string
	}{
		{
			name: "returns copy of URIs with multiple elements",
			issuers: &OIDCIssuers{
				URIs: []string{"https://issuer1.example.com", "https://issuer2.example.com", "https://issuer3.example.com"},
			},
			expected: []string{"https://issuer1.example.com", "https://issuer2.example.com", "https://issuer3.example.com"},
		},
		{
			name: "returns copy of URIs with single element",
			issuers: &OIDCIssuers{
				URIs: []string{"https://issuer.example.com"},
			},
			expected: []string{"https://issuer.example.com"},
		},
		{
			name: "returns empty slice when URIs is empty",
			issuers: &OIDCIssuers{
				URIs: []string{},
			},
			expected: []string{},
		},
		{
			name: "returns empty slice when URIs is nil",
			issuers: &OIDCIssuers{
				URIs: nil,
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.issuers.GetURIs()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOIDCIssuers_GetURIs_ReturnsIndependentCopy(t *testing.T) {
	original := &OIDCIssuers{
		URIs: []string{"https://issuer1.example.com", "https://issuer2.example.com"},
	}

	// Get a copy
	urisCopy := original.GetURIs()

	// Modify the copy
	urisCopy[0] = "https://modified.example.com"

	// Verify original is unchanged
	assert.Equal(t, []string{"https://issuer1.example.com", "https://issuer2.example.com"}, original.URIs)
	assert.Len(t, original.URIs, 2)
}
