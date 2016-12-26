package common

import (
	"time"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/libredis/structures"
)

type CloneRequest struct {
	Origin   string
	Clone    string
	Role     string
	Reconfig bool
	Promote  bool
}

type FailoverRequest struct {
	Podname   string
	ReturnNew bool
}

type AddSlaveRequest struct {
	Podname      string
	SlaveAddress string
	SlavePort    int
	SlaveAuth    string
}

type MonitorRequest struct {
	Podname       string
	MasterAddress string
	AuthToken     string
	MasterPort    int
	Quorum        int
}

type RedisNode struct {
	Name                      string
	Address                   string
	Port                      int
	MaxMemory                 int
	LastStart                 time.Time
	Info                      structures.RedisInfoAll
	Slaves                    []*RedisNode
	AOFEnabled                bool
	SaveEnabled               bool
	PercentUsed               float64
	MemoryUseWarn             bool
	MemoryUseCritical         bool
	HasEnoughMemoryForMaster  bool
	Auth                      string
	LastUpdate                time.Time
	LastUpdateValid           bool
	LastUpdateDelay           time.Duration
	HasValidAuth              bool
	Connected                 bool
	LatencyHistory            client.LatencyHistory
	LatencyHistoryFastCommand client.LatencyHistory
	LatencyThreshold          int
	LatencyDoctor             string
	LatencyMonitoringEnabled  bool
	SlowLogThreshold          int64
	SlowLogLength             int64
	SlowLogRecords            []*client.SlowLog
}

type RedisPod struct {
	Name                  string
	Info                  structures.MasterInfo
	Slaves                []structures.InfoSlaves
	Master                *RedisNode
	SentinelCount         int
	ActiveSentinelCount   int
	ReportedSentinelCount int
	AuthToken             string
	ValidAuth             bool
	ValidMasterConnection bool
	NeededSentinels       int
	MissingSentinels      bool
	TooManySentinels      bool
	HasInfo               bool
	NeedsReset            bool
	HasValidSlaves        bool
}
