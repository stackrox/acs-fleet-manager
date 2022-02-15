package ocm

import (
	"fmt"
	"strings"
	"testing"
)

func Test_idGenerator_Generate(t *testing.T) {
	type fields struct {
		prefix string
	}
	tests := []struct {
		name       string
		fields     fields
		validateFn func(id string) error
	}{
		{
			name: "valid prefix",
			fields: fields{
				prefix: "mk-",
			},
			validateFn: func(id string) error {
				if !strings.HasPrefix(id, "mk-") {
					return fmt.Errorf("expected id to have prefix %s, got = %s", "mk-", id)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := idGenerator{
				prefix: tt.fields.prefix,
			}
			got := i.Generate()
			if err := tt.validateFn(got); err != nil {
				t.Errorf("Generate() = %v", err.Error())
			}
			if len(got) > MaxClusterNameLength {
				t.Errorf("Generated ID length should not exceed 15 chars: %v", got)
			}
		})
	}
}
