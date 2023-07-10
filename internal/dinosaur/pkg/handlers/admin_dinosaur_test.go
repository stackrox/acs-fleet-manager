package handlers

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_updateCentralRequest(t *testing.T) {
	tests := []struct {
		name               string
		state              string
		patch              string
		wantCentral        string
		wantScanner        string
		wantForceReconcile string
		wantErr            func(t *testing.T, err error)
	}{
		{
			name:        "empty update on empty central should have no effect",
			state:       `{}`,
			patch:       `{}`,
			wantCentral: `{"resources":{}}`,
		}, {
			name:        "empty update on defined central should have no effect",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{}`,
			wantCentral: `{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}`,
		}, {
			name:        "replacing central limits",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":"2","memory":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"2","memory":"2"},"requests":{"cpu":"1","memory":"1"}}}`,
		}, {
			name:        "replacing central limits when requests are not set",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":"2","memory":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"2","memory":"2"}}}`,
		}, {
			name:        "replacing central CPU limits",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"2","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}`,
		}, {
			name:        "unsetting central CPU limits",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":null}}}}`,
			wantCentral: `{"resources":{"limits":{"memory":"1"},"requests":{"cpu":"1","memory":"1"}}}`,
		}, {
			name:        "unsetting central limits",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":null}}}`,
			wantCentral: `{"resources":{"requests":{"cpu":"1","memory":"1"}}}`,
		}, {
			name:        "unsetting central resources",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":null}}`,
			wantCentral: `{"resources":{}}`,
		}, {
			name:        "unsetting central altogether",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":null}`,
			wantCentral: `{"resources":{}}`,
		}, {
			name:        "replacing central CPU limits when memory is not set",
			state:       `{"central":{"resources":{"limits":{"cpu":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"2"}}}`,
		}, {
			name:        "replacing central CPU limits when CPU is not set",
			state:       `{"central":{"resources":{"limits":{"memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"limits":{"cpu":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"2","memory":"1"}}}`,
		}, {
			name:        "update with existing central",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"requests":{"cpu":"2","memory":"2"}}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"2","memory":"2"}}}`,
		}, {
			name:        "should ignore unknown fields",
			state:       `{"central":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
			patch:       `{"central":{"resources":{"requests":{"cpu":"2","memory":"2"}},"unknown":{"foo":"bar"}}}`,
			wantCentral: `{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"2","memory":"2"}}}`,
		}, {
			name:        "empty update on empty scanner should have no effect",
			state:       `{"scanner":{}}`,
			patch:       `{}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "empty update on defined scanner should have no effect",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}},"scaling":{"autoScaling":"Enabled","replicas":1,"minReplicas":1,"maxReplicas":1}},"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}},"scaling":{"autoScaling":"Enabled","replicas":1,"minReplicas":1,"maxReplicas":1}},"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
		}, {
			name:        "replacing scanner analyzer resources",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"2","memory":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"2","memory":"2"},"requests":{"cpu":"1","memory":"1"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer CPU resources",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"2","memory":"1"},"requests":{"cpu":"1","memory":"1"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer memory resources",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"limits":{"memory":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"2"},"requests":{"cpu":"1","memory":"1"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer requests",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"requests":{"cpu":"2","memory":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"2","memory":"2"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer CPU requests",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"requests":{"cpu":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"2","memory":"1"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer scaling",
			state:       `{"scanner":{"analyzer":{"scaling":{"autoScaling":"Enabled","replicas":1,"minReplicas":1,"maxReplicas":1}}}}`,
			patch:       `{"scanner":{"analyzer":{"scaling":{"autoScaling":"Disabled","replicas":2,"minReplicas":2,"maxReplicas":2}}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{"autoScaling":"Disabled","replicas":2,"minReplicas":2,"maxReplicas":2}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner analyzer scaling replicas",
			state:       `{"scanner":{"analyzer":{"scaling":{"autoScaling":"Enabled","replicas":1,"minReplicas":1,"maxReplicas":1}}}}`,
			patch:       `{"scanner":{"analyzer":{"scaling":{"replicas":2}}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{"autoScaling":"Enabled","replicas":2,"minReplicas":1,"maxReplicas":1}},"db":{"resources":{}}}`,
		}, {
			name:        "replacing scanner db",
			state:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"2","memory":"2"},"requests":{"cpu":"2","memory":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{"limits":{"cpu":"2","memory":"2"},"requests":{"cpu":"2","memory":"2"}}}}`,
		}, {
			name:        "replacing scanner db CPU request",
			state:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{"limits":{"cpu":"2","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}`,
		}, {
			name:        "replacing scanner db requests",
			state:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"db":{"resources":{"requests":{"cpu":"2","memory":"2"}}}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"2","memory":"2"}}}}`,
		}, {
			name:        "unset scanner analyzer resources",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":null}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "unset scanner analyzer scaling",
			state:       `{"scanner":{"analyzer":{"scaling":{"autoScaling":"Enabled","replicas":1,"minReplicas":1,"maxReplicas":1}}}}`,
			patch:       `{"scanner":{"analyzer":{"scaling":null}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "unset scanner db resources",
			state:       `{"scanner":{"db":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"db":{"resources":null}}}`,
			wantScanner: `{"analyzer":{"resources":{},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:        "unset scanner analyzer resources limits",
			state:       `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch:       `{"scanner":{"analyzer":{"resources":{"limits":null}}}}`,
			wantScanner: `{"analyzer":{"resources":{"requests":{"cpu":"1","memory":"1"}},"scaling":{}},"db":{"resources":{}}}`,
		}, {
			name:               "replacing forceReconcile",
			state:              `{"forceReconcile":"foo"}`,
			patch:              `{"forceReconcile":"bar"}`,
			wantForceReconcile: "bar",
		}, {
			name:               "unsetting forceReconcile",
			state:              `{"forceReconcile":"foo"}`,
			patch:              `{"forceReconcile":null}`,
			wantForceReconcile: "",
		}, {
			name:               "setting forceReconcile to empty string",
			state:              `{"forceReconcile":"foo"}`,
			patch:              `{"forceReconcile":""}`,
			wantForceReconcile: "",
		}, {
			name:  "should fail if the patch is invalid json",
			state: `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch: `foo`,
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		}, {
			name:  "should fail if the resource name is not cpu or memory",
			state: `{"scanner":{"analyzer":{"resources":{"limits":{"cpu":"1","memory":"1"},"requests":{"cpu":"1","memory":"1"}}}}}`,
			patch: `{"scanner":{"analyzer":{"resources":{"limits":{"foo":"1"}}}}}`,
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var request *dbapi.CentralRequest
			require.NoError(t, json.Unmarshal([]byte(tt.state), &request))
			err := updateCentralRequest(request, []byte(tt.patch))
			if tt.wantErr != nil {
				tt.wantErr(t, err)
			} else {
				require.NoError(t, err)
				if len(tt.wantScanner) > 0 {
					assert.Equal(t, string(tt.wantScanner), string(request.Scanner))
				}
				if len(tt.wantCentral) > 0 {
					assert.Equal(t, string(tt.wantCentral), string(request.Central))
				}
			}
		})
	}
}
