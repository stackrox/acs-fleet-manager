// Package dbapi ...
package dbapi

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestNullTimeToTimePtr(t *testing.T) {
	testTime := time.Unix(123456, 0)

	tests := map[string]struct {
		nullTime sql.NullTime
		want     *time.Time
	}{
		"nil": {
			nullTime: sql.NullTime{},
			want:     nil,
		},
		"not nil": {
			nullTime: sql.NullTime{Time: testTime, Valid: true},
			want:     &testTime,
		},
		"invalid": {
			nullTime: sql.NullTime{Time: testTime, Valid: false},
			want:     nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := NullTimeToTimePtr(test.nullTime)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestTimePtrToNullTime(t *testing.T) {
	testTime := time.Unix(123456, 0)

	tests := map[string]struct {
		nullTime *time.Time
		want     sql.NullTime
	}{
		"nil": {
			nullTime: nil,
			want:     sql.NullTime{},
		},
		"not nil": {
			nullTime: &testTime,
			want:     sql.NullTime{Time: testTime, Valid: true},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := TimePtrToNullTime(test.nullTime)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestCentralRequestNoPointers(t *testing.T) {
	assert.True(t, db.IsStructWithoutPointers(&CentralRequest{}))
}
