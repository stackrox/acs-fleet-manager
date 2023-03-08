package operator

import (
	"context"
	"reflect"
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestACSOperatorManager_Upgrade(t *testing.T) {
	type fields struct {
		client         client.Client
		resourcesChart *chart.Chart
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &ACSOperatorManager{
				client:         tt.fields.client,
				resourcesChart: tt.fields.resourcesChart,
			}
			if err := u.Upgrade(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Upgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewACSOperatorManager(t *testing.T) {
	type args struct {
		k8sClient client.Client
	}
	tests := []struct {
		name string
		args args
		want *ACSOperatorManager
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewACSOperatorManager(tt.args.k8sClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewACSOperatorManager() = %v, want %v", got, tt.want)
			}
		})
	}
}
