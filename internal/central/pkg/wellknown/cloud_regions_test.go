package wellknown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCloudRegionDisplayName(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		regionName   string
		want         string
	}{
		{
			name:         "known provider and region",
			providerName: "aws",
			regionName:   "us-east-1",
			want:         "US East (N. Virginia)",
		}, {
			name:         "known provider and unknown region",
			providerName: "aws",
			regionName:   "foobar",
			want:         "foobar",
		}, {
			name:         "unknown provider and unknown region",
			providerName: "unknown-provider",
			regionName:   "foobar",
			want:         "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetCloudRegionDisplayName(tt.providerName, tt.regionName))
		})
	}
}
