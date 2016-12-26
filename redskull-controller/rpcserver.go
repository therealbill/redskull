package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"time"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/redskull/redskull-controller/actions"
	"github.com/therealbill/redskull/redskull-controller/common"
	"github.com/therealbill/redskull/redskull-controller/handlers"
	"github.com/therealbill/redskull/redskull-controller/rpcclient"
)

type Item struct {
	key  string
	Item interface{}
}

type RPC struct {
	constellation *actions.Constellation
	mu            *sync.RWMutex
}

func badContextError(err error) {
	if err != nil {
		panic("Constellation is not initialized")
	}
}

// AddSlaveToPod is used for adding a new slave to an existing pod.
// TODO: technically the actual implementation should be moved into the actions
// package and the UI's handlers package can then also call it. As it is, it is
// also implemented there.
func (r *RPC) AddSlaveToPod(nsr rsclient.AddSlaveToPodRequest, resp *bool) error {
	pod, err := r.constellation.GetPod(nsr.Pod)
	if err != nil {
		return errors.New("Pod not found")
	}
	name := fmt.Sprintf("%s:%d", nsr.SlaveIP, nsr.SlavePort)
	new_slave, err := client.DialWithConfig(&client.DialConfig{Address: name, Password: nsr.SlaveAuth})
	defer new_slave.ClosePool()
	if err != nil {
		log.Print("ERR: Dialing slave -", err)
		return errors.New("Server was unable to connect to slave")
	}
	err = new_slave.SlaveOf(pod.Info.IP, fmt.Sprintf("%d", pod.Info.Port))
	if err != nil {
		log.Printf("Err: %v", err)
		if strings.Contains(err.Error(), "Already connected to specified master") {
			return errors.New("Already connected to specified master")
		}
	}

	new_slave.ConfigSet("masterauth", pod.AuthToken)
	new_slave.ConfigSet("requirepass", pod.AuthToken)
	pod.Master.LastUpdateValid = false
	r.constellation.PodMap[pod.Name] = pod
	*resp = true
	return err
}

func (r *RPC) CheckPodAuth(podname string, resp *map[string]bool) error {
	pod, err := r.constellation.GetPod(podname)
	if err != nil || pod == nil {
		log.Print("No pod. Error: ", err)
		return err
	}
	psresults := make(map[string]bool)
	if pod.Master == nil {
		mnode, err := common.LoadNodeFromHostPort(pod.Master.Address, pod.Master.Port, pod.AuthToken)
		if err != nil {
			log.Print("Connection error: ", err)
			return errors.New("Unable to connect to master nod at all. Check server logs for why")
		}
		pod.Master = mnode
	}
	mres := pod.Master.Ping()
	psresults[pod.Master.Name] = mres
	for _, slave := range pod.Master.Slaves {
		log.Printf("Checking ping/auth for slave %s", slave.Name)
		sres := slave.Ping()
		psresults[slave.Name] = sres
	}
	*resp = psresults
	return nil
}

func (r *RPC) GetSentinelsForPod(podname string, resp *[]string) error {
	sentinels := r.constellation.GetSentinelsForPod(podname)
	var snames []string
	for _, s := range sentinels {
		snames = append(snames, s.Name)
	}
	*resp = snames
	return nil
}

func (r *RPC) AddPod(pr rsclient.NewPodRequest, resp *common.RedisPod) (err error) {
	gob.Register(common.RedisPod{})
	ok, err := r.constellation.MonitorPod(pr.Name, pr.IP, pr.Port, pr.Quorum, pr.Auth)
	if err != nil {
		log.Printf("MonitorPod call ('%+v') Failed. Error: %s", pr, err.Error())
		return err
	}
	if !ok {
		log.Printf("MonitorPod call ('%+v') Failed. No Error", pr)
		err = errors.New("MonitorPod call returned false, no error")
	}
	time.Sleep(time.Second * 2)
	pod, err := r.constellation.GetPod(pr.Name)
	if pod == nil || pod.Name == "" {
		err = errors.New("New Pod Not found")
		return err
	}
	*resp = *pod
	return err
}

func (r *RPC) GetPod(podname string, resp *common.RedisPod) (err error) {
	gob.Register(common.RedisPod{})
	pod, err := r.constellation.GetPod(podname)
	if pod == nil || pod.Name == "" {
		err = errors.New("Pod Not found")
		return err
	}
	*resp = *pod
	return err
}

func (r *RPC) RemovePod(podname string, resp *bool) (err error) {
	ok, err := r.constellation.RemovePod(podname)
	*resp = ok
	return err
}

func (r *RPC) AddSentinel(address string, resp *bool) (err error) {
	err = r.constellation.AddSentinelByAddress(address)
	if err == nil {
		*resp = true
	}
	return err
}

func (r *RPC) BalancePod(podname string, resp *bool) (err error) {
	pod, err := r.constellation.GetPod(podname)
	if pod == nil || pod.Name == "" {
		err = errors.New("Pod Not found")
		*resp = false
	}
	r.constellation.BalancePod(pod)
	*resp = true
	return err
}

func (r *RPC) ValidatePodSentinels(podname string, resp *map[string]bool) error {
	pod, err := r.constellation.GetPod(podname)
	res := make(map[string]bool)
	if pod == nil || pod.Name == "" {
		err = errors.New("Pod Not found")
		*resp = res
		return err
	}
	results, err := r.constellation.ValidatePodSentinels(podname)
	*resp = results
	return err

}

func NewRPC() *RPC {
	context, err := handlers.NewPageContext()
	badContextError(err)
	return &RPC{
		constellation: context.Constellation,
	}
}

func (r *RPC) GetPodList(verbose bool, resp *[]string) (err error) {
	var podlist []string
	for k, _ := range r.constellation.PodMap {
		if verbose {
			log.Printf("found pod %s", k)
		}
		podlist = append(podlist, k)
	}
	*resp = podlist
	return nil
}

func ServeRPC() {
	rpc.Register(NewRPC())
	rpc_on := fmt.Sprintf("%s:%d", config.BindAddress, config.RPCPort)
	l, e := net.Listen("tcp", rpc_on)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	rpc.Accept(l)
}
