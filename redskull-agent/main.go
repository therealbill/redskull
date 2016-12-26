package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli"
)

var (
	Build string
	key   string
	app   *cli.App
	agent RedAgentService
)

type ConstellationConfig struct {
	Nodes           []string
	DeadNodes       []string
	ConnectionCount int64
}

func RefreshData() {
	//t := time.Tick(60 * time.Second)
	//context, err := handlers.NewPageContext()
	//if err != nil {
	//log.Fatal("[RefreshData]", err)
	//}
	//mc := context.Constellation
	//if err != nil {
	//log.Fatal("Unable to connect to constellation")
	//}
	log.Printf("TODO: Refreshing routine needs rebuilt")
}

type LaunchConfig struct {
	GroupName           string
	NodeRefreshInterval time.Duration
}

func init() {
	//key = os.Getenv("AIRBRAKE_API_KEY")
	//airbrake.Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"
	//airbrake.ApiKey = key
	//airbrake.Environment = os.Getenv("RSM_ENVIRONMENT")
	//if len(airbrake.Environment) == 0 {
	//airbrake.Environment = "Development"
	//}
}

func main() {
	app = cli.NewApp()
	app.Name = "redskull-agent"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "a,consuladdr",
			Usage:  "Consul server address",
			Value:  "localhost:8500",
			EnvVar: "CONSUL_ADDR",
		},
		cli.StringFlag{
			Name:   "c,cellname",
			Usage:  "Redskull cell name",
			Value:  "localhost:8500",
			EnvVar: "REDSKULL_CELL",
		},
	}
	// TODO: add commands to be used by the local sentinel event handler, and
	// add self to config for each pod on the sentinel
	app.Commands = []cli.Command{
		{
			Name:   "serve",
			Usage:  "run the RPC service",
			Action: runRPCServer,
		},
	}
	app.Run(os.Args)

}
func runRPCServer(c *cli.Context) error {
	cell := c.String("cellname")
	if cell == "" {
		cell = "cell0"
	}
	agent = NewRedAgentService(cell)
	agent.ServeRPC()
	return nil
}
