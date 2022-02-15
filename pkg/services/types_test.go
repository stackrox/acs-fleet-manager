package services

import (
	"testing"

	. "github.com/onsi/gomega"
)

func makeParams(orderByAry []string) map[string][]string {
	params := make(map[string][]string)
	params["orderBy"] = orderByAry
	return params
}

func Test_ValidateOrderBy(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string][]string
		wantErr bool
	}{
		{
			name:    "One Column Asc",
			params:  makeParams([]string{"name asc"}),
			wantErr: false,
		},
		{
			name:    "One Column Desc",
			params:  makeParams([]string{"region desc"}),
			wantErr: false,
		},
		{
			name:    "Multiple Columns Mixed Sorting",
			params:  makeParams([]string{"region desc, name asc, cloud_provider desc"}),
			wantErr: false,
		},
		{
			name:    "Multiple Columns Mixed Sorting with invalid column",
			params:  makeParams([]string{"region desc, name asc, invalid desc"}),
			wantErr: true,
		},
		{
			name:    "Multiple Columns Mixed Sorting with invalid sort",
			params:  makeParams([]string{"region desc, name asc, cloud_provider random"}),
			wantErr: true,
		},
		{
			name:    "Multiple Columns Multiple spaces",
			params:  makeParams([]string{"region    desc  ,    name   asc ,    cloud_provider asc"}),
			wantErr: false,
		},
		{
			name:    "Invalid Column Desc",
			params:  makeParams([]string{"invalid desc"}),
			wantErr: true,
		},
		{
			name:    "No ordering - first",
			params:  makeParams([]string{"region"}),
			wantErr: false,
		},
		{
			name:    "No ordering - mixed",
			params:  makeParams([]string{"region, cloud_provider desc, name"}),
			wantErr: false,
		},
		{
			name:    "No ordering - all valid params",
			params:  makeParams(GetAcceptedOrderByParams()),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			la := NewListArguments(tt.params)
			err := la.Validate()
			if tt.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
