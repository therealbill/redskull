package rsclient

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"time"

	"github.com/therealbill/redskull/redskull-controller/common"
)

type Client struct {
	connection *rpc.Client
}

// NewPodRequest is a struct used for passing in the pod information from the
// client
type NewPodRequest struct {
	Name   string
	IP     string
	Port   int
	Quorum int
	Auth   string
}

// AddSlaveToPodRequest is a struct for passing slave+pod information over the
// wire
type AddSlaveToPodRequest struct {
	Pod       string
	SlaveIP   string
	SlavePort int
	SlaveAuth string
}

// NewClient returns a client connection
func NewClient(dsn string, timeout time.Duration) (*Client, error) {
	connection, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{connection: rpc.NewClient(connection)}, nil
}

//CheckPodAuth has the server check it's authenticationn capability to the
//master and all attached slaves for the pod. It returns a map true/false for
//each IP in the pod. If the master can not be authed against it returns an
//empty map.
func (c *Client) CheckPodAuth(podname string) (map[string]bool, error) {
	authmap := make(map[string]bool)
	if c == nil {
		log.Fatal("c is nil! This means you didn't call NewClient to get a working client connection")
	}
	err := c.connection.Call("RPC.CheckPodAuth", podname, &authmap)
	if err != nil {
		log.Print(err)
		return authmap, err
	}
	return authmap, nil
}

//AddSlaveToPod is used to instruct Red skull to add a slave to the given pod.
func (c *Client) AddSlaveToPod(podname, slaveip string, slaveport int, slaveauth string) (bool, error) {
	nsr := AddSlaveToPodRequest{Pod: podname, SlaveIP: slaveip, SlavePort: slaveport, SlaveAuth: slaveauth}
	var added bool
	err := c.connection.Call("RPC.AddSlaveToPod", nsr, &added)
	if err != nil {
		log.Print(err)
		return false, err
	}
	return true, nil
}

// GetSentinelsForPod(podname)  returns the number and list of sentinels for
// the given podname
func (c *Client) GetSentinelsForPod(address string) (int, []string, error) {
	var sc int
	var sentinels []string
	err := c.connection.Call("RPC.GetSentinelsForPod", address, &sentinels)
	if err != nil {
		log.Print(err)
	} else {
		sc = len(sentinels)
	}
	return sc, sentinels, err
}

// AddSentinel(address) will instuct Redskull to add the sentinel at the given
// address
func (c *Client) AddSentinel(address string) (bool, error) {
	var ok bool
	err := c.connection.Call("RPC.AddSentinel", address, &ok)
	if err != nil {
		log.Print(err)
	} else {
		ok = true
	}
	return ok, err
}

// AddPod(NewPodRequest) will take the information in the PodRequest and
// instruct Redskull to add it to it's monitor list.
func (c *Client) AddPod(name, ip string, port, quorum int, auth string) (common.RedisPod, error) {
	var pod common.RedisPod
	pr := NewPodRequest{Name: name, IP: ip, Port: port, Quorum: quorum, Auth: auth}
	err := c.connection.Call("RPC.AddPod", pr, &pod)
	if err != nil {
		log.Print(err)
	}
	return pod, err

}

// GetPod(podname) will return the common.RedisPod type for the given pod, if
// found.
func (c *Client) GetPod(podname string) (common.RedisPod, error) {
	var pod common.RedisPod
	err := c.connection.Call("RPC.GetPod", podname, &pod)
	if err != nil {
		log.Print(err)
	}
	return pod, err
}

//RemovePod(podname) removes the prod from Redskull and associated sentinels
func (c *Client) RemovePod(podname string) error {
	ok := false
	err := c.connection.Call("RPC.RemovePod", podname, &ok)
	if err != nil {
		log.Printf("ok=%s,Error: %s", ok, err.Error())
		return err
	}
	if !ok {
		return errors.New("Unable to remove pod")
	}
	return nil
}

//BalancePod will attempt to rebalance the pod's sentinels
func (c *Client) BalancePod(podname string) error {
	ok := false
	err := c.connection.Call("RPC.BalancePod", podname, &ok)
	if !ok {
		if err != nil {
			log.Printf("Error: %s", err.Error())
		}
		return errors.New("Was unable to initiate balance operation")
	}
	return nil
}

//ValidatePodSentinels validates the sentinels listed for the given pod.
func (c *Client) ValidatePodSentinels(podname string) (map[string]bool, error) {
	checks := make(map[string]bool)
	return checks, nil
}
