package api

import (
	"reflect"
	"testing"
)

func Test_buildAwareSemanticVersioningCompare(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}

	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "when v1 is greater than v2 1 is returned",
			args: args{
				v1: "1.0.1",
				v2: "1.0.0",
			},
			want: 1,
		},
		{
			name: "when v2 is greater than v1 -1 is returned",
			args: args{
				v1: "1.0.0",
				v2: "1.0.1",
			},
			want: -1,
		},
		{
			name: "when v1 is equal to v2 0 is returned",
			args: args{
				v1: "1.0.0",
				v2: "1.0.0",
			},
			want: 0,
		},
		{
			name: "comparison works with the 'v' prefix is supported",
			args: args{
				v1: "v1.2.0",
				v2: "v2.1.0",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "comparing versions without patch level version number works",
			args: args{
				v1: "1.1",
				v2: "1.3",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "comparison with pre-release suffix is supported",
			args: args{
				v1: "1.0.0-rc.1+build2",
				v2: "1.0.0-rc.2+build1",
			},
			want:    -1,
			wantErr: false,
		},
		{
			name: "When two versions have same x.y.z but different build metadata a lexicographical sort of the metadata is performed",
			args: args{
				v1: "1.0.0+buildexample2.2.4",
				v2: "1.0.0+buildexample2.2.1",
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "When two versions have same x.y.z and same pre-release version but a different build metadata a lexicographical sort of the metadata is performed",
			args: args{
				v1: "1.0.0-rc.1+buildexample2",
				v2: "1.0.0-rc.1+buildexample1",
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := buildAwareSemanticVersioningCompare(tt.args.v1, tt.args.v2)
			gotErr := err != nil
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("wantErr: %v got: %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("want: %v got: %v", tt.want, res)
			}
		})
	}

}
