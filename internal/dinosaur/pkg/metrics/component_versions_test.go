package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
)

func TestVersionsMetrics_Collect(t *testing.T) {
	type fields struct {
		dinosaurService services.DinosaurService
	}

	type args struct {
		ch chan<- prometheus.Metric
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "will generate metrics",
			fields: fields{dinosaurService: &services.DinosaurServiceMock{
				ListComponentVersionsFunc: func() ([]services.DinosaurComponentVersions, error) {
					return []services.DinosaurComponentVersions{
						{
							ID:                             "1",
							ClusterID:                      "cluster1",
							DesiredDinosaurOperatorVersion: "1.0.1",
							ActualDinosaurOperatorVersion:  "1.0.0",
							DinosaurOperatorUpgrading:      true,
							DesiredDinosaurVersion:         "1.0.1",
							ActualDinosaurVersion:          "1.0.0",
							DinosaurUpgrading:              false,
						},
					}, nil
				},
			}},
			args: args{ch: make(chan prometheus.Metric, 100)},
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newVersionMetrics(tt.fields.dinosaurService)
			ch := tt.args.ch
			m.Collect(ch)
			if len(ch) != tt.want {
				t.Errorf("expect to have %d metrics but got %d", tt.want, len(ch))
			}
		})
	}
}
