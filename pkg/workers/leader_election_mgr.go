package workers

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"gorm.io/gorm"
)

const (
	mgrRepeatInterval = 15 * time.Second
	leaseRenewTime    = 1 * time.Minute
)

// LeaderElectionManager ...
type LeaderElectionManager struct {
	workers           []Worker
	connectionFactory *db.ConnectionFactory
	tearDown          chan struct{}
	mgrRepeatInterval time.Duration
	leaseRenewTime    time.Duration
	workerGrp         sync.WaitGroup
	forceLeader       bool
}

// leaderLeaseAcquisition a wrapper for a lease and whether it's been acquired/is owned by another worker
type leaderLeaseAcquisition struct {
	// acquired whether the worker successfully acquired/retained the leader lease
	acquired bool
	// currentLease the current lease, it may not necessarily belong to the worker, see acquired
	currentLease *api.LeaderLease
}

// NewLeaderElectionManager ...
func NewLeaderElectionManager(workers []Worker, connectionFactory *db.ConnectionFactory, serverConfig *server.ServerConfig) *LeaderElectionManager {
	if serverConfig.ForceLeader {
		glog.Warningf("LEADER ELECTION HAS BEEN DISABLED FOR TEST ENVIRONMENT")
	}
	return &LeaderElectionManager{
		workers:           workers,
		connectionFactory: connectionFactory,
		mgrRepeatInterval: mgrRepeatInterval,
		leaseRenewTime:    leaseRenewTime,
		forceLeader:       serverConfig.ForceLeader,
	}
}

// Start ...
func (s *LeaderElectionManager) Start() {
	glog.Infoln("Starting LeaderElectionManager")

	s.tearDown = make(chan struct{})
	waitWorkersStart := make(chan struct{})
	go func() {
		// Starts once immediately
		s.startWorkers()
		close(waitWorkersStart) // let Start() to proceed
		ticker := time.NewTicker(s.mgrRepeatInterval)
		for {
			select {
			case <-ticker.C:
				s.startWorkers()
			case <-s.tearDown:
				ticker.Stop()
				for _, worker := range s.workers {
					if worker.IsRunning() {
						glog.Infoln(fmt.Sprintf("Stopping running worker %T [%s]", worker, worker.GetID()))
						worker.Stop()
						s.workerGrp.Done()
					}
				}
				return
			}
		}
	}()

	// wait for all workers started before leave.
	<-waitWorkersStart
}

// Stop impl. Stoppable
// Workers started with Leader Election manager should be stop from here
func (s *LeaderElectionManager) Stop() {
	if s.tearDown == nil {
		return
	}
	select {
	case <-s.tearDown:
		return // already closed/stopped
	default:
		close(s.tearDown) // it is started before, now close/stop
		// wait for all workers are stopped before leaving
		s.workerGrp.Wait()
	}
}

func (s *LeaderElectionManager) startWorkers() {
	for _, worker := range s.workers {
		isLeader := s.isWorkerLeader(worker)
		if isLeader && !worker.IsRunning() {
			glog.Infoln(fmt.Sprintf("Running as the leader and starting worker %T [%s]", worker, worker.GetID()))
			worker.Start()
			s.workerGrp.Add(1) // a new worker is added to the group
		} else if !isLeader && worker.IsRunning() {
			glog.Infoln(fmt.Sprintf("No longer the leader and stopping worker %T [%s]", worker, worker.GetID()))
			worker.Stop()
			s.workerGrp.Done() // a worker is removed from the group
		}
	}
}

func (s *LeaderElectionManager) isWorkerLeader(worker Worker) bool {
	if s.forceLeader {
		return true
	}
	dbConn := s.connectionFactory.New()
	leaderLeaseAcquisition, err := s.acquireLeaderLease(worker.GetID(), worker.GetWorkerType(), dbConn)
	if err != nil {
		// we don't know whether we're the leader or not, set metric to false for now
		// metrics.UpdateLeaderStatusMetric(false)
		glog.Infof("failed to acquire leader lease: %s", err)
		return false
	}

	if !leaderLeaseAcquisition.acquired {
		glog.Infof("not currently leader, skipping reconcile %T [%s]", worker, worker.GetID())
		return false
	}

	return true
}

// acquireLeaderLease attempt to claim the leader role using a provided table and return a leaderLeaseAcquisition
// containing the lease
func (s *LeaderElectionManager) acquireLeaderLease(workerID string, workerType string, dbConn *gorm.DB) (*leaderLeaseAcquisition, error) {
	// read the leader lease, to see whether the worker has an opportunity to:
	// - acquire the lease
	// - extend their lease, if they are the leader, if they are the leader and the lease expiry time is close
	// - continue with their existing lease, if they are the leader and the lease expiry time is far away
	var leaseList api.LeaderLeaseList
	if err := dbConn.Raw("SELECT * FROM leader_leases where deleted_at is null and lease_type = ?  LIMIT 1", workerType).Scan(&leaseList).Error; err != nil {
		return nil, errors.Wrap(err, "failed to retrieve leader leases")
	}

	// we failed to read the current lease, we always expect a single lease to exist, create one so that worker can proceed.
	if len(leaseList) == 0 {
		return nil, errors.Errorf("expected to find a lease entry, found none for :%s", workerType)
	}

	// the lease will be the first entry returned
	lease := leaseList[0]
	// if we get the opportunity to acquire or extend the lease, use this expiry time
	newExpiryTime := time.Now().Add(s.leaseRenewTime)
	// assume we're not the leader by default
	isLeader := false

	// determine if we have an opportunity to acquire or extend the lease (extend if the lease is going to expire in one min)
	if isExpired(lease) || (lease.Leader == workerID && lease.Expires.Before(time.Now().Add(30*time.Second))) {
		// begin a new transaction
		// we must ensure we commit or rollback this transaction to avoid stale transactions being left around
		leaderTx := dbConn.Begin() // starts a new transaction

		// attempt to lock the leader lease
		if err := leaderTx.Raw("SELECT * FROM leader_leases where deleted_at is null and lease_type = ? FOR UPDATE SKIP LOCKED LIMIT 1", workerType).Scan(&leaseList).Error; err != nil {
			leaderTx.Rollback()
			return nil, errors.Wrap(err, "failed to retrieve leader leases")
		}

		/* if length is 0 we missed our opportunity, another worker has taken the lease for update:
		- if we've passed the lease expiry, it could be another worker on this node or another, so fail
		  gracefully
		- if we haven't passed the lease expiry, it's another worker on this node extending the lease, so
		  continue the reconcile
		*/
		if len(leaseList) == 0 && isExpired(lease) {
			glog.Infof("failed to acquire lock on leader lease for update, skipping")
			leaderTx.Rollback()
			return &leaderLeaseAcquisition{
				acquired:     false,
				currentLease: lease,
			}, nil
		}

		// if length is non-zero then we have claimed a lock on the lease entry
		// we have the opportunity to extend or acquire the leader lease
		if len(leaseList) > 0 {
			if err := leaderTx.Model(&lease).Updates(map[string]interface{}{"leader": workerID, "expires": newExpiryTime}).Error; err != nil {
				leaderTx.Rollback()
				return nil, errors.Wrap(err, "failed to update leader lease")
			}
		}

		// if we got to this point we either:
		// - have acquired an expired lease
		// - extended our existing lease
		// - are in our existing lease that doesn't need to be extended
		//
		// we are the leader at this point
		isLeader = true

		// ensure we persist our update by committing the transaction
		leaderTx.Commit()
	} else if lease.Leader == workerID {
		// we didn't have the opportunity to acquire or extend the lease, but we are marked as the leader on the
		// existing, unexpired lease, so continue assuming we're leader
		isLeader = true
	}

	return &leaderLeaseAcquisition{
		acquired:     isLeader,
		currentLease: lease,
	}, nil
}

func isExpired(lease *api.LeaderLease) bool {
	return lease.Leader == "" || time.Now().After(*lease.Expires)
}
