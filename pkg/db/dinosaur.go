package db

import "time"

// CentralAdditionalLeasesExpireTime Set new additional leases expire time to a minute later from now so that the old "central" leases finishes
// its execution before the new jobs kicks in.
var CentralAdditionalLeasesExpireTime = time.Now().Add(1 * time.Minute)
