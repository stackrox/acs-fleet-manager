package dbapi

import (
	"reflect"
	"testing"
)

func TestDataPlaneDinosaurstatus_GetReadyCondition(t *testing.T) {
	tests := []struct {
		name         string
		wantCondType string
		wantOK       bool
		statusConds  []DataPlaneDinosaurStatusCondition
	}{
		{
			name:   "When no ready condition exists ok is false",
			wantOK: false,
			statusConds: []DataPlaneDinosaurStatusCondition{
				DataPlaneDinosaurStatusCondition{Type: "CondType1"},
				DataPlaneDinosaurStatusCondition{Type: "CondType2"},
			},
		},
		{
			name:         "When ready condition exists ok is true",
			wantOK:       true,
			wantCondType: "Ready",
			statusConds: []DataPlaneDinosaurStatusCondition{
				DataPlaneDinosaurStatusCondition{Type: "CondType1"},
				DataPlaneDinosaurStatusCondition{Type: "Ready"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := DataPlaneDinosaurStatus{Conditions: tt.statusConds}
			res, ok := input.GetReadyCondition()
			if !reflect.DeepEqual(ok, tt.wantOK) {
				t.Errorf("want: %v got: %v", tt.wantOK, ok)
			}
			if !reflect.DeepEqual(res.Type, tt.wantCondType) {
				t.Errorf("want: %v got: %v", tt.wantCondType, res.Type)
			}
		})
	}
}
