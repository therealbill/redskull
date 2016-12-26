package lib

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const GCPORT = "8008"

// SentinelPodConfig is a struct carrying information about a Pod's config as
// pulled from the sentinel config file.
type SentinelPodConfig struct {
	IP        string
	Port      int
	Name      string
	AuthToken string
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
	NumErrorPods        int
	Connected           bool
	LocalSentinel       Sentinel
	SentinelConfigName  string
	SentinelConfig      LocalSentinelConfig
	PodAuthMap          map[string]string
	ConfiguredSentinels map[string]interface{}
	CellName            string
}
type SentinelOverrides struct {
	BindAddress string
}

// GetConstellation returns an instance of a constellation. It requires the
// configuration and a cell name. The cell name identifies the cluster the
// constellation, and hence this RedSkull instance, belongs to.
// In the future this will be used in clsuter coordination as well as for a
// protective measure against cluster merge
func GetConstellation(name, cfg, cell, sentinelAddress string) (Constellation, error) {
	con := Constellation{Name: name}
	con.SentinelConfig.ManagedPodConfigs = make(map[string]SentinelPodConfig)
	con.PodMap = make(map[string]*RedisPod)
	con.CellName = cell
	con.SentinelConfigName = cfg
	con.LoadSentinelConfigFile()
	return con, nil
}

// GetPodAuth will return an authstring from the local config
func (c *Constellation) GetPodAuth(podname string) string {
	return c.PodMap[podname].AuthToken
}

// extractSentinelDirective parses the sentinel directives from the
// sentinel config file
func (c *Constellation) extractSentinelDirective(entries []string) error {
	switch entries[0] {
	case "monitor":
		pname := entries[1]
		port, _ := strconv.Atoi(entries[3])
		spc := SentinelPodConfig{Name: pname, IP: entries[2], Port: port}
		// normally we should not see duplicate IP:PORT combos, however it
		// can happen when people do things manually and dont' clean up.
		// We need to detect them and ignore the second one if found,
		// reporting the error condition this will require tracking
		// ip:port pairs...
		addr := fmt.Sprintf("%s:%d", entries[2], port)
		_, exists := c.SentinelConfig.ManagedPodConfigs[addr]
		if !exists {
			c.SentinelConfig.ManagedPodConfigs[entries[1]] = spc
		}
		log.Printf("read pod config: %+v", spc)
		return nil

	case "auth-pass":
		pname := entries[1]
		pc := c.SentinelConfig.ManagedPodConfigs[pname]
		pc.AuthToken = entries[2]
		c.SentinelConfig.ManagedPodConfigs[pname] = pc
		return nil

	case "known-sentinel":
		return nil

	case "known-slave":
		// Currently ignoring this, but may add call to a node manager.
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
				//log.Printf("Local sentinel is bound to port %d", c.SentinelConfig.Port)
			case "dir":
				c.SentinelConfig.Dir = entries[1]
			case "bind":
				log.Printf("Local sentinel is listening on IP %s", c.SentinelConfig.Host)
			case "":
				if err == io.EOF {
					log.Print("File load complete?")
					if c.Name == "" {
						c.Name = fmt.Sprintf("%s:%d", c.SentinelConfig.Host, c.SentinelConfig.Port)
					}
					return nil
				}
				//log.Printf("Local:config -> %+v", c.SentinelConfig)
				//log.Printf("Found %d REMOTE sentinels", len(c.RemoteSentinels))
				//return nil
			default:
				log.Printf("UNhandled Sentinel Directive: %s", line)
			}
		} else {
			log.Print("=============== LOAD FILE ERROR ===============")
			log.Fatal(err)
		}
	}
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
