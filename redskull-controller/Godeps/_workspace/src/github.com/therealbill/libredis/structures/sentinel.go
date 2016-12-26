package structures

import "time"

type Sentinel struct {
	Address        string
	LastConnection time.Time
}

type SentinelList []Sentinel

// MasterAddress is a small struct to provide connection information for a
// Master as returned from get-master-addr-by-name
type MasterAddress struct {
	Host string
	Port int
}

// MasterInfo is a struct providing the information available from sentinel
// about a given master (aka pod)
// The way this works is you tag the field with the name redis returns
// and reflect is used in the methods which return this structure to populate
// it with the data from Redis
//
// Note this means it will nee dto be updated when new fields are added in
// sentinel. Fortunately this appears to be rare.
//
// Currently the list is:
// 'pending-commands'
// 'ip'
// 'down-after-milliseconds'
// 'role-reported'
// 'runid'
// 'port'
// 'last-ok-ping-reply'
// 'last-ping-sent'
// 'failover-timeout'
// 'config-epoch'
// 'quorum'
// 'role-reported-time'
// 'last-ping-reply'
// 'name'
// 'parallel-syncs'
// 'info-refresh'
// 'flags'
// 'num-slaves'
// 'num-other-sentinels'
type MasterInfo struct {
	Name                  string `redis:"name"`
	Port                  int    `redis:"port"`
	NumSlaves             int    `redis:"num-slaves"`
	Quorum                int    `redis:"quorum"`
	NumOtherSentinels     int    `redis:"num-other-sentinels"`
	ParallelSyncs         int    `redis:"parallel-syncs"`
	Runid                 string `redis:"runid"`
	IP                    string `redis:"ip"`
	DownAfterMilliseconds int    `redis:"down-after-milliseconds"`
	IsMasterDown          bool   `redis:"is-master-down"`
	LastOkPingReply       int    `redis:"last-ok-ping-reply"`
	RoleReportedTime      int    `redis:"role-reported-time"`
	InfoRefresh           int    `redis:"info-refresh"`
	RoleReported          string `redis:"role-reported"`
	LastPingReply         int    `redis:"last-ping-reply"`
	LastPingSent          int    `redis:"last-ping-sent"`
	FailoverTimeout       int    `redis:"failover-timeout"`
	ConfigEpoch           int    `redis:"config-epoch"`
	Flags                 string `redis:"flags"`
}

// SlaveInfo is a struct for the results returned from slave queries,
// specifically the individual entries of the  `sentinel slave <podname>`
// command. As with the other Sentinel structs this may change and will need
// updated for new entries
// Currently the members defined by sentinel are as follows.
// "name"
// "ip"
// "port"
// "runid"
// "flags"
// "pending-commands"
// "last-ping-sent"
// "last-ok-ping-reply"
// "last-ping-reply"
// "down-after-milliseconds"
// "info-refresh"
// "role-reported"
// "role-reported-time"
// "master-link-down-time"
// "master-link-status"
// "master-host"
// "master-port"
// "slave-priority"
// "slave-repl-offset"
type SlaveInfo struct {
	Name                   string `redis:"name"`
	Host                   string `redis:"ip"`
	Port                   int    `redis:"port"`
	Runid                  string `redis:"runid"`
	Flags                  string `redis:"flags"`
	PendingCommands        int    `redis:"pending-commands"`
	IsMasterDown           bool   `redis:"is-master-down"`
	LastOkPingReply        int    `redis:"last-ok-ping-reply"`
	RoleReportedTime       int    `redis:"role-reported-time"`
	LastPingReply          int    `redis:"last-ping-reply"`
	LastPingSent           int    `redis:"last-ping-sent"`
	InfoRefresh            int    `redis:"info-refresh"`
	RoleReported           string `redis:"role-reported"`
	MasterLinkDownTime     int    `redis:"master-link-down-time"`
	MasterLinkStatus       string `redis:"master-link-status"`
	MasterHost             string `redis:"master-host"`
	MasterPort             int    `redis:"master-port"`
	SlavePriority          int    `redis:"slave-priority"`
	SlaveReplicationOffset int    `redis:"slave-repl-offset"`
}

// SentinelInfo represents the information returned from a "SENTINEL SENTINELS
// <name>" command
type SentinelInfo struct {
	Name                  string `redis:"name"`
	IP                    string `redis:"ip"`
	Port                  int    `redis:"port"`
	Runid                 string `redis:"runid"`
	Flags                 string `redis:"flags"`
	PendingCommands       int    `redis:"pending-commands"`
	LastPingReply         int    `redis:"last-ping-reply"`
	LastPingSent          int    `redis:"last-ping-sent"`
	LastOkPingReply       int    `redis:"last-ok-ping-reply"`
	DownAfterMilliseconds int    `redis:"down-after-milliseconds"`
	LastHelloMessage      int    `redis:"last-hello-message"`
	VotedLeader           string `redis:"voted-leader"`
	VotedLeaderEpoch      int    `redis:"voted-leader-epoch"`
}
