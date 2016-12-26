package main // import "github.com/therealbill/redskull/redskull-controller"

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/therealbill/airbrake-go"
	"github.com/therealbill/redskull/redskull-controller/actions"
	"github.com/therealbill/redskull/redskull-controller/handlers"
	"github.com/zenazn/goji"
)

var Build string
var key string

type ConstellationConfig struct {
	Nodes           []string
	DeadNodes       []string
	ConnectionCount int64
}

func RefreshData() {
	t := time.Tick(60 * time.Second)
	context, err := handlers.NewPageContext()
	if err != nil {
		log.Fatal("[RefreshData]", err)
	}
	mc := context.Constellation
	if err != nil {
		log.Fatal("Unable to connect to constellation")
	}
	for _ = range t {
		mc.LoadSentinelConfigFile()
		mc.GetAllSentinels()
		for _, pod := range mc.RemotePodMap {
			_, _ = mc.LocalPodMap[pod.Name]
			auth := mc.GetPodAuth(pod.Name)
			if pod.AuthToken != auth && auth > "" {
				pod.AuthToken = auth
			}
		}
		for _, pod := range mc.LocalPodMap {
			pod.AuthToken = mc.GetPodAuth(pod.Name)
		}
		mc.IsBalanced()
		log.Printf("Main Cache Stats: %+v", mc.AuthCache.GetStats())
		log.Printf("Hot Cache Stats: %+v", mc.AuthCache.GetHotStats())
	}
}

type LaunchConfig struct {
	Name                string
	Port                int
	IP                  string
	SentinelConfigFile  string
	GroupName           string
	BindAddress         string
	SentinelHostAddress string
	TemplateDirectory   string
	NodeRefreshInterval float64
	RPCPort             int
}

var config LaunchConfig

func init() {
	err := envconfig.Process("redskull", &config)
	if err != nil {
		log.Fatal(err)
	}
	if config.NodeRefreshInterval == 0 {
		config.NodeRefreshInterval = 60
	}
	actions.NodeRefreshInterval = config.NodeRefreshInterval

	log.Printf("Launch Config: %+v", config)
	if config.BindAddress > "" {
		flag.Set("bind", config.BindAddress)
	} else {
		if config.Port == 0 {
			log.Print("ENV contained no port, using default")
			config.Port = 8000
		}
	}

	if config.RPCPort == 0 {
		config.RPCPort = config.Port + 1
	}

	ps := fmt.Sprintf("%s:%d", config.IP, config.Port)
	log.Printf("binding to '%s'", ps)
	flag.Set("bind", ps)

	if config.TemplateDirectory > "" {
		if !strings.HasSuffix(config.TemplateDirectory, "/") {
			config.TemplateDirectory += "/"
		}
	}
	handlers.TemplateBase = config.TemplateDirectory

	// handle absent sentinel config file w/a default
	if config.SentinelConfigFile == "" {
		log.Print("ENV contained no SentinelConfigFile, using default")
		config.SentinelConfigFile = "/etc/redis/sentinel.conf"
	}

	// handle absent sentinel config file w/a default
	if config.GroupName == "" {
		config.GroupName = "redskull:1"
		log.Print("ENV contained no GroupName, using default:" + config.GroupName)
	}

	config_json, _ := json.Marshal(config)
	log.Printf("Config: %s", config_json)

	key = os.Getenv("AIRBRAKE_API_KEY")
	airbrake.Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"
	airbrake.ApiKey = key
	airbrake.Environment = os.Getenv("RSM_ENVIRONMENT")
	if len(airbrake.Environment) == 0 {
		airbrake.Environment = "Development"
	}
	if len(Build) == 0 {
		Build = ".1"
		return
	}
}

func main() {
	mc, err := actions.GetConstellation(config.Name, config.SentinelConfigFile, config.GroupName, config.SentinelHostAddress)
	if err != nil {
		log.Fatal("Unable to connect to constellation")
	}
	//log.Print("Starting refresh ticker")
	//go RefreshData()
	_, _ = mc.GetPodMap()
	mc.IsBalanced()
	//mc = mc
	//_ = handlers.NewPageContext()
	if mc.AuthCache == nil {
		log.Print("Uninitialized AuthCache, StartCache not called, calling now")
		mc.StartCache()
	}
	log.Printf("Main Cache Stats: %+v", mc.AuthCache.GetStats())
	log.Printf("Hot Cache Stats: %+v", mc.AuthCache.GetHotStats())
	handlers.SetConstellation(mc)

	go ServeRPC()

	// HTML Interface URLS
	goji.Get("/constellation/", handlers.ConstellationInfoHTML) // Needs moved? instance tree?
	goji.Get("/dashboard/", handlers.Dashboard)                 // Needs moved? instance tree?
	goji.Get("/constellation/addpodform/", handlers.AddPodForm)
	goji.Post("/constellation/addpod/", handlers.AddPodHTML)
	goji.Post("/constellation/addsentinel/", handlers.AddSentinelHTML)
	goji.Get("/constellation/addsentinelform/", handlers.AddSentinelForm)
	goji.Get("/constellation/rebalance/", handlers.RebalanceHTML)
	goji.Get("/constellation/removepod/:podname", handlers.RemovePodHTML)
	//goji.Get("/pod/:podName/dropslave", handlers.DropSlaveHTML)
	goji.Get("/pod/:podName/addslave", handlers.AddSlaveHTML)
	goji.Post("/pod/:podName/addslave", handlers.AddSlaveHTMLProcessor)
	goji.Post("/pod/:name/failover", handlers.DoFailoverHTML)
	goji.Post("/pod/:name/reset", handlers.ResetPodProcessor)
	goji.Post("/pod/:name/balance", handlers.BalancePodProcessor)
	goji.Get("/pod/:podName", handlers.ShowPod)
	goji.Get("/pods/", handlers.ShowPods)
	goji.Get("/nodes/", handlers.ShowNodes)
	goji.Get("/node/:name", handlers.ShowNode)
	goji.Get("/", handlers.Root) // Needs moved? instance tree?

	// API URLS
	goji.Get("/api/knownpods", handlers.APIGetPods)
	goji.Put("/api/monitor/:podName", handlers.APIMonitorPod)
	goji.Post("/api/constellation/:podName/failover", handlers.APIFailover)

	goji.Get("/api/pod/:podName", handlers.APIGetPod)
	goji.Put("/api/pod/:podName", handlers.APIMonitorPod)
	goji.Put("/api/pod/:podName/addslave", handlers.APIAddSlave)
	goji.Delete("/api/pod/:podName", handlers.APIRemovePod)
	goji.Get("/api/pod/:podName/master", handlers.APIGetMaster)
	goji.Get("/api/pod/:podName/slaves", handlers.APIGetSlaves)

	goji.Post("/api/node/clone", handlers.Clone) // Needs moved to the node tree
	goji.Get("/api/node/:name", handlers.GetNodeJSON)

	goji.Get("/static/*", handlers.Static) // Needs moved? instance tree?
	//goji.Abandon(middleware.Logger)

	goji.Serve()

}
