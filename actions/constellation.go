package actions

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/libredis/structures"
	"github.com/therealbill/redskull/common"
)

const GCPORT = "8008"

// SentinelPodConfig is a struct carrying information about a Pod's config as
// pulled from the sentinel config file.
type SentinelPodConfig struct {
	IP        string
	Port      int
	Quorum    int
	Name      string
	AuthToken string
	Sentinels map[string]string
}

// LocalSentinelConfig is a struct holding information about the sentinel RS is
// running on.
type LocalSentinelConfig struct {
	Name              string
	Host              string
	Port              int
	ManagedPodConfigs map[string]SentinelPodConfig
	Dir               string
}

// Constellation is a construct which holds information about the constellation
// as well providing an interface for taking actions against it.
type Constellation struct {
	Name                string
	PodMap              map[string]*RedisPod
	LocalPodMap         map[string]*RedisPod
	RemotePodMap        map[string]*RedisPod
	PodsInError         []*RedisPod
	NumErrorPods        int
	LastErrorCheck      time.Time
	Connected           bool
	RemoteSentinels     map[string]*Sentinel
	BadSentinels        map[string]*Sentinel
	LocalSentinel       Sentinel
	SentinelConfigName  string
	SentinelConfig      LocalSentinelConfig
	PodToSentinelsMap   map[string][]*Sentinel
	Balanced            bool
	PodAuthMap          map[string]string
	NodeMap             map[string]*RedisNode
	NodeNameToPodMap    map[string]string
	ConfiguredSentinels map[string]interface{}
	Metrics             ConstellationStats
	LocalOverrides      SentinelOverrides
}
type SentinelOverrides struct {
	BindAddress string
}

// GetConstellation returns an instance of a constellation. It requires the
// configuration and a group name. The group name identifies the cluster the
// constellation, and hence this RedSkull instance, belongs to.
// In the future this will be used in clsuter coordination as well as for a
// protective measure against cluster merge
func GetConstellation(name, cfg, group, sentinelAddress string) (Constellation, error) {
	con := Constellation{Name: name}
	con.SentinelConfig.ManagedPodConfigs = make(map[string]SentinelPodConfig)
	con.PodToSentinelsMap = make(map[string][]*Sentinel)
	con.RemoteSentinels = make(map[string]*Sentinel)
	con.BadSentinels = make(map[string]*Sentinel)
	con.PodAuthMap = make(map[string]string)
	con.PodMap = make(map[string]*RedisPod)
	con.LocalPodMap = make(map[string]*RedisPod)
	con.RemotePodMap = make(map[string]*RedisPod)
	con.NodeMap = make(map[string]*RedisNode)
	con.NodeNameToPodMap = make(map[string]string)
	con.ConfiguredSentinels = make(map[string]interface{})
	con.LocalOverrides = SentinelOverrides{BindAddress: sentinelAddress}
	con.SentinelConfigName = cfg
	con.LoadSentinelConfigFile()
	con.LoadLocalPods()
	con.LoadRemoteSentinels()
	con.Balanced = true
	//con.GetStats()
	return con, nil
}

// ConstellationStats holds mtrics about the constellation. As the
// Constellation term is undergoing a change, this will also need to change to
// reflect the new terminology.
// As soon as it is determined
type ConstellationStats struct {
	PodCount        int
	NodeCount       int
	TotalPodMemory  int64
	TotalNodeMemory int64
	SentinelCount   int
	PodSizes        map[int64]int64
	MemoryUsed      int64
	MemoryPctAvail  float64
}

// GetStats returns metrics about the constellation
func (c *Constellation) GetStats() ConstellationStats {
	// first: pod crawling
	var metrics ConstellationStats
	metrics.PodSizes = make(map[int64]int64)
	pmap := c.PodMap
	metrics.PodCount = len(pmap)
	metrics.SentinelCount = len(c.RemoteSentinels) + 1

	for _, pod := range pmap {
		master := pod.Master
		if master == nil {
			address := fmt.Sprintf("%s:%d", pod.Info.IP, pod.Info.Port)
			var err error
			master, err = c.GetNode(address, pod.Name, pod.AuthToken)
			if err != nil {
				log.Printf("Unable to get master for pod '%s', ERR='%s'", pod.Name, err)
				continue
			}
		}
		master.UpdateData()
		metrics.NodeCount++
		for _, slave := range master.Slaves {
			metrics.NodeCount++
			metrics.TotalNodeMemory += int64(slave.MaxMemory)
		}
		podmem := int64(master.MaxMemory)
		metrics.TotalPodMemory += podmem
		metrics.TotalNodeMemory += int64(podmem)
		metrics.PodSizes[podmem]++
	}
	c.Metrics = metrics
	return metrics
}

// GetNode will retun an instance of a RedisNode.
// It also attempts to determine dynamic data such as sentinels and booleans
// like CanFailover
func (c *Constellation) GetNode(name, podname, auth string) (node *RedisNode, err error) {
	if c.NodeMap == nil {
		log.Fatal("c.NodeMap is not initialized. wtf?!")
	}
	node, exists := c.NodeMap[name]
	if exists {
		_, err := node.UpdateData()
		if err != nil {
			log.Print("ERROR in GetNode:Update -> ", err)
			// somehow I need to find a good way to bubble up this error as it usually means bad auth
			return node, err
		}
		if c.NodeMap == nil {
			c.NodeMap = make(map[string]*RedisNode)
		}
		if c.NodeNameToPodMap == nil {
			c.NodeNameToPodMap = make(map[string]string)
		}
		c.NodeMap[name] = node
		c.NodeNameToPodMap[name] = podname
		return node, err
	}
	if auth == "" {
		log.Print("Auth was blank when called, trying to determine it from store - ", podname)
		auth = c.GetPodAuth(podname)
	}
	host, port, err := GetAddressPair(name)
	if err != nil {
		log.Print("Unable to determine connection info. Err:", err)
		return
	}
	node, err = LoadNodeFromHostPort(host, port, auth)
	if err != nil {
		log.Print("Unable to obtain connection . Err:", err)
		return
	}
	c.NodeMap[name] = node
	return
}

// GetPodAuth will return an authstring from the store, or local configs if not in store
func (c *Constellation) GetPodAuth(podname string) string {
	auth, err := common.Backingstore.Get(fmt.Sprintf("%s/%s/auth", common.StoreConfig.Podbase, podname))
	if err != nil {
		log.Printf("Error on libkv Get call: '%v'", err)
		return ""
	} else {
		return string(auth.Value)
	}
	return ""
}

// LoadLocalPods uses the PodConfigs read from the sentinel config file and
// talks to the local sentinel to develop the list of pods the local sentinel
// knows about.
func (c *Constellation) LoadLocalPods() error {
	if c.LocalPodMap == nil {
		c.LocalPodMap = make(map[string]*RedisPod)
	}
	if c.RemotePodMap == nil {
		c.RemotePodMap = make(map[string]*RedisPod)
	}
	if c.PodMap == nil {
		c.PodMap = make(map[string]*RedisPod)
	}
	// Initialize local sentinel
	if c.LocalSentinel.Name == "" {
		log.Print("Initializing LOCAL sentinel")
		var address string
		var err error
		if c.SentinelConfig.Host == "" {
			log.Print("No Hostname, determining local hostname")
			myhostname, err := os.Hostname()
			if err != nil {
				log.Print(err)
			}
			myip, err := net.LookupHost(myhostname)
			if err != nil {
				log.Fatal(err)
			}
			c.LocalSentinel.Host = myip[0]
			c.SentinelConfig.Host = myip[0]
			log.Printf("%+v", myip)
			address = fmt.Sprintf("%s:%d", myip[0], c.SentinelConfig.Port)
			c.LocalSentinel.Name = address
			log.Printf("Determined LOCAL address is: %s", address)
			log.Printf("Determined LOCAL name is: %s", c.LocalSentinel.Name)
			c.Name = address
		} else {
			address = fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
			log.Printf("Determined LOCAL address is: %s", address)
			c.LocalSentinel.Name = address
			log.Printf("Determined LOCAL name is: %s", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Host = c.SentinelConfig.Host
		c.LocalSentinel.Port = c.SentinelConfig.Port
		c.LocalSentinel.Connection, err = client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here!
			//log.Printf("SentinelConfig=%+v", c.SentinelConfig)
			log.Fatalf("LOCAL Sentinel '%s' failed connection attempt", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Info, _ = c.LocalSentinel.Connection.SentinelInfo()
	}
	log.Print("INitial iteration through ManagedPodConfigs")
	local_config_count := len(c.SentinelConfig.ManagedPodConfigs)
	ctr := 0
	for pname, pconfig := range c.SentinelConfig.ManagedPodConfigs {
		log.Printf("pconfig: %+v\n", pconfig)
		keybase := fmt.Sprintf("%s/%s", common.StoreConfig.Podbase, pname)
		log.Printf("MPCRoll podbase: %s", common.StoreConfig.Podbase)
		log.Printf("MPCRoll keybase: %s", keybase)
		common.Backingstore.Put(keybase+"/auth", []byte(pconfig.AuthToken), nil)
		common.Backingstore.Put(keybase+"/master/ip", []byte(pconfig.IP), nil)
		common.Backingstore.Put(keybase+"/master/port", []byte(fmt.Sprintf("%d", pconfig.Port)), nil)
		common.Backingstore.Put(keybase+"/quorum", []byte(fmt.Sprintf("%d", pconfig.Quorum)), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/nodes/%s:%d/auth", keybase, pconfig.IP, pconfig.Port), []byte(pconfig.AuthToken), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/nodes/%s:%d/role", keybase, pconfig.IP, pconfig.Port), []byte("master"), nil)
		sentinels_string := fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
		common.Backingstore.Put(keybase+"/sentinels/"+fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port), []byte(sentinels_string), nil)
		for k, _ := range pconfig.Sentinels {
			common.Backingstore.Put(keybase+"/sentinels/"+k, []byte(k), nil)
			if sentinels_string != "" {
				sentinels_string += ","
			}
			sentinels_string += k
		}
		log.Printf("SentinelConfig: %+v\n", c.SentinelConfig)
		mi, err := c.LocalSentinel.GetMaster(pname)
		if err != nil {
			log.Printf("WARNING: Pod '%s' in config but not found when talking to the sentinel controller. Err: '%s'", pname, err)
			continue
		}
		address := fmt.Sprintf("%s:%d", mi.Host, mi.Port)
		common.Backingstore.Put(keybase+"/master/address", []byte(address), nil)
		pod, err := c.LocalSentinel.GetPod(pname)
		master, err := c.GetNode(address, pname, pconfig.AuthToken)
		//c.GetNode(address, pname, pconfig.AuthToken)
		if err != nil {
			log.Printf("Was unable to get node '%s' for pod '%s' with auth '%s'", address, pname, pconfig.AuthToken)
			if strings.Contains(err.Error(), "password") {
				log.Print("marking pod/node auth invalid")
				master.HasValidAuth = false
				pod.ValidAuth = false
			}
		} else {
			pod.ValidAuth = true
			master.HasValidAuth = true
		}
		if err != nil {
			log.Printf("ERROR: No pod found on LOCAL sentinel for %s", pname)
		}
		if c.PodMap == nil {
			c.PodMap = make(map[string]*RedisPod)
		}
		if c.LocalPodMap == nil {
			c.LocalPodMap = make(map[string]*RedisPod)
		}
		pod.Master = master
		c.PodMap[pod.Name] = &pod
		c.LocalPodMap[pod.Name] = &pod
		c.LoadNodesForPod(&pod, &c.LocalSentinel)
		ctr++
		log.Printf("Loaded %d of %d configured local pods", ctr, local_config_count)
	}

	log.Print("Done with LocalSentinel initialization")
	return nil
}

// GetAuthForPodFromConfig looks in the local sentinel config file to find an
// authentication token for the given pod.
func (c *Constellation) GetAuthForPodFromConfig(podname string) string {
	config, _ := c.SentinelConfig.ManagedPodConfigs[podname]
	return config.AuthToken
}

// IsBalanced is likely to be deprecated. What it currently does is to look
// across the known sentinels and pods and determine if any pod is
// "unbalanced".
func (c *Constellation) IsBalanced() (isbal bool) {
	if c.Balanced == false {
		return c.Balanced
	}
	isbal = true
	needed_monitors := 0
	monitors := 0
	for _, sentinel := range c.GetAllSentinelsQuietly() {
		pc := sentinel.PodCount()
		monitors += pc
	}
	for name, sentinels := range c.PodToSentinelsMap {
		// First try to get from local sentinel, then iterate over the rest to
		// find it
		pod, err := c.LocalSentinel.GetPod(name)
		if err != nil {
			for _, s := range sentinels {
				pod, err = s.GetPod(name)
				if err == nil {
					break
				}
			}
		}
		if pod.Name == "" {
			log.Printf("WARNING: Unable to get pod '%s' from anywhere", name)
		}
		needed := pod.Info.Quorum + 1
		needed_monitors += needed
		if len(sentinels) == 0 {
			log.Printf("WARNING: Pod %s has no sentinels?? trying to find some", pod.Name)
			sentinels = c.GetSentinelsForPod(pod.Name)
			c.PodToSentinelsMap[pod.Name] = sentinels
		}
		pod.SentinelCount = len(sentinels)
		if pod.SentinelCount < needed {
			log.Printf("Pod '%s' has %d of %d needed sentinels monitoring it, thus we are unbalanced", pod.Name, pod.SentinelCount, needed)
			isbal = false
			c.Balanced = isbal
			return isbal
		}
	}
	if needed_monitors > monitors {
		log.Printf("Need total of %d monitors, have %d", needed_monitors, monitors)
		isbal = false
	}
	c.Balanced = isbal
	return isbal
}

// MonitorPod is used to add a pod/master to the constellation cluster.
func (c *Constellation) MonitorPod(podname, address string, port, quorum int, auth string) (ok bool, err error) {
	_, havekey := c.LocalPodMap[podname]
	if havekey {
		err = fmt.Errorf("C:MP -> Pod '%s' already being monitored", podname)
		return false, err
	}
	_, havekey = c.RemotePodMap[podname]
	if havekey {
		err = fmt.Errorf("C:MP -> Pod '%s' already being monitored", podname)
		return false, err
	}
	quorumReached := false
	successfulSentinels := 0
	var pod RedisPod
	neededSentinels := quorum + 1

	sentinels, err := c.GetAvailableSentinels(podname, neededSentinels)
	if err != nil {
		log.Print("NO sentinels available! Error:", err)
		return false, err
	}
	//c.PodAuthMap[podname] = auth
	common.Backingstore.Put(fmt.Sprintf("%s/%s/auth", common.StoreConfig.Podbase, podname), []byte(auth), nil)
	cfg := SentinelPodConfig{Name: podname, AuthToken: auth, IP: address, Port: port, Quorum: quorum}
	c.SentinelConfig.ManagedPodConfigs[podname] = cfg
	isLocal := false
	for _, sentinel := range sentinels {
		log.Printf("C:MP -> Adding pod to %s", sentinel.Name)
		if sentinel.Name == c.LocalSentinel.Name {
			isLocal = true
		}
		pod, err = sentinel.MonitorPod(podname, address, port, quorum, auth)
		successfulSentinels++
	}
	// I generally dislike sleeps. Hoeever in
	// this case it is a decent ooption for refreshing data from the
	// sentinels
	time.Sleep(2 * time.Second)
	pod.SentinelCount = successfulSentinels
	if isLocal {
		c.LocalPodMap[podname] = &pod
	} else {
		c.RemotePodMap[podname] = &pod
	}
	c.PodToSentinelsMap[podname] = sentinels
	quorumReached = successfulSentinels >= quorum
	if !quorumReached {
		return false, fmt.Errorf("C:MP -> Quorum not reached for pod '%s'", podname)
	}

	return true, nil
}

// RemovePod removes a pod from each of it's sentinels.
func (c *Constellation) RemovePod(podname string) (bool, error) {
	var err error
	sentinels := c.GetSentinelsForPod(podname)
	if err != nil {
		log.Print("RemovePod GetAllSentinels err: ", err)
		return false, err
	}
	log.Printf("Found %d sentinels handling %s", len(sentinels), podname)
	for _, sentinel := range sentinels {
		log.Printf("Removing pod from %s", sentinel.Name)
		ok, err := sentinel.RemovePod(podname)
		if err != nil || !ok {
			log.Printf("Unable to remove %s from %s. Error:%s", podname, sentinel.Name, err.Error())
		}
	}
	delete(c.SentinelConfig.ManagedPodConfigs, podname)
	delete(c.PodMap, podname)
	delete(c.PodToSentinelsMap, podname)
	delete(c.LocalPodMap, podname)
	delete(c.RemotePodMap, podname)
	return true, err
}

// Initiates a failover on a given pod.
func (c *Constellation) Failover(podname string) (ok bool, err error) {
	// change this to iterate over known sentinels via the
	// GetSentinelsForPod call
	didFailover := false
	for _, s := range c.PodToSentinelsMap[podname] {
		didFailover, err = s.DoFailover(podname)
		if didFailover {
			return true, nil
		}
	}
	return didFailover, err
}

// GetAllSentinels returns all known sentinels
func (c *Constellation) GetAllSentinels() (sentinels []*Sentinel, err error) {
	for name, pod := range c.LocalPodMap {
		slist, _ := c.LocalSentinel.GetSentinels(name)
		for _, sent := range slist {
			if sent.Name == c.LocalSentinel.Name {
				continue
			}
			_, exists := c.RemoteSentinels[sent.Name]
			if !exists {
				c.AddSentinelByAddress(sent.Name)
				log.Printf("Added REMOTE sentinel '%s' for LOCAL pod", sent.Name)
			}
		}
		c.PodToSentinelsMap[name] = slist
		pod.SentinelCount = len(slist)
		c.LocalPodMap[pod.Name] = pod
	}
	for name, pod := range c.RemotePodMap {
		_, islocal := c.LocalPodMap[pod.Name]
		if islocal {
			continue
		}
		slist, _ := c.LocalSentinel.GetSentinels(name)
		for _, sent := range slist {
			if sent.Name == c.LocalSentinel.Name {
				continue
			}
			_, exists := c.RemoteSentinels[sent.Name]
			if !exists {
				c.RemoteSentinels[sent.Name] = sent
				c.AddSentinelByAddress(sent.Name)
				log.Printf("Added REMOTE sentinel '%s' for REMOTE pod", sent.Name)
			}
		}
		pod.SentinelCount = len(slist)
		c.RemotePodMap[pod.Name] = pod
	}
	for _, s := range c.RemoteSentinels {
		_, err := s.GetPods()
		if err != nil {
			log.Printf("Sentinel %s -> GetPods err: '%s'", s.Name, err)
			continue
		}
		sentinels = append(sentinels, s)
	}
	sentinels = append(sentinels, &c.LocalSentinel)
	return sentinels, nil
}

// GetSentinelsForPod returns all sentinels the pod is monitored by. In
// other words, the pod's constellation
func (c *Constellation) GetSentinelsForPod(podname string) (sentinels []*Sentinel) {
	pod, err := c.GetPod(podname)
	if err != nil || pod == nil {
		log.Printf("Unable to get pod '%s' from constellation", podname)
		return
	}
	all_sentinels, _ := c.GetAllSentinels()
	knownSentinels := make(map[string]*Sentinel)
	var current_sentinels []*Sentinel
	for _, s := range all_sentinels {
		conn, err := s.GetConnection()
		if err != nil {
			log.Printf("Unable to connect to sentinel %s", s.Name)
			continue
		}
		defer conn.ClosePool()
		reportedSentinels, _ := conn.SentinelSentinels(podname)
		if len(reportedSentinels) == 0 {
			log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset. This can also occur if the master is non-responsive and there are no known slaes for the master.", s.Name, podname)
			continue
		}
		slist, err := s.GetSentinels(podname)
		if err != nil {
			log.Print(err)
			continue
		}
		knownSentinels[s.Name] = s

		// deal with Sentinel not having updated info on sentinels for example
		// if a sentinel loses a pod, nothing is updated. we need to catch
		// this.
		if len(slist) > 0 {
			for _, sentinel := range slist {
				_, known := knownSentinels[sentinel.Name]
				if known {
					continue
				}
				p, err := sentinel.GetSentinels(podname)
				if err != nil {
					log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset", sentinel.Name, podname)
					log.Print("GetPod Err:", err)
				} else {
					if len(p) == 0 {
						log.Printf("Sentinel %s was reported as having pod %s. It doesn't. Pod Needs Reset", sentinel.Name, podname)
						continue
					}
					knownSentinels[sentinel.Name] = sentinel
				}
			}
		}
	}
	for _, sentinel := range knownSentinels {
		current_sentinels = append(current_sentinels, sentinel)
	}
	c.PodToSentinelsMap[podname] = current_sentinels
	pod.SentinelCount = len(current_sentinels)
	log.Printf("Found %d known sentinels for pod %s", pod.SentinelCount, pod.Name)
	return current_sentinels
}

// GetAvailableSentinels returns a list of sentinels the give pod is *not*
// already monitored by. It will return the least-used of the available
// sentinels in an effort to level sentinel use.
func (c *Constellation) GetAvailableSentinels(podname string, needed int) (sentinels []*Sentinel, err error) {
	all, err := c.GetAllSentinels()
	if err != nil {
		return sentinels, err
	}
	if len(all) < needed {
		return sentinels, fmt.Errorf("Not enough sentinels to achieve quorum!")
	}
	pcount := func(s1, s2 *Sentinel) bool { return s1.PodCount() < s2.PodCount() }
	By(pcount).Sort(all)
	if len(all) < needed {
		log.Printf("WTF? needed %d sentinels but only %d available?", needed, len(sentinels))
	}
	// time to do some tricky testing to ensure we get valid sentinels: ones
	// which do not already have this pod on them
	// THis might be cleaner with a dl-list
	w := 0 // write index
	existing_sentinels := c.GetSentinelsForPod(podname)
	if len(existing_sentinels) > 0 {
	loop:
		for _, x := range all {
			for _, es := range existing_sentinels {
				if es.Name == x.Name {
					continue loop
				}
			}
			all[w] = x
			w++
		}
	}
	usable := all[:needed]
	return usable, nil
}

// AddSentinelByAddress is a convenience function to add a sentinel by
// it's ip:port string
func (c *Constellation) AddSentinelByAddress(address string) error {
	apair := strings.Split(address, ":")
	ip := apair[0]
	port, _ := strconv.Atoi(apair[1])
	return c.AddSentinel(ip, port)
}

// LoadRemoteSentinels interrogates all known remote sentinels and crawls
// the results to explore non-local configuration
func (c *Constellation) LoadRemoteSentinels() {
	for k := range c.ConfiguredSentinels {
		log.Printf("INIT REMOTE SENTINEL: %s", k)
		c.AddSentinelByAddress(k)
	}
}

// AddSentinel adds a sentinel to the constellation
func (c *Constellation) AddSentinel(ip string, port int) error {
	if c.LocalSentinel.Name == "" {
		log.Print("Initializing LOCAL sentinel")
		if c.SentinelConfig.Host == "" {
			myhostname, err := os.Hostname()
			if err != nil {
				log.Print(err)
			}
			myip, err := net.LookupHost(myhostname)
			if err != nil {
				log.Print(err)
			}
			c.LocalSentinel.Host = myip[0]
		}
		address := fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
		c.LocalSentinel.Name = address
		var err error
		c.LocalSentinel.Connection, err = client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here! I don't thnk we want to do a
			// fatal here anymore
			log.Fatalf("LOCAL Sentinel '%s' failed connection attempt", c.LocalSentinel.Name)
		}
		c.LocalSentinel.Info, _ = c.LocalSentinel.Connection.SentinelInfo()
	}
	var sentinel Sentinel
	if port == 0 {
		err := fmt.Errorf("AddSentinel called w/ZERO port .. wtf, man?")
		return err
	}
	address := fmt.Sprintf("%s:%d", ip, port)
	//log.Printf("*****************] Local Name: %s Add Called For: %s", c.LocalSentinel.Name, address)
	if address == c.LocalSentinel.Name {
		return nil
	}
	_, exists := c.RemoteSentinels[address]
	if exists {
		return nil
	}
	_, exists = c.BadSentinels[address]
	if exists {
		return nil
	}
	sentinel.Name = address
	sentinel.Host = ip
	sentinel.Port = port
	_, known := c.RemoteSentinels[address]
	if known {
		log.Printf("Already have crawled '%s'", sentinel.Name)
	} else {
		log.Printf("Adding REMOTE Sentinel '%s'", address)
		conn, err := client.DialWithConfig(&client.DialConfig{Address: address})
		if err != nil {
			// Handle error reporting here!
			err = fmt.Errorf("AddSentinel -> '%s' failed connection attempt", address)
			c.BadSentinels[address] = &sentinel
			return err
		}
		sentinel.Connection = conn
		sentinel.Info, _ = sentinel.Connection.SentinelInfo()
		if address != c.LocalSentinel.Name {
			log.Print("discovering pods on remote sentinel " + sentinel.Name)
			sentinel.LoadPods()
			pods, _ := sentinel.GetPods()
			log.Printf("%d Pods to load from %s ", len(pods), address)
			c.RemoteSentinels[address] = &sentinel
			for _, pod := range pods {
				if pod.Name == "" {
					log.Print("WUT: Have a nameless pod. This is probably a bug.")
					continue
				}
				_, islocal := c.LocalPodMap[pod.Name]
				if islocal {
					continue
				}
				_, isremote := c.RemotePodMap[pod.Name]
				if isremote {
					continue
				}
				log.Print("Adding DISCOVERED remotely managed pod " + pod.Name)
				c.GetPodAuth(pod.Name)
				log.Print("Got auth for pod")
				c.LoadNodesForPod(&pod, &sentinel)
				newsentinels, _ := sentinel.GetSentinels(pod.Name)
				pod.SentinelCount = len(newsentinels)
				c.PodToSentinelsMap[pod.Name] = newsentinels
				c.RemotePodMap[pod.Name] = &pod
				for _, ns := range newsentinels {
					_, known := c.RemoteSentinels[ns.Name]
					if known {
						continue
					}
					if ns.Name == c.LocalSentinel.Name || ns.Name == sentinel.Name {
						continue
					}
					c.AddSentinelByAddress(ns.Name)
				}
			}
		}
	}
	return nil
}

// LoadNodesForPod is called to add the master and slave nodes for the
// given pod.
func (c *Constellation) LoadNodesForPod(pod *RedisPod, sentinel *Sentinel) {
	mi, err := sentinel.GetMaster(pod.Name)
	if err != nil {
		log.Printf("WARNING: Pod '%s' in config but not found when talking to the sentinel controller. Err: '%s'", pod.Name, err)
		return
	}
	address := fmt.Sprintf("%s:%d", mi.Host, mi.Port)
	node, err := c.GetNode(address, pod.Name, pod.AuthToken)
	if err != nil {
		log.Printf("Was unable to get node '%s' for pod '%s' with auth '%s'", address, pod.Name, pod.AuthToken)
		if strings.Contains(err.Error(), "password") {
			log.Print("marking auth invalid")
			pod.ValidAuth = false
		}
		return
	}
	pod.ValidAuth = true
	slaves := node.Slaves
	for _, si := range slaves {
		c.GetNode(si.Name, pod.Name, pod.AuthToken)
		keybase := fmt.Sprintf("%s/%s", common.StoreConfig.Podbase, pod.Name)
		common.Backingstore.Put(fmt.Sprintf("%s/nodes/%s/auth", keybase, si.Name), []byte(pod.AuthToken), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/nodes/%s/role", keybase, si.Name), []byte("slave"), nil)
	}

}

// GetAllSentinelsQuietly is a convenience function used primarily in the
// UI.
func (c *Constellation) GetAllSentinelsQuietly() (sentinels []*Sentinel) {
	sentinels, _ = c.GetAllSentinels()
	return
}

// SentinelCount returns the number of known sentinels, including this one
func (c *Constellation) SentinelCount() int {
	//scount := 0
	// I don't like this, but libkv doesn't (yet?) have a way to only give me
	// one level rather than everything under a given node
	skey := fmt.Sprintf("%s/sentinels/", common.StoreConfig.ConstellationBase)
	depth := strings.Count(skey, "/")
	keys, err := common.Backingstore.List(skey)
	if err != nil {
		log.Printf("Error on store List call: '%v'", err)
		return 0
	}
	log.Printf("Sentinels from store: %+v", keys)
	sentinels := make(map[string]interface{})
	for _, x := range keys {
		levels := strings.Split(string(x.Key), "/")
		s := levels[depth]
		sentinels[s] = nil
		//if strings.Count(string(x.Key), "/")-depth == 1 {
		//log.Printf("key: '%s'", string(x.Key))
		//scount++
		//}
	}
	return len(sentinels)
}

// LoadRemotePods loads pods discovered through remote sentinel
// interrogation or througg known-sentinel directives
func (c *Constellation) LoadRemotePods() error {
	if c.RemotePodMap == nil {
		c.RemotePodMap = make(map[string]*RedisPod)
	}
	sentinels := []*Sentinel{&c.LocalSentinel}
	log.Printf("Loading pods on %d sentinels", len(sentinels))
	if len(sentinels) == 0 {
		err := fmt.Errorf("C:LP-> ERROR: All Sentinels failed connection") // This error is becoming more common in the code perhaps moving it to a dedicated method?
		log.Println(err)
		return err
	}
	for si, sentinel := range c.RemoteSentinels {
		log.Printf("Loading remote pods on sentinel % of %", si, len(c.RemoteSentinels))
		if sentinel.Name != c.LocalSentinel.Name {
			pods, err := sentinel.GetPods()
			if err != nil {
				log.Print("C:LP-> sentinel error:", err)
				continue
			}
			for i, pod := range pods {
				_, exists := c.PodMap[pod.Name]
				if exists {
					continue
				}
				log.Printf("loading pod %d of %s", i, len(pods))
				if pod.Name == "" {
					log.Print("WUT: Have a nameless pod. This is probably a bug.")
					continue
				}
				_, err := sentinel.GetSentinels(pod.Name)
				if err != nil {
					log.Printf("WTF? Sentinel returned no sentinels list for it's own pod '%s'", pod.Name)
				} else {
					podauth := c.GetPodAuth(pod.Name)
					pod.AuthToken = podauth
					c.RemotePodMap[pod.Name] = &pod
					c.PodMap[pod.Name] = &pod
				}
			}
		}
	}
	return nil
}

// GetAnySentinel is deprecated and calls to it need to be found and
// refactored so it can die.
func (c *Constellation) GetAnySentinel() (sentinel Sentinel, err error) {
	// randomized code pulled out. this needs to just go away.
	return c.LocalSentinel, nil
}

// HasPodsInErrorState returns true if at least one pod is in an error
// state.
// TODO: this needs to be "cloned" to a HasPodsInWarningState when that
// refoctoring takes place.
func (c *Constellation) HasPodsInErrorState() bool {
	c.NumErrorPods = c.ErrorPodCount()
	if c.NumErrorPods > 0 {
		return true
	}
	return false
}

// ErrorPodCount returns the number of pods currently reporting errors
func (c *Constellation) ErrorPodCount() (count int) {
	log.Print("ErrorPodCount called")
	if time.Since(c.LastErrorCheck) < (3 * time.Second) {
		log.Print("short interval, not refreshing data")
		return c.NumErrorPods
	}
	log.Print("ErrorPodCount calling full check")
	var epods []*RedisPod
	errormap := make(map[string]*RedisPod)
	cleanmap := make(map[string]*RedisPod)
	for _, pod := range c.PodMap {
		_, inerror := errormap[pod.Name]
		_, clean := cleanmap[pod.Name]
		if clean || inerror {
			log.Printf("Pod %s is being checked for errors again..skipping", pod.Name)
			continue
		}
		if pod.HasErrors() {
			log.Printf("pod %s has errors", pod.Name)
			errormap[pod.Name] = pod
			continue
		} else {
			cleanmap[pod.Name] = pod
		}
	}
	for _, pod := range errormap {
		epods = append(epods, pod)
	}
	c.PodsInError = epods
	c.LastErrorCheck = time.Now()
	c.NumErrorPods = len(epods)
	return c.NumErrorPods
}

// GetPodsInError is used to get the list of pods currently reporting
// errors
func (c *Constellation) GetPodsInError() (errors []*RedisPod) {
	log.Print("GetPodsInError called")
	c.ErrorPodCount()
	return c.PodsInError
}

// PodCount updates current pod information and returns the number of pods
// managed by the constellation
func (c *Constellation) PodCount() int {
	podmap, _ := c.GetPodMap()
	return len(podmap)
}

// BalancePod is used to rebalance a pod. This means pulling a lis tof
// available sentinels, determining how many are "missing" and adding
// the pod to the appropriate number of sentinels to bring it up to spec
func (c *Constellation) BalancePod(pod *RedisPod) {
	pod, _ = c.GetPod(pod.Name) // testing a theory
	log.Print("Balance called on Pod" + pod.Name)
	neededTotal := pod.Info.Quorum + 1
	sentinels := c.GetSentinelsForPod(pod.Name)
	pod.SentinelCount = len(sentinels)
	log.Printf("Pod needs %d sentinels, has %d sentinels", neededTotal, pod.SentinelCount)
	if pod.SentinelCount < neededTotal {
		log.Printf("Attempting rebalance of %s \n'%+v' ", pod.Name, pod)
		needed := neededTotal - pod.SentinelCount
		if pod.AuthToken == "" {
			pod.AuthToken = c.GetPodAuth(pod.Name)
		}
		log.Printf("%s on %d sentinels, needs %d more", pod.Name, pod.SentinelCount, needed)
		sentinels, _ := c.GetAvailableSentinels(pod.Name, needed)
		log.Printf("Request %d sentinels for %s, got %d to use", needed, pod.Name, len(sentinels))
		isLocal := false
		for _, sentinel := range sentinels {
			log.Print("Adding to sentinel ", sentinel.Name)
			if sentinel.Name == c.LocalSentinel.Name {
				isLocal = true
			}
			pod, err := sentinel.MonitorPod(pod.Name, pod.Info.IP, pod.Info.Port, pod.Info.Quorum, pod.AuthToken)
			if err != nil {
				log.Printf("Sentinel %s Pod: %s, Error: %s", sentinel.Name, pod.Name, err)
				continue
			}
			c.PodToSentinelsMap[pod.Name] = append(c.PodToSentinelsMap[pod.Name], sentinel)
		}
		time.Sleep(500 * time.Millisecond) // wait for propagation between sentinels
		slist := c.GetSentinelsForPod(pod.Name)
		c.PodToSentinelsMap[pod.Name] = slist
		pod.SentinelCount = len(slist)
		if isLocal {
			c.LocalPodMap[pod.Name] = pod
		} else {
			c.RemotePodMap[pod.Name] = pod
		}
		log.Printf("Rebalance of %s completed, it now has %d sentinels", pod.Name, pod.SentinelCount)
	} else if pod.SentinelCount > neededTotal {
		remove := pod.SentinelCount - neededTotal
		log.Printf("[%s] has too many sentinels, reducing sentinel count by %d", pod.Name, remove)
		index := rand.Intn(neededTotal)
		sentinel := sentinels[index]
		ok, err := sentinel.RemovePod(pod.Name)
		if !ok || err != nil {
			log.Printf("Unable to remove %s from %s. Err:", pod.Name, sentinel.Name)
		}
		c.ResetPod(pod.Name, true)
	}
}

// Balance will attempt to balance the constellation
// A constellation is unbalanced if any pod is not listed as managed by enough
// sentinels to achieve quorum+1
// It will first verify the current balance state to avoid unnecessary balance
// attempts.
// This will likely be deprecated
func (c *Constellation) Balance() {
	log.Print("Balance called on constellation")
	c.HasPodsInErrorState()
	allpods := c.GetPods()

	log.Printf("Constellation rebalance initiated, have %d pods unbalanced", len(c.PodsInError))
	for _, pod := range allpods {
		c.BalancePod(pod)
	}
	c.Balanced = true
}

// Getmaster returns the current structures.MasterAddress struct for the given
// pod
func (c *Constellation) GetMaster(podname string) (master structures.MasterAddress, err error) {
	sentinels, _ := c.GetAllSentinels()
	for _, sentinel := range sentinels {
		master, err := sentinel.GetMaster(podname)
		if err == nil {
			return master, nil
		}
	}
	return master, fmt.Errorf("No Sentinels available for pod '%s'", podname)
}

// GetPod returns a *RedisPod instance for the given podname
func (c *Constellation) GetPod(podname string) (pod *RedisPod, err error) {
	pod, islocal := c.LocalPodMap[podname]
	authtest := c.GetPodAuth(podname)
	log.Printf("got authtest: %s", authtest)

	if islocal {
		spod, err := c.LocalSentinel.GetPod(podname)
		address := fmt.Sprintf("%s:%d", spod.Info.IP, spod.Info.Port)
		auth := spod.AuthToken
		if auth == "" {
			auth = c.GetPodAuth(podname)
		}
		master, _ := c.GetNode(address, podname, auth)
		spod.Master = master
		c.LocalSentinel.GetSlaves(podname)
		c.LoadNodesForPod(pod, &c.LocalSentinel)
		pod = &spod
		c.LocalPodMap[podname] = pod
		c.PodMap[podname] = pod
		return pod, err
	}
	if pod == nil || pod.Master != nil {
		return pod, nil
	}
	sentinels, _ := c.GetAllSentinels()
	for _, s := range sentinels {
		conn, err := s.GetConnection()
		if err != nil {
			log.Printf("Unable to connect to sentinel '%s'", s.Name)
			continue
		}
		defer conn.ClosePool()
		mi, _ := conn.SentinelMasterInfo(podname)
		if mi.Name == podname {
			auth := c.GetPodAuth(podname)
			address := fmt.Sprintf("%s:%d", mi.IP, mi.Port)
			master, err := c.GetNode(address, podname, auth)
			if err != nil {
				log.Print("Unable to get master node ===========")
			}
			pod, _ := NewMasterFromMasterInfo(mi, auth)
			pod.Master = master
			c.RemotePodMap[podname] = &pod
			return &pod, nil
		}
	}

	if err != nil {
		log.Printf("Could NOT load pod '%s' from %s", podname, err)
		return pod, err
	}
	return pod, nil
}

// GetSlaves return a list of client.SlaveInfo structs for the given pod
func (c *Constellation) GetSlaves(podname string) (slaves []structures.SlaveInfo, err error) {
	sentinels, err := c.GetAllSentinels()
	for _, sentinel := range sentinels {
		slaves, err = sentinel.GetSlaves(podname)
		if err == nil {
			return
		}
	}
	return
}

// GetPods returns the list of known pods
func (c *Constellation) GetPods() (pods []*RedisPod) {
	podmap, _ := c.GetPodMap()
	havepods := make(map[string]interface{})
	for _, pod := range podmap {
		if pod.Name == "" {
			log.Print("WUT: Have a nameless pod. Probably a bug")
			continue
		}
		_, have := havepods[pod.Name]
		if !have {
			pods = append(pods, pod)
		}
	}
	return pods
}

// GetPodMap returs the current pod mapping. This combines local and
// remote sentinels to get all known pods in the cluster
func (c *Constellation) GetPodMap() (pods map[string]*RedisPod, err error) {
	pods = make(map[string]*RedisPod)
	for k, v := range c.LocalPodMap {
		pods[k] = v
	}
	for k, v := range c.RemotePodMap {
		_, local := c.LocalPodMap[k]
		if !local {
			_, haveit := pods[k]
			if !haveit {
				pods[k] = v
			}
		}
	}
	return pods, nil
}

// extractSentinelDirective parses the sentinel directives from the
// sentinel config file
func (c *Constellation) extractSentinelDirective(entries []string) error {
	kvroot := fmt.Sprintf("redskull/constellations/primary/sentinels/%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
	podroot := fmt.Sprintf("%s/pods", common.StoreConfig.Podbase)
	switch entries[0] {
	case "monitor":
		pname := entries[1]
		port, _ := strconv.Atoi(entries[3])
		quorum, _ := strconv.Atoi(entries[4])
		spc := SentinelPodConfig{Name: pname, IP: entries[2], Port: port, Quorum: quorum}
		addr := fmt.Sprintf("%s:%d", entries[2], port)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/master/port", podroot, pname), []byte(entries[3]), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/master/ip", podroot, pname), []byte(entries[2]), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/nodes/ip", podroot, pname), []byte(entries[2]), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/nodes/port", podroot, pname), []byte(entries[3]), nil)
		// add to this sentinel's list of pods
		common.Backingstore.Put(fmt.Sprintf("%s/pods/%s/quorum", kvroot, pname), []byte(entries[4]), nil)
		spc.Sentinels = make(map[string]string)
		// normally we should not see duplicate IP:PORT combos, however it
		// can happen when people do things manually and dont' clean up.
		// We need to detect them and ignore the second one if found,
		// reporting the error condition this will require tracking
		// ip:port pairs...
		_, exists := c.SentinelConfig.ManagedPodConfigs[addr]
		if !exists {
			c.SentinelConfig.ManagedPodConfigs[entries[1]] = spc
		}
		return nil

	case "auth-pass":
		pname := entries[1]
		pc := c.SentinelConfig.ManagedPodConfigs[pname]
		pc.AuthToken = entries[2]
		common.Backingstore.Put(fmt.Sprintf("%s/%s/master/auth", podroot, pname), []byte([]byte(pc.AuthToken)), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/auth", podroot, pname), []byte([]byte(pc.AuthToken)), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/port", podroot, pname), []byte([]byte(pc.AuthToken)), nil)
		c.SentinelConfig.ManagedPodConfigs[pname] = pc
		return nil

	case "known-sentinel":
		podname := entries[1]
		sentinel_address := entries[2] + ":" + entries[3]
		pc := c.SentinelConfig.ManagedPodConfigs[podname]
		pc.Sentinels[sentinel_address] = ""
		c.ConfiguredSentinels[sentinel_address] = sentinel_address
		common.Backingstore.Put(fmt.Sprintf("%s/%s/sentinels/%s/%s", podroot, podname, sentinel_address), []byte(sentinel_address), nil)
		return nil

	case "known-slave":
		// Currently ignoring this, but may add call to a node manager.
		podname := entries[1]
		slave_address := entries[2] + ":" + entries[3]
		common.Backingstore.Put(fmt.Sprintf("%s/%s/nodes/%s/%s/role", podroot, podname, slave_address), []byte([]byte("slave")), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/nodes/%s/%s/ip", podroot, podname, slave_address), []byte(entries[2]), nil)
		common.Backingstore.Put(fmt.Sprintf("%s/%s/nodes/%s/%s/port", podroot, podname, slave_address), []byte(entries[2]), nil)
		return nil

	case "config-epoch", "leader-epoch", "current-epoch", "down-after-milliseconds":
		// We don't use these keys
		return nil

	default:
		err := fmt.Errorf("Unhandled sentinel directive: %+v", entries)
		log.Print(err)
		return nil
	}
}

// LoadSentinelConfigFile loads the local config file pulled from the
// environment variable "REDSKULL_SENTINELCONFIGFILE"
func (c *Constellation) LoadSentinelConfigFile() error {
	file, err := os.Open(c.SentinelConfigName)
	if err != nil {
		log.Print(err)
		return err
	}
	defer file.Close()
	bf := bufio.NewReader(file)
	kvroot := fmt.Sprintf("redskull/constellations/primary") // this needs to be dynamic ones we've got a way to name sentinels
	common.StoreConfig.ConstellationBase = kvroot
	for {
		rawline, err := bf.ReadString('\n')
		if err == nil || err == io.EOF {
			line := strings.TrimSpace(rawline)
			// ignore comments
			if strings.Contains(line, "#") {
				continue
			}
			entries := strings.Split(line, " ")
			//Most values are key/value pairs
			switch entries[0] {
			case "sentinel": // Have a sentinel directive
				err := c.extractSentinelDirective(entries[1:])
				if err != nil {
					// TODO: Fix this to return a different error if we can't
					// connect to the sentinel
					log.Printf("Misshapen sentinel directive: '%s'", line)
				}
			case "port":
				iport, _ := strconv.Atoi(entries[1])
				c.SentinelConfig.Port = iport
			case "dir":
				c.SentinelConfig.Dir = entries[1]
			case "bind":
				if c.LocalOverrides.BindAddress > "" {
					log.Printf("Overriding Sentinel BIND directive '%s' with '%s'", entries[1], c.LocalOverrides.BindAddress)
					c.SentinelConfig.Host = c.LocalOverrides.BindAddress
				} else {
					c.SentinelConfig.Host = entries[1]
				}
				log.Printf("Local sentinel is listening on IP %s", c.SentinelConfig.Host)
			case "":
				if err == io.EOF {
					log.Print("File load complete?")
					if c.Name == "" {
						c.Name = fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
					}
					if c.LocalOverrides.BindAddress > "" {
						c.SentinelConfig.Host = c.LocalOverrides.BindAddress
						log.Printf("Local sentinel is listening on IP %s", c.SentinelConfig.Host)
					}
					keybase := fmt.Sprintf("%s/sentinels/%s", common.StoreConfig.ConstellationBase, c.Name)
					common.StoreConfig.SentinelBase = keybase
					podbase := fmt.Sprintf("%s/pods", common.StoreConfig.ConstellationBase)
					common.StoreConfig.Podbase = podbase
					log.Printf("Podbase: %s", podbase)
					common.Backingstore.Put(keybase+"/ip", []byte(c.SentinelConfig.Host), nil)
					common.Backingstore.Put(keybase+"/port", []byte(fmt.Sprintf("%d", c.SentinelConfig.Port)), nil)
					return nil
				}
			default:
				log.Printf("UNhandled Sentinel Directive: %s", line)
			}
		} else {
			log.Print("=============== LOAD FILE ERROR ===============")
			log.Fatal(err)
		}
	}
}

// ResetPod this is the constellation cluster level call to issue a reset
// against the sentinels for the given pod.
func (c *Constellation) ResetPod(podname string, simultaneous bool) {
	sentinels := c.GetSentinelsForPod(podname)
	log.Printf("Calling reset on %d sentinels for pod '%s'", len(sentinels), podname)
	if len(sentinels) == 0 {
		log.Print("ERROR: Attempt to call reset on pod with no sentinels??:" + podname)
		return
	}
	for _, sentinel := range sentinels {
		log.Print("Issuing reset for " + podname)
		if simultaneous {
			go sentinel.ResetPod(podname)
		} else {
			sentinel.ResetPod(podname)
			time.Sleep(2 * time.Second)
		}
	}
	c.GetAllSentinelsQuietly()
}

// By is a convenience type to enable sorting sentinels by their
// monitored pod count
type By func(s1, s2 *Sentinel) bool

// Sort sorts sentinels using sentinelSorter
func (by By) Sort(sentinels []*Sentinel) {
	ss := &sentinelSorter{
		sentinels: sentinels,
		by:        by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ss)
}

// sentinelSorter sorts sentinels by ther "length"
type sentinelSorter struct {
	sentinels []*Sentinel
	by        func(s1, s2 *Sentinel) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *sentinelSorter) Len() int {
	return len(s.sentinels)
}

// Swap is part of sort.Interface.
func (s *sentinelSorter) Swap(i, j int) {
	s.sentinels[i], s.sentinels[j] = s.sentinels[j], s.sentinels[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *sentinelSorter) Less(i, j int) bool {
	return s.by(s.sentinels[i], s.sentinels[j])
}

// GetAddressPair is a convenience function for converting an ip and port
// into the ip:port string. Probably need to move this to the common
// package
func GetAddressPair(astring string) (host string, port int, err error) {
	apair := strings.Split(astring, ":")
	host = apair[0]
	port, err = strconv.Atoi(apair[1])
	if err != nil {
		log.Printf("Unable to convert %s to port integer!", apair[1])
	}
	return
}

//ValidatePodSentinels will attempt to connect to each sentinel listed for a pod
// and pull the master info from it. This is to validate we can 1) connect to
// it, and 2) it actually has the pod in it's list
func (c *Constellation) ValidatePodSentinels(podname string) (map[string]bool, error) {
	_, exists := c.PodMap[podname]
	checks := make(map[string]bool)
	if !exists {
		return checks, errors.New("Pod not found")
	}
	allvalid := true
	//PodToSentinelsMap   map[string][]*Sentinel
	for _, s := range c.PodToSentinelsMap[podname] {
		sname := s.Name
		sc, err := client.DialWithConfig(&client.DialConfig{Address: s.Name})
		if err != nil {
			checks[sname] = false
			allvalid = false
			continue
		}
		_, err = sc.SentinelGetMaster(podname)
		if err != nil {
			checks[sname] = false
			allvalid = false
			continue
		}
		checks[sname] = true
	}
	if !allvalid {
		return checks, errors.New("Not all sentinels validated")
	}
	return checks, nil
}
