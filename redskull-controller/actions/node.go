package actions

import (
	"time"

	"github.com/therealbill/redskull/redskull-controller/common"
)

/*
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
*/

var NodeRefreshInterval float64
var NodesMap map[string]*common.RedisNode
var DialTimeout time.Duration = 900 * time.Millisecond

func init() {
	NodesMap = make(map[string]*common.RedisNode)
}
