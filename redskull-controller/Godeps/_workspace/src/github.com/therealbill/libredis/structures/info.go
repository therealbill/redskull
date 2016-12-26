package structures

// InfoServer is a struct for the results returned from an 'INFO server'
// command
//
// Currently the members defined by redis are as follows.
// redis_version
// redis_git_sha1
// redis_git_dirty
// redis_build_id
// redis_mode
// os
// arch_bits
// multiplexing_api
// gcc_version
// process_id
// run_id
// tcp_port
// uptime_in_seconds
// uptime_in_days
// hz
// lru_clock
// config_file
//
type InfoServer struct {
	Version         string `redis:"redis_version"`
	Git_sha1        int    `redis:"redis_git_sha1"`
	Git_dirty       bool   `redis:"redis_git_dirty"`
	Mode            string `redis:"redis_mode"`
	OS              string `redis:"os"`
	Arch_bits       int    `redis:"arch_bits"`
	GCC_version     string `redis:"gcc_version"`
	ProcessId       int    `redis:"process_id"`
	TCPPort         int    `redis:"tcp_port"`
	UptimeInSeconds int    `redis:"uptime_in_seconds"`
	UptimeInDays    int    `redis:"uptime_in_days"`
	Hz              int    `redis:"hz"`
	LRU_clock       int    `redis:"lru_clock"`
	ConfigFile      string `redis:"config_file"`
}

// InfoCluster is a struct representing the Cluster section of Redis Info
type InfoCluster struct {
	Enabled bool
}

// InfoMemory is the struct returning the memory section for Redis Info
type InfoMemory struct {
	UsedMemory               int     `redis:"used_memory"`
	UsedMemoryHuman          string  `redis:"used_memory_human"`
	UsedMemoryRss            int     `redis:"used_memory_rss"`
	UsedMemoryPeak           int     `redis:"used_memory_peak"`
	UsedMemoryPeakHuman      string  `redis:"used_memory_peak_human"`
	UsedMemoryLua            int     `redis:"used_memory_lua"`
	MemoryFragmentationRatio float64 `redis:"mem_fragmentation_ratio"`
	MemoryAllocator          string  `redis:"mem_allocator"`
}

// InfoClients represents the Client section of Redis INFO
type InfoClients struct {
	ConnectedClients         int `redis:"connected_clients"`
	ClientLongestOutputList  int `redis:"client_longest_output_list"`
	ClientBiggestInputBuffer int `redis:"client_biggest_input_buf"`
	BlockedClients           int `redis:"blocked_clients"`
}

//InfoPersistence reprsents the Perisstence section of Redis INFO
type InfoPersistence struct {
	Loading                     bool   `redis:"loading"`
	ChangesSinceSave            int    `redis:"rdb_changes_since_last_save"`
	BGSavesInProgress           bool   `redis:"rdb_bgsave_in_progress"`
	LastSaveTime                uint   `redis:"rdb_last_save_time"`
	LastBGSaveStatus            string `redis:"rdb_last_bgsave_status"`
	LastBGSaveTime              uint   `redis:"rdb_last_bgsave_time_sec"`
	CurrentBGSaveTime           uint   `redis:"rdb_current_bgsave_time_sec"`
	AOFEnabled                  bool   `redis:"aof_enabled"`
	FRewriteInProgress          bool   `redis:"aof_rewrite_in_progress"`
	RewriteScheduled            bool   `redis:"aof_rewrite_scheduled"`
	LastRewriteTimeInSeconds    int    `redis:"aof_last_rewrite_time_sec"`
	CurrentRewriteTimeInSeconds int    `redis:"aof_current_rewrite_time_sec"`
	LastBGRewriteStatus         string `redis:"aof_last_bgrewrite_status"`
	LastAOFWriteSTats           string `redis:"aof_last_write_status"`
}

// InfoStats represents the Stats section of Redis INFO
type InfoStats struct {
	TotalConnectionsRecevied int     `redis:"total_connections_received"`
	TotalCommandsProcessed   int     `redis:"total_commands_processed"`
	InstanteousOpsPerSecond  int     `redis:"instantaneous_ops_per_sec"`
	TotalNetInputBytes       int     `redis:"total_net_input_bytes"`
	TotalNetOutputBytes      int     `redis:"total_net_output_bytes"`
	InstanteousInputKbps     float64 `redis:"instantaneous_input_kbps"`
	InstanteousOutputKbps    float64 `redis:"instantaneous_output_kbps"`
	RejectedConnections      int     `redis:"rejected_connections"`
	SyncFill                 int     `redis:"sync_full"`
	SyncPartialOk            int     `redis:"sync_partial_ok"`
	SyncPartialErr           int     `redis:"sync_partial_err"`
	ExpiredKeys              int     `redis:"expired_keys"`
	EvictedKeys              int     `redis:"evicted_keys"`
	KeyspaceHits             int     `redis:"keyspace_hits"`
	KeyspaceMisses           int     `redis:"keyspace_misses"`
	PubSubChannels           int     `redis:"pubsub_channels"`
	PubSubPatterns           int     `redis:"pubsub_patterns"`
	LatestForkUsec           int     `redis:"latest_fork_usec"`
}

// InfoReplication represents the Replication section of Redis INFO
type InfoReplication struct {
	Role                        string       `redis:"role"`
	ConnectedSlaves             int          `redis:"connected_slaves"`
	MasterReplicationOffset     int          `redis:"master_repl_offset"`
	ReplicationBacklogActive    int          `redis:"repl_backlog_active"`
	ReplicationBacklogSize      int          `redis:"repl_backlog_size"`
	ReplicationBacklogFirstByte int          `redis:"repl_backlog_first_byte_offset"`
	ReplicationBacklogHistLen   int          `redis:"repl_backlog_histlen"`
	MasterHost                  string       `redis:"master_host"`
	MasterPort                  int          `redis:"master_port"`
	MasterLinkStatus            string       `redis:"master_link_status"`
	MasterLastIOSecondsAgo      int          `redis:"master_last_io_seconds_ago"`
	MasterSyncInProgress        bool         `redis:"master_sync_in_progress"`
	SlaveReplicationOffset      int          `redis:"slave_repl_offset"`
	SlavePriority               int          `redis:"slave_priority"`
	SlaveReadOnly               bool         `redis:"slave_read_only"`
	Slaves                      []InfoSlaves `redis:"slave*"`
}

// InfoCPU represents the CPU section of Redis INFO
type InfoCPU struct {
	UsedCPUSystem       float64 `redis:"used_cpu_sys"`
	UsedCPUUser         float64 `redis:"used_cpu_user"`
	UsedCPUChilden      float64 `redis:"used_cpu_sys_children"`
	UsedCPUUserChildren float64 `redis:"used_cpu_user_children"`
}

// InfoKeyspace represents the Keyspace section of Redis INFO
type InfoKeyspace struct {
	Databases []map[string]int64
}

// InfoCommandStats represents the CommandStats section of Redis INFO
type InfoCommandStats struct {
	Stats map[string]map[string]float64 `redis:"cmdstat_del"`
}

// RedisInfoAll is a struct containing structs for each redis section
type RedisInfoAll struct {
	Server       InfoServer       `section:"server"`
	CPU          InfoCPU          `section:"cpu"`
	Client       InfoClients      `section:"client"`
	Replication  InfoReplication  `section:"replication"`
	Memory       InfoMemory       `section:"memory"`
	Stats        InfoStats        `section:"stats"`
	Persistence  InfoPersistence  `section:"persistence"`
	Keyspace     InfoKeyspace     `section:"keyspace"`
	Commandstats InfoCommandStats `section:"commandstats"`
}

// AllInfoConfig is used to contain the RedisInfoAll struct and the data parsed
// from the info command
type AllInfoConfig struct {
	Input map[string]map[string]string
	Info  RedisInfoAll
}

//InfoSlaves represents the slave identity in the replication section of the
//Redis INFO command
type InfoSlaves struct {
	IP     string
	Port   int
	State  string
	Offset int
	Lag    int
}
