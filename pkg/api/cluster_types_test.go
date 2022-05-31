package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGetAvailableDinosaurOperatorVersions(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    []CentralOperatorVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available dinosaur operator versions those are returned",
			cluster: func() *Cluster {
				inputDinosaurOperatorVersions := []CentralOperatorVersion{
					CentralOperatorVersion{Version: "v3", Ready: true},
					CentralOperatorVersion{Version: "v6", Ready: false},
					CentralOperatorVersion{Version: "v7", Ready: true},
				}
				inputDinosaurOperatorVersionsJSON, err := json.Marshal(inputDinosaurOperatorVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableDinosaurOperatorVersions: inputDinosaurOperatorVersionsJSON}
				return &res
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "v3", Ready: true},
				CentralOperatorVersion{Version: "v6", Ready: false},
				CentralOperatorVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available dinosaur operator the empty list is returned",
			cluster: func() *Cluster {
				inputDinosaurOperatorVersions := []CentralOperatorVersion{}
				inputDinosaurOperatorVersionsJSON, err := json.Marshal(inputDinosaurOperatorVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableDinosaurOperatorVersions: inputDinosaurOperatorVersionsJSON}
				return &res
			},
			want:    []CentralOperatorVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available dinosaur operator the empty list is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableDinosaurOperatorVersions: nil}
				return &res
			},
			want:    []CentralOperatorVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has an invalid JSON an error is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableDinosaurOperatorVersions: []byte(`"keyone": valueone`)}
				return &res
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableDinosaurOperatorVersions()
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

func TestGetAvailableAndReadyDinosaurOperatorVersions(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    []CentralOperatorVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available dinosaur operator versions those ready returned",
			cluster: func() *Cluster {
				inputDinosaurOperatorVersions := []CentralOperatorVersion{
					CentralOperatorVersion{Version: "v3", Ready: true},
					CentralOperatorVersion{Version: "v6", Ready: false},
					CentralOperatorVersion{Version: "v7", Ready: true},
				}
				inputDinosaurOperatorVersionsJSON, err := json.Marshal(inputDinosaurOperatorVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableDinosaurOperatorVersions: inputDinosaurOperatorVersionsJSON}
				return &res
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "v3", Ready: true},
				CentralOperatorVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available dinosaur operator the empty list is returned",
			cluster: func() *Cluster {
				inputDinosaurOperatorVersions := []CentralOperatorVersion{}
				inputDinosaurOperatorVersionsJSON, err := json.Marshal(inputDinosaurOperatorVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableDinosaurOperatorVersions: inputDinosaurOperatorVersionsJSON}
				return &res
			},
			want:    []CentralOperatorVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available dinosaur operator the empty list is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableDinosaurOperatorVersions: nil}
				return &res
			},
			want:    []CentralOperatorVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has an invalid JSON an error is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableDinosaurOperatorVersions: []byte(`"keyone": valueone`)}
				return &res
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableAndReadyDinosaurOperatorVersions()
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

func TestSetAvailableDinosaurOperatorVersions(t *testing.T) {
	tests := []struct {
		name                          string
		inputDinosaurOperatorVersions []CentralOperatorVersion
		want                          []CentralOperatorVersion
		wantErr                       bool
	}{
		{
			name: "When setting a non empty ordered list of dinosaur operator versions that list is stored as is",
			inputDinosaurOperatorVersions: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.7.0.0-0", Ready: true},
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.6.0.0-0", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.7.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of dinosaur operator versions that list is stored in semver ascending order",
			inputDinosaurOperatorVersions: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.0.0-0", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.2.0.0-0", Ready: true},
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.2.0.0-0", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.3.0.0-0", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of dinosaur operator versions that list is stored in semver ascending order (case 2)",
			inputDinosaurOperatorVersions: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.10.0-3", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.8.0-9", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.2.0.0-0", Ready: true},
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{Version: "dinosaur-operator-v.2.0.0-0", Ready: true},
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.8.0-9", Ready: false},
				CentralOperatorVersion{Version: "dinosaur-operator-v.5.10.0-3", Ready: true},
			},
			wantErr: false,
		},
		{
			name:                          "When setting an empty list of dinosaur operator versions that list is stored as the empty list",
			inputDinosaurOperatorVersions: []CentralOperatorVersion{},
			want:                          []CentralOperatorVersion{},
			wantErr:                       false,
		},
		{
			name:                          "When setting a nil list of dinosaur operator versions that list is stored as the empty list",
			inputDinosaurOperatorVersions: nil,
			want:                          []CentralOperatorVersion{},
			wantErr:                       false,
		},
		{
			name: "Dinosaur versions are stored and in sorted order",
			inputDinosaurOperatorVersions: []CentralOperatorVersion{
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.5.10.0-3",
					Ready:   true,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "2.7.5"},
						CentralVersion{Version: "2.7.3"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.5.8.0-9",
					Ready:   false,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "2.9.4"},
						CentralVersion{Version: "2.2.1"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.2.0.0-0",
					Ready:   true,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "4.5.6"},
						CentralVersion{Version: "1.2.3"},
					},
				},
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.2.0.0-0",
					Ready:   true,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "1.2.3"},
						CentralVersion{Version: "4.5.6"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.5.8.0-9",
					Ready:   false,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "2.2.1"},
						CentralVersion{Version: "2.9.4"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.5.10.0-3",
					Ready:   true,
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "2.7.3"},
						CentralVersion{Version: "2.7.5"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &Cluster{}
			err := cluster.SetAvailableDinosaurOperatorVersions(tt.inputDinosaurOperatorVersions)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				var got []CentralOperatorVersion
				err := json.Unmarshal(cluster.AvailableDinosaurOperatorVersions, &got)
				if err != nil {
					panic(err)
				}

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}

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
				[]CentralOperatorVersion{CentralOperatorVersion{Version: "dinosaur-operator-v.nonsemver243-0"}, CentralOperatorVersion{Version: "dinosaur-operator-v.2.5.6-0"}},
			},
			wantErr: true,
		},
		{
			name: "All different versions are deeply sorted",
			args: args{
				versions: []CentralOperatorVersion{
					CentralOperatorVersion{
						Version: "dinosaur-operator-v.2.7.5-0",
						CentralVersions: []CentralVersion{
							CentralVersion{Version: "1.5.8"},
							CentralVersion{Version: "0.7.1"},
							CentralVersion{Version: "1.5.1"},
						},
					},
					CentralOperatorVersion{
						Version: "dinosaur-operator-v.2.7.3-0",
						CentralVersions: []CentralVersion{
							CentralVersion{Version: "1.0.0"},
							CentralVersion{Version: "2.0.0"},
							CentralVersion{Version: "5.0.0"},
						},
					},
					CentralOperatorVersion{
						Version: "dinosaur-operator-v.2.5.2-0",
						CentralVersions: []CentralVersion{
							CentralVersion{Version: "2.6.1"},
							CentralVersion{Version: "5.7.2"},
							CentralVersion{Version: "2.3.5"},
						},
					},
				},
			},
			want: []CentralOperatorVersion{
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.2.5.2-0",
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "2.3.5"},
						CentralVersion{Version: "2.6.1"},
						CentralVersion{Version: "5.7.2"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.2.7.3-0",
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "1.0.0"},
						CentralVersion{Version: "2.0.0"},
						CentralVersion{Version: "5.0.0"},
					},
				},
				CentralOperatorVersion{
					Version: "dinosaur-operator-v.2.7.5-0",
					CentralVersions: []CentralVersion{
						CentralVersion{Version: "0.7.1"},
						CentralVersion{Version: "1.5.1"},
						CentralVersion{Version: "1.5.8"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DinosaurOperatorVersionsDeepSort(tt.args.versions)
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
