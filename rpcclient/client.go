package rsclient

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"time"

	"github.com/therealbill/redskull/actions"
)

type (
	Client struct {
		connection *rpc.Client
	}
)

type NewPodRequest struct {
	Name   string
	IP     string
	Port   int
	Quorum int
	Auth   string
}

func NewClient(dsn string, timeout time.Duration) (*Client, error) {
	connection, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{connection: rpc.NewClient(connection)}, nil
}

func (c *Client) GetSentinelsForPod(address string) (int, []string, error) {
	var scount int
	var sentinels []string
	err := c.connection.Call("RPC.GetSentinelsForPod", address, &sentinels)
	if err != nil {
		log.Print(err)
	} else {
		scount = len(sentinels)
	}
	return scount, sentinels, err
}

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

func (c *Client) AddPod(name, ip string, port, quorum int, auth string) (actions.RedisPod, error) {
	var pod actions.RedisPod
	pr := NewPodRequest{Name: name, IP: ip, Port: port, Quorum: quorum, Auth: auth}
	err := c.connection.Call("RPC.AddPod", pr, &pod)
	if err != nil {
		log.Print(err)
	}
	return pod, err

}

func (c *Client) GetPod(podname string) (actions.RedisPod, error) {
	var pod actions.RedisPod
	err := c.connection.Call("RPC.GetPod", podname, &pod)
	if err != nil {
		log.Print(err)
	}
	return pod, err
}

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
