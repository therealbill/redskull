package structures

// <id> <ip:port> <flags> <master> <ping-sent> <pong-recv> <config-epoch> <link-state> <slot> <slot> ... <slot>
type ClusterNode struct {
	Id          string
	Address     string
	Flags       []string
	Master      string
	PingSent    int64
	PongRecv    int64
	ConfigEpoch int
	LinkStateUp bool
	Slots       []ClusterSlot
}

type ClusterSlot struct {
	Start      int64
	End        int64
	MasterHost string
	MasterPort int64
	Slaves     []string
}
