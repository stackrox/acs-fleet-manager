package db

import "time"

// DinosaurAdditionalLeasesExpireTime Set new additional leases expire time to a minute later from now so that the old "dinosaur" leases finishes
// its execution before the new jobs kicks in.
// Not used in leader election anymore, kept for database migrations
var DinosaurAdditionalLeasesExpireTime = time.Now().Add(1 * time.Minute)
