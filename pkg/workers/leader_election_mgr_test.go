package workers

import (
	"errors"
	"strings"
	"testing"
	"time"

	mocket "github.com/selvatico/go-mocket"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"gorm.io/gorm"
)

func TestLeaderElectionManager_acquireLeaderLease(t *testing.T) {
	type args struct {
		workerId   string
		workerType string
		dbConn     *gorm.DB
	}
	type errorCheck struct {
		wantErr     bool
		expectedErr string
	}

	tests := []struct {
		name       string
		args       args
		wantFn     func(*leaderLeaseAcquisition) error
		errorCheck errorCheck
		setupFn    func()
	}{
		{
			name: "failure listing any lease record in lease table",
			args: args{
				workerId:   "000-000",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: nil,
			errorCheck: errorCheck{
				wantErr:     true,
				expectedErr: "expected to find a lease entry",
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "failure listing leases table results in error",
			args: args{
				workerId:   "000-001",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: nil,
			errorCheck: errorCheck{
				wantErr: true,
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successfully acquired lease and elected  as the leader",
			args: args{
				workerId:   "000-002",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if !acquisition.acquired {
					return errors.New("expected lease acquisition succeeded.")
				}
				if acquisition.currentLease.Leader != "000-002" {
					return errors.New("unexpected leader acquired.")
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-002",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT * FROM leader_leases").WithReply([]map[string]interface{}{mockEntry})
			},
		},
		{
			name: "valid lease within a minute of expiry extends lease",
			args: args{
				workerId:   "000-003",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if !acquisition.acquired {
					return errors.New("expected lease acquisition succeeded.")
				}
				if !acquisition.currentLease.Expires.After(time.Now().Add(time.Second * 30)) {
					return errors.New("expected lease to be extended")
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-003",
					"id":      "leader-lease-id",
					"expires": time.Now().Add(time.Second * 30),
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1 FOR UPDATE`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().
					WithQuery(`UPDATE "leader_leases" SET "expires"=$1,"leader"=$2,"updated_at"=$3 WHERE "leader_leases"."deleted_at" IS NULL AND "id" = $4`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "valid lease for another worker not to be acquired",
			args: args{
				workerId:   "000-004",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if acquisition.acquired {
					return errors.New("expected lease acquisition to not be acquired")
				}
				if acquisition.currentLease.Leader != "otherLeader" {
					return errors.New("unexpected lease leader: " + acquisition.currentLease.Leader)
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "otherLeader",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			s := &LeaderElectionManager{leaseRenewTime: 3 * time.Minute}
			got, err := s.acquireLeaderLease(tt.args.workerId, tt.args.workerType, tt.args.dbConn)
			if (err != nil) != tt.errorCheck.wantErr {
				t.Errorf("acquireLeaderLease() error = %v, wantErr %v", err, tt.errorCheck.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errorCheck.expectedErr) {
				t.Errorf("Expecting error '%s' but got '%s'", tt.errorCheck.expectedErr, err.Error())
			}
			if tt.wantFn == nil {
				return
			}
			if err := tt.wantFn(got); err != nil {
				t.Errorf("acquireLeaderLease() got = %v, want %v", got, err.Error())
			}
		})
	}
}
