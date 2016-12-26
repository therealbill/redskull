package common

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/therealbill/libredis/client"
)

var NodeRefreshInterval float64
var NodesMap map[string]*RedisNode
var DialTimeout time.Duration = 900 * time.Millisecond

// UpdateData will check if an update is needed, and update if so. It returns a
// boolean indicating if an update was done and an err.
func (n *RedisNode) UpdateData() (bool, error) {
	// If the last update was successful and it has been less than 10 seconds,
	// don't bother.
	if n == nil {
		log.Print("WTF?! a nill node?")
		return false, errors.New("Node given does not exist in the system. SOmewhere there is a bug.")
	}
	if n.LastUpdateValid {
		elapsed := time.Since(n.LastUpdate)
		if elapsed.Seconds() < NodeRefreshInterval {
			n.LastUpdateDelay = time.Since(n.LastUpdate)
			return false, nil
		}
	}
	dconf := client.DialConfig{Address: n.Name, Password: n.Auth, Network: "tcp", Timeout: DialTimeout}
	conn, err := client.DialWithConfig(&dconf)
	//deadline := time.Now().Add(DialTimeout)
	if err != nil {
		log.Print("unable to connect to node. Err:", err)
		n.LastUpdateValid = false
		n.LastUpdateDelay = time.Since(n.LastUpdate)
		return false, err
	}
	defer conn.ClosePool()
	nodeinfo, err := conn.Info()
	if err != nil {
		log.Print("Info error on node. Err:", err)
		n.LastUpdateValid = false
		n.LastUpdateDelay = time.Since(n.LastUpdate)
		return false, err
	}
	n.LastUpdate = time.Now()
	if nodeinfo.Server.Version == "" {
		log.Print("WARNING: Unable to get INFO or node!")
		n.LastUpdateValid = false
		n.LastUpdateDelay = time.Since(n.LastUpdate)
		return false, fmt.Errorf("Unable to pull valid INFO for %s", n.Name)
	}

	n.Info = nodeinfo
	res, _ := conn.ConfigGet("maxmemory")
	maxmem, err := strconv.Atoi(res["maxmemory"])
	n.MaxMemory = maxmem
	if err != nil {
	}
	uptime := time.Duration(n.Info.Server.UptimeInSeconds) * time.Second
	now := time.Now()
	ud := time.Duration(0 - uptime)
	n.LastStart = now.Add(ud)

	cfg, err := conn.ConfigGet("save")
	if err != nil {
		log.Print("Unable to get 'save' from config call")
	}
	does_save := cfg["save"]
	if len(does_save) != 0 {
		n.SaveEnabled = true
	}
	n.AOFEnabled = n.Info.Persistence.AOFEnabled
	if n.MaxMemory == 0 {
		n.PercentUsed = 100.0
		n.MemoryUseCritical = true
	} else {
		n.PercentUsed = (float64(n.Info.Memory.UsedMemory) / float64(n.MaxMemory)) * 100.0
		if n.PercentUsed >= 80 {
			n.MemoryUseCritical = true
		} else if n.PercentUsed >= 60 {
			n.MemoryUseWarn = true
		}
	}

	// Pull Latency data
	res, _ = conn.ConfigGet("latency-monitor-threshold")
	n.LatencyThreshold, err = strconv.Atoi(res["latency-monitor-threshold"])
	if err == nil && n.LatencyThreshold > 0 {
		n.LatencyHistory, _ = conn.LatencyHistory("command")
		n.LatencyHistoryFastCommand, _ = conn.LatencyHistory("fast-command")
		n.LatencyDoctor, _ = conn.LatencyDoctor()
		n.LatencyMonitoringEnabled = true
	}

	// Pull slowlog data
	res, _ = conn.ConfigGet("slowlog-log-slower-than")
	n.SlowLogThreshold, err = strconv.ParseInt(res["slowlog-log-slower-than"], 0, 64)
	n.SlowLogLength, _ = conn.SlowLogLen()
	n.SlowLogRecords, _ = conn.SlowLogGet(n.SlowLogLength)

	var slavenodes []*RedisNode
	for _, slave := range n.Info.Replication.Slaves {
		snode, err := LoadNodeFromHostPort(slave.IP, slave.Port, n.Auth)
		if err != nil {
			log.Printf("Unable to load node from %s:%d. Error:%s", slave.IP, slave.Port, err)
			continue
		}
		slavenodes = append(slavenodes, snode)
	}
	if n.Slaves == nil {
		n.Slaves = make([]*RedisNode, 5)
	}
	n.Slaves = slavenodes
	n.LastUpdateValid = true
	n.LastUpdate = time.Now()
	n.LastUpdateDelay = time.Since(n.LastUpdate)
	NodesMap[n.Name] = n
	return true, nil
}

func (n *RedisNode) UptimeHuman() string {
	return humanize.Time(n.LastStart)
}

func (n *RedisNode) MaxMemoryHuman() string {
	return humanize.Bytes(uint64(n.MaxMemory))
}

func (n *RedisNode) InErrorState() bool {
	n.UpdateData()
	if n.MemoryUseCritical {
		return true
	}
	if !n.LastUpdateValid {
		return true
	}
	return false
}

func (n *RedisNode) IsPromotable() bool {
	return n.Info.Replication.SlavePriority > 0
}

func (n *RedisNode) IsFree() bool {
	return false
}

func (n *RedisNode) Ping() bool {
	dconf := client.DialConfig{Address: n.Name, Password: n.Auth, Network: "tcp", Timeout: DialTimeout}
	conn, err := client.DialWithConfig(&dconf)
	if err != nil {
		return false
	}
	err = conn.Ping()
	if err != nil {
		return false
	}
	return true
}

type NodeStore struct {
	Name         string
	Type         string
	Nodes        []*RedisNode
	NodesMap     map[string]*RedisNode
	NodesInError []*RedisNode
	FreeNodes    []*RedisNode
	//HasNodesInErrorState bool
}

type NodeManager interface {
	GetNodes() []*RedisNode
	GetNode(string) *RedisNode
	GetFreeNodes() []*RedisNode
	GetNodesInError() []*RedisNode
	LoadNodes() bool
	HasNodesInErrorState() bool
	NodeCount() int
	ErrorNodeCount() int
	FreeNodeCount() int
	HasFreeNodes() bool
	AddNode(*RedisNode)
}

func LoadNodeFromHostPort(ip string, port int, authtoken string) (node *RedisNode, err error) {
	name := fmt.Sprintf("%s:%d", ip, port)
	node, exists := NodesMap[name]
	if exists {
		return node, nil
	}
	node = &RedisNode{Name: name, Address: ip, Port: port, Auth: authtoken}
	node.LastUpdateValid = false
	node.Slaves = make([]*RedisNode, 5)

	conn, err := client.DialWithConfig(&client.DialConfig{Address: name, Password: authtoken, Timeout: DialTimeout})
	if err != nil {
		log.Printf("Failed connection to %s:%d. Error:%s", ip, port, err.Error())
		return node, err
	}
	defer conn.ClosePool()

	node.Connected = true
	nodeInfo, err := conn.Info()
	if err != nil {
		if strings.Contains(err.Error(), "password") {
			node.HasValidAuth = false
		}
		log.Printf("WARNING: NODE '%s' was unable to return Info(). Error='%s'", name, err)
		return node, err
	}
	if nodeInfo.Server.Version == "" {
		log.Printf("WARNING: NODE '%s' was unable to return Info(). Error=NONE", name)
		return node, err
	}
	node.HasValidAuth = true
	_, err = node.UpdateData()
	if err != nil {
		log.Printf("Node %s has invalid state. Err from UpdateData call: %s", node.Name, err)
		return node, err
	}
	node.Info = nodeInfo
	NodesMap[name] = node
	//log.Printf("node: %+v", node)
	return node, nil
}

func (nm *NodeStore) HasFreeNodes() bool {
	if len(nm.FreeNodes) != 0 {
		return true
	}
	return false
}
func (nm *NodeStore) HasNodesInErrorState() bool {
	if len(nm.NodesInError) != 0 {
		return true
	}
	for _, n := range nm.NodesMap {
		if n.InErrorState() {
			return true
		}
	}
	return false
}

func (nm *NodeStore) LoadNodes() bool {
	return true
}

func (nm *NodeStore) AddNode(node *RedisNode) {
	if nm.NodesMap == nil {
		nm.NodesMap = make(map[string]*RedisNode)
	}
	if _, ok := nm.NodesMap[node.Name]; !ok {
		nm.Nodes = append(nm.Nodes, node)
		nm.NodesMap[node.Name] = node
		for _, snode := range node.Slaves {
			snode.Auth = node.Auth
			nm.AddNode(snode)
		}
		return
	}
	for _, snode := range node.Slaves {
		snode.Auth = node.Auth
		nm.AddNode(snode)
	}
}

func (nm *NodeStore) ErrorNodeCount() (count int) {
	for _, node := range nm.NodesMap {
		if node.InErrorState() {
			count++
		}
	}
	return
}

func (nm *NodeStore) FreeNodeCount() int {
	return len(nm.FreeNodes)
}

func (nm *NodeStore) NodeCount() int {
	return len(nm.Nodes)
}

func (nm *NodeStore) GetNodesInError() (nodes []*RedisNode) {
	for _, node := range nm.NodesMap {
		if node.InErrorState() {
			nodes = append(nodes, node)
		}
	}
	return
}

func (nm *NodeStore) GetFreeNodes() (nodes []*RedisNode) {
	return
}
func (nm *NodeStore) GetNodes() (nodes []*RedisNode) {
	for _, node := range nm.NodesMap {
		nodes = append(nodes, node)
	}
	return
}

func (nm *NodeStore) GetNode(name string) (node *RedisNode) {
	if len(name) == 0 {
		log.Print("Called w/empty name")
	}
	for _, node := range nm.Nodes {
		if node.Name == name {
			node.UpdateData()
			return node
		}
	}
	log.Print("Node not found:", name)
	return node
}
