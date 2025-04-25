package types

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_CloudProviderRegionInfoList_Merge(t *testing.T) {
	type fields struct {
		target *CloudProviderRegionInfoList
	}

	type args struct {
		source *CloudProviderRegionInfoList
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   []CloudProviderRegionInfo
	}{
		{
			name: "total items in target should remain empty when targe and source are empty",
			fields: fields{
				target: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{},
				},
			},
			args: args{
				source: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{},
				},
			},
			want: []CloudProviderRegionInfo{},
		},
		{
			name: "total items in target should remain the same when source is empty",
			fields: fields{
				target: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{
						{
							ID: "some-id",
						},
					},
				},
			},
			args: args{
				source: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{},
				},
			},
			want: []CloudProviderRegionInfo{
				{
					ID: "some-id",
				},
			},
		},
		{
			name: "copy source items into target when target is empty",
			fields: fields{
				target: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{},
				},
			},
			args: args{
				source: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{
						{
							ID: "some-id",
						},
					},
				},
			},
			want: []CloudProviderRegionInfo{
				{
					ID: "some-id",
				},
			},
		},
		{
			name: "merge source items into target",
			fields: fields{
				target: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{
						{
							ID: "some-id-2",
						},
						{
							ID: "some-id",
						},
					},
				},
			},
			args: args{
				source: &CloudProviderRegionInfoList{
					Items: []CloudProviderRegionInfo{
						{
							ID: "some-id",
						},
						{
							ID: "some-id-3",
						},
					},
				},
			},
			want: []CloudProviderRegionInfo{
				{
					ID: "some-id-2",
				},
				{
					ID: "some-id",
				},
				{
					ID: "some-id-3",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			tt.fields.target.Merge(tt.args.source)
			Expect(tt.want).To(Equal(tt.fields.target.Items))
		})
	}
}
