package client

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/therealbill/libredis/structures"
)

// buildSlaveInfoStruct builods the struct for a slave from the Redis slaves command
func (r *Redis) buildSlaveInfoStruct(info map[string]string) (master structures.SlaveInfo, err error) {
	s := reflect.ValueOf(&master).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		p := typeOfT.Field(i)
		f := s.Field(i)
		tag := p.Tag.Get("redis")
		if f.Type().Name() == "int" {
			val, err := strconv.ParseInt(info[tag], 10, 64)
			if err != nil {
				//println("Unable to convert to data from sentinel server:", info[tag], err)
			} else {
				f.SetInt(val)
			}
		}
		if f.Type().Name() == "string" {
			f.SetString(info[tag])
		}
		if f.Type().Name() == "bool" {
			// This handles primarily the xxx_xx style fields in the return data from redis
			if info[tag] != "" {
				val, err := strconv.ParseInt(info[tag], 10, 64)
				if err != nil {
					//println("[bool] Unable to convert to data from sentinel server:", info[tag], err)
					fmt.Println("Error:", err)
				} else {
					if val > 0 {
						f.SetBool(true)
					}
				}
			}
		}
	}
	return
}

// SentinelSlaves takes a podname and returns a list of SlaveInfo structs for
// each known slave.
func (r *Redis) SentinelSlaves(podname string) (slaves []structures.SlaveInfo, err error) {
	rp, err := r.ExecuteCommand("SENTINEL", "SLAVES", podname)
	if err != nil {
		fmt.Println("error on slaves command:", err)
		return
	}
	for i := 0; i < len(rp.Multi); i++ {
		slavemap, err := rp.Multi[i].HashValue()
		if err != nil {
			log.Println("unable to get slave info, err:", err)
		} else {
			info, err := r.buildSlaveInfoStruct(slavemap)
			if err != nil {
				fmt.Printf("Unable to get slaves, err: %s\n", err)
			}
			slaves = append(slaves, info)
		}
	}
	return
}

// SentinelMonitor executes the SENTINEL MONITOR command on the server
// This is used to add pods to the sentinel configuration
func (r *Redis) SentinelMonitor(podname string, ip string, port int, quorum int) (bool, error) {
	res, err := r.ExecuteCommand("SENTINEL", "MONITOR", podname, ip, port, quorum)
	if err != nil {
		return false, err
	}
	return res.BoolValue()
}

// SentinelRemove executes the SENTINEL REMOVE command on the server
// This is used to remove pods to the sentinel configuration
func (r *Redis) SentinelRemove(podname string) (bool, error) {
	res, err := r.ExecuteCommand("SENTINEL", "REMOVE", podname)
	rsp, _ := res.StatusValue()
	if rsp == "OK" {
		return true, err
	} else {
		return false, err
	}
}

func (r *Redis) SentinelReset(podname string) error {
	res, err := r.ExecuteCommand("SENTINEL", "RESET", podname)
	log.Print(res)
	log.Print(res)
	return err
}

// SentinelSetString will set the value of skey to sval for a
// given pod. This is used when the value is known to be a string
func (r *Redis) SentinelSetString(podname string, skey string, sval string) error {
	_, err := r.ExecuteCommand("SENTINEL", "SET", podname, skey, sval)
	return err
}

// SentinelSetInt will set the value of skey to sval for a
// given pod. This is used when the value is known to be an Int
func (r *Redis) SentinelSetInt(podname string, skey string, sval int) error {
	_, err := r.ExecuteCommand("SENTINEL", "SET", podname, skey, sval)
	return err
}

// SentinelSetPass will set the value to be used in the AUTH command for a
// given pod
func (r *Redis) SentinelSetPass(podname string, password string) error {
	_, err := r.ExecuteCommand("SENTINEL", "SET", podname, "AUTHPASS", password)
	return err
}

// SentinelSentinels returns the list of known Sentinels
func (r *Redis) SentinelSentinels(podName string) (sentinels []structures.SentinelInfo, err error) {
	reply, err := r.ExecuteCommand("SENTINEL", "SENTINELS", podName)
	if err != nil {
		log.Print("Err in sentinels command:", err)
		return
	}
	count := len(reply.Multi)
	for i := 0; i < count; i++ {
		data, err := reply.Multi[i].HashValue()
		if err != nil {
			log.Fatal("Error:", err)
		}
		sentinel, err := r.buildSentinelInfoStruct(data)
		sentinels = append(sentinels, sentinel)
	}
	return
}

// SentinelMasters returns the list of known pods
func (r *Redis) SentinelMasters() (masters []structures.MasterInfo, err error) {
	rp, err := r.ExecuteCommand("SENTINEL", "MASTERS")
	if err != nil {
		return
	}
	podcount := len(rp.Multi)
	for i := 0; i < podcount; i++ {
		pod, err := rp.Multi[i].HashValue()
		if err != nil {
			log.Fatal("Error:", err)
		}
		minfo, err := r.buildMasterInfoStruct(pod)
		masters = append(masters, minfo)
	}
	return
}

// SentinelMaster returns the master info for the given podname
func (r *Redis) SentinelMaster(podname string) (master structures.MasterInfo, err error) {
	rp, err := r.ExecuteCommand("SENTINEL", "MASTER", podname)
	if err != nil {
		return
	}
	pod, err := rp.HashValue()
	if err != nil {
		return
	}
	master, err = r.buildMasterInfoStruct(pod)
	return
}

func (r *Redis) buildSentinelInfoStruct(info map[string]string) (sentinel structures.SentinelInfo, err error) {
	s := reflect.ValueOf(&sentinel).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		p := typeOfT.Field(i)
		f := s.Field(i)
		tag := p.Tag.Get("redis")
		if f.Type().Name() == "int" {
			val, err := strconv.ParseInt(info[tag], 10, 64)
			if err != nil {
				//fmt.Println("Unable to convert to integer from sentinel server:", tag, info[tag], err)
			} else {
				f.SetInt(val)
			}
		}
		if f.Type().Name() == "string" {
			f.SetString(info[tag])
		}
		if f.Type().Name() == "bool" {
			// This handles primarily the xxx_xx style fields in the return data from redis
			if info[tag] != "" {
				val, err := strconv.ParseInt(info[tag], 10, 64)
				if err != nil {
					//println("Unable to convert to bool from sentinel server:", info[tag])
					fmt.Println(info[tag])
					fmt.Println("Error:", err)
				} else {
					if val > 0 {
						f.SetBool(true)
					}
				}
			}
		}
	}
	return
}

func (r *Redis) buildMasterInfoStruct(info map[string]string) (master structures.MasterInfo, err error) {
	s := reflect.ValueOf(&master).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		p := typeOfT.Field(i)
		f := s.Field(i)
		tag := p.Tag.Get("redis")
		if f.Type().Name() == "int" {
			if info[tag] > "" {
				val, err := strconv.ParseInt(info[tag], 10, 64)
				if err != nil {
					fmt.Println("Unable to convert to integer from sentinel server:", tag, info[tag], err)
				} else {
					f.SetInt(val)
				}
			}
		}
		if f.Type().Name() == "string" {
			f.SetString(info[tag])
		}
		if f.Type().Name() == "bool" {
			// This handles primarily the xxx_xx style fields in the return data from redis
			if info[tag] != "" {
				val, err := strconv.ParseInt(info[tag], 10, 64)
				if err != nil {
					//println("Unable to convert to bool from sentinel server:", info[tag])
					fmt.Println(info[tag])
					fmt.Println("Error:", err)
				} else {
					if val > 0 {
						f.SetBool(true)
					}
				}
			}
		}
	}
	return
}

// SentinelMasterInfo returns the information about a pod or master
func (r *Redis) SentinelMasterInfo(podname string) (master structures.MasterInfo, err error) {
	rp, err := r.ExecuteCommand("SENTINEL", "MASTER", podname)
	if err != nil {
		return master, err
	}
	info, err := rp.HashValue()
	return r.buildMasterInfoStruct(info)
}

// SentinelGetMaster returns the information needed to connect to the master of
// a given pod
func (r *Redis) SentinelGetMaster(podname string) (conninfo structures.MasterAddress, err error) {
	rp, err := r.ExecuteCommand("SENTINEL", "get-master-addr-by-name", podname)
	if err != nil {
		return conninfo, err
	}
	info, err := rp.ListValue()
	if len(info) != 0 {
		conninfo.Host = info[0]
		conninfo.Port, err = strconv.Atoi(info[1])
		if err != nil {
			fmt.Println("Got bad port info from server, causing err:", err)
		}
	}
	return conninfo, err
}

func (r *Redis) SentinelFailover(podname string) (bool, error) {
	rp, err := r.ExecuteCommand("SENTINEL", "failover", podname)
	if err != nil {
		log.Println("Error on failover command execution:", err)
		return false, err
	}

	if rp.Error != "" {
		log.Println("Error on failover command execution:", rp.Error)
		return false, fmt.Errorf(rp.Error)
	}
	return true, nil
}

func (r *Redis) GetPodName() (name string, err error) {
	// we can get this by subscribing to the sentinel channel and getting a hello message
	// TODO: Will need to update this when I make PubSub message structs
	maxmsgcnt := 10
	p, _ := r.PubSub()
	p.Subscribe("__sentinel__:hello")
	var msg []string
	for x := 0; x < maxmsgcnt; x++ {
		msg, err = p.Receive()
		if err != nil {
			return name, err
		}
		switch msg[0] {
		case "message":
			fields := strings.Split(msg[2], ",")
			name = fields[4]
			return name, nil
		}
	}
	return name, errors.New("No messages contained the needed hello")
}

func (r *Redis) GetKnownSentinels() (sentinels map[string]string, err error) {
	maxmsgcnt := 10
	p, _ := r.PubSub()
	p.Subscribe("__sentinel__:hello")
	sentinels = make(map[string]string)
	var msg []string
	for x := 0; x < maxmsgcnt; x++ {
		msg, err = p.Receive()
		if err != nil {
			return
		}
		switch msg[0] {
		case "message":
			fields := strings.Split(msg[2], ",")
			//name = fields[4]
			ip := fields[0]
			port := fields[1]
			//sentinelid := fields[2]
			sentinels[ip] = port
		}
	}
	if len(sentinels) == 0 {
		return sentinels, errors.New("No messages contained the needed hello")
	} else {
		return sentinels, nil
	}
}
