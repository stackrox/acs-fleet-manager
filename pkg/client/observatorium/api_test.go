package observatorium

import (
	"testing"
)

func TestServiceObservatorium_GetMetrics(t *testing.T) {
	type fields struct {
		client *Client
	}

	type args struct {
		metrics   *DinosaurMetrics
		namespace string
		rq        *MetricsReqParams
	}

	obsClientMock, err := NewClientMock(&Configuration{})
	if err != nil {
		t.Fatal("failed to create a mock observatorium client")
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Return metrics successfully for Query result type",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &DinosaurMetrics{},
				namespace: "dinosaur-test",
				rq: &MetricsReqParams{
					ResultType: Query,
				},
			},
			wantErr: false,
		},
		{
			name: "Return metrics successfully for Query result type",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &DinosaurMetrics{},
				namespace: "dinosaur-test",
				rq: &MetricsReqParams{
					ResultType: RangeQuery,
				},
			},
			wantErr: false,
		},
		{
			name: "Return an error if result type is not supported",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &DinosaurMetrics{},
				namespace: "dinosaur-test",
				rq: &MetricsReqParams{
					ResultType: "unsupported",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &ServiceObservatorium{
				client: tt.fields.client,
			}
			if err := obs.GetMetrics(tt.args.metrics, tt.args.namespace, tt.args.rq); (err != nil) != tt.wantErr {
				t.Errorf("ServiceObservatorium.GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.args.metrics == nil {
				t.Errorf("metrics list should not be empty")
			}
		})
	}
}
