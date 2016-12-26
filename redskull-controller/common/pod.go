package common

import (
	"log"
	"strings"
)

// HasQuorum checks to see if the pod has Quorum.
func (rp *RedisPod) HasQuorum() bool {
	return rp.SentinelCount >= rp.Info.Quorum
}

// CanFailover tests failover conditions to determine if a failover call would
// succeed
func (rp *RedisPod) CanFailover() bool {
	if rp.AuthToken == "" {
		log.Printf("%s has no valid auth, so considered unable to failover", rp.Name)
		return false
	}
	promotable_slaves := 0
	if rp.Master == nil {
		master, err := LoadNodeFromHostPort(rp.Info.IP, rp.Info.Port, rp.AuthToken)
		if err != nil {
			log.Printf("Unable to load %s. Err: '%s'", rp.Name, err)
			if strings.Contains(err.Error(), "invalid password") {
				rp.ValidAuth = false
				master.HasValidAuth = false
			} else {
				rp.ValidMasterConnection = false
			}
			return false
		}
		rp.ValidAuth = true
		rp.ValidMasterConnection = true
		rp.Master = master
	}
	if !rp.Master.HasValidAuth {
		return false
	}
	if !rp.Master.LastUpdateValid {
		rp.HasInfo = false
		log.Printf("Pod %s has no valid update", rp.Name)
	} else {
		rp.HasInfo = true
		for _, slave := range rp.Master.Slaves {
			if slave.Info.Server.Version == "" {
				log.Print("slave had no info stored, skipping")
				continue
			}
			if slave.Info.Replication.SlavePriority > 0 {
				rp.HasValidSlaves = true
				promotable_slaves++
			}
		}
	}
	if promotable_slaves == 0 {
		rp.HasValidSlaves = false
		return false
	} else {
		rp.HasValidSlaves = true
	}
	if !rp.HasQuorum() {
		return false
	}
	return true
}

// SlavesHaveEnoughMemory checks all slaves for their maximum memory to
// validate they match or beter the master
func (rp *RedisPod) SlavesHaveEnoughMemory() bool {
	ok := true
	// This should filter out slaves which have a slave priority of 0
	if rp.Master == nil {
		return false
	}
	for _, node := range rp.Master.Slaves {
		if node == nil {
			log.Print("Node is nil!")
			continue
		}
		if node.MaxMemory < rp.Master.MaxMemory {
			node.HasEnoughMemoryForMaster = false
			ok = false
		}
	}
	return ok
}

// HasErrors checks various error conditions and returns t/f
// TODO: Some of these are better categorized as warnings and this should be
// split into a pair of functions: one for errors and one for warning.
// This will require additional work to incorporate the HasWarnings concept
// through the system.
func (rp *RedisPod) HasErrors() bool {
	rp.NeededSentinels = rp.Info.Quorum + 1
	rp.ReportedSentinelCount = rp.Info.NumOtherSentinels
	hasErrors := false
	if rp.Master != nil {
		rp.Master.LastUpdateValid = false
		rp.Master.UpdateData()
	}
	if rp.Info.NumOtherSentinels > 0 {
		rp.ReportedSentinelCount++
	}
	if rp.NeededSentinels > rp.SentinelCount {
		rp.MissingSentinels = true
		return true
	}
	if rp.Info.NumOtherSentinels+1 > rp.NeededSentinels {
		rp.NeedsReset = true
		return true
	}
	if rp.ReportedSentinelCount >= (rp.Info.Quorum * 2) {
		rp.TooManySentinels = true
		return true
	}
	if !rp.CanFailover() {
		return true
	}
	if !rp.SlavesHaveEnoughMemory() {
		return true
	}
	return hasErrors
}
