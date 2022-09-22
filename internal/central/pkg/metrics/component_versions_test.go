package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
)

func TestVersionsMetrics_Collect(t *testing.T) {
	type fields struct {
		dinosaurService services.CentralService
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
			fields: fields{dinosaurService: &services.CentralServiceMock{
				ListComponentVersionsFunc: func() ([]services.CentralComponentVersions, error) {
					return []services.CentralComponentVersions{
						{
							ID:                            "1",
							ClusterID:                     "cluster1",
							DesiredCentralOperatorVersion: "1.0.1",
							ActualCentralOperatorVersion:  "1.0.0",
							CentralOperatorUpgrading:      true,
							DesiredCentralVersion:         "1.0.1",
							ActualCentralVersion:          "1.0.0",
							CentralUpgrading:              false,
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
