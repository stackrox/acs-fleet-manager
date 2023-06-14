package api

import (
	"reflect"
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name                          string
		inputDinosaurOperatorVersion1 CentralOperatorVersion
		inputDinosaurOperatorVersion2 CentralOperatorVersion
		want                          int
		wantErr                       bool
	}{
		{
			name:                          "When inputDinosaurOperatorVersion1 is smaller than inputDinosaurOperatorVersion2 -1 is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: false},
			want:                          -1,
			wantErr:                       false,
		},
		{
			name:                          "When inputDinosaurOperatorVersion1 is equal than inputDinosaurOperatorVersion2 0 is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: false},
			want:                          0,
			wantErr:                       false,
		},
		{
			name:                          "When inputDinosaurOperatorVersion1 is bigger than inputDinosaurOperatorVersion2 1 is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: false},
			want:                          1,
			wantErr:                       false,
		},
		{
			name:                          "Check that semver-level comparison is performed",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6.3.10-6", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.6.3.8-9", Ready: false},
			want:                          1,
			wantErr:                       false,
		},
		{
			name:                          "When inputDinosaurOperatorVersion1 is empty an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: false},
			wantErr:                       true,
		},
		{
			name:                          "When inputDinosaurOperatorVersion2 is empty an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "", Ready: false},
			wantErr:                       true,
		},
		{
			name:                          "When inputDinosaurOperatorVersion1 has an invalid semver version format an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6invalid.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.7.0.0-0", Ready: false},
			wantErr:                       true,
		},
		{
			name:                          "When inputDinosaurOperatorVersion1 has an invalid expected format an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.7.0.0-0", Ready: false},
			wantErr:                       true,
		},
		{
			name:                          "When inputDinosaurOperatorVersion2 has an invalid semver version format an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.6invalid.0.0-0", Ready: false},
			wantErr:                       true,
		},
		{
			name:                          "When inputDinosaurOperatorVersion2 has an invalid expected format an error is returned",
			inputDinosaurOperatorVersion1: CentralOperatorVersion{Version: "dinosaur-operator-v.7.0.0-0", Ready: true},
			inputDinosaurOperatorVersion2: CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0", Ready: true},
			wantErr:                       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.inputDinosaurOperatorVersion1.Compare(tt.inputDinosaurOperatorVersion2)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}

func Test_DinosaurOperatorVersionsDeepSort(t *testing.T) {
	type args struct {
		versions []CentralOperatorVersion
	}

	tests := []struct {
		name    string
		args    args
		cluster func() *Cluster
		want    []CentralOperatorVersion
		wantErr bool
	}{
		{
			name: "When versions to sort is empty result is empty",
			args: args{
				versions: []CentralOperatorVersion{},
			},
			want: []CentralOperatorVersion{},
		},
		{
			name: "When versions to sort is nil result is nil",
			args: args{
				versions: nil,
			},
			want: nil,
		},
		{
			name: "When one of the dinosaur operator versions does not follow semver an error is returned",
			args: args{
				[]CentralOperatorVersion{{Version: "dinosaur-operator-v.nonsemver243-0"}, {Version: "dinosaur-operator-v.2.5.6-0"}},
			},
			wantErr: true,
		},
		{
			name: "All different versions are deeply sorted",
			args: args{
				versions: []CentralOperatorVersion{
					{
						Version: "dinosaur-operator-v.2.7.5-0",
						CentralVersions: []CentralVersion{
							{Version: "1.5.8"},
							{Version: "0.7.1"},
							{Version: "1.5.1"},
						},
					},
					{
						Version: "dinosaur-operator-v.2.7.3-0",
						CentralVersions: []CentralVersion{
							{Version: "1.0.0"},
							{Version: "2.0.0"},
							{Version: "5.0.0"},
						},
					},
					{
						Version: "dinosaur-operator-v.2.5.2-0",
						CentralVersions: []CentralVersion{
							{Version: "2.6.1"},
							{Version: "5.7.2"},
							{Version: "2.3.5"},
						},
					},
				},
			},
			want: []CentralOperatorVersion{
				{
					Version: "dinosaur-operator-v.2.5.2-0",
					CentralVersions: []CentralVersion{
						{Version: "2.3.5"},
						{Version: "2.6.1"},
						{Version: "5.7.2"},
					},
				},
				{
					Version: "dinosaur-operator-v.2.7.3-0",
					CentralVersions: []CentralVersion{
						{Version: "1.0.0"},
						{Version: "2.0.0"},
						{Version: "5.0.0"},
					},
				},
				{
					Version: "dinosaur-operator-v.2.7.5-0",
					CentralVersions: []CentralVersion{
						{Version: "0.7.1"},
						{Version: "1.5.1"},
						{Version: "1.5.8"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CentralOperatorVersionsDeepSort(tt.args.versions)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}

func TestCompareSemanticVersionsMajorAndMinor(t *testing.T) {
	tests := []struct {
		name    string
		current string
		desired string
		want    int
		wantErr bool
	}{
		{
			name:    "When desired major is smaller than current major, 1 is returned",
			current: "3.6.0",
			desired: "2.6.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "When desired major is greater than current major -1, is returned",
			current: "2.7.0",
			desired: "3.7.0",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is greater than current minor, -1 is returned",
			current: "2.7.0",
			desired: "2.8.0",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is smaller than current minor, 1 is returned",
			current: "2.8.0",
			desired: "2.7.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is equal to current minor, 0 is returned",
			current: "2.7.0",
			desired: "2.7.0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is equal to current patch, 0 is returned",
			current: "2.7.0",
			desired: "2.7.0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is greater than current patch, 0 is returned",
			current: "2.7.0",
			desired: "2.7.1",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is smaller than current patch, 0 is returned",
			current: "2.7.2",
			desired: "2.7.1",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When current is empty an error is returned",
			current: "",
			desired: "2.7.1",
			wantErr: true,
		},
		{
			name:    "When desired is empty an error is returned",
			current: "2.7.1",
			desired: "",
			wantErr: true,
		},
		{
			name:    "When current has an invalid semver version format an error is returned",
			current: "2invalid.6.0",
			desired: "2.7.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareSemanticVersionsMajorAndMinor(tt.current, tt.desired)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}
