package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/therealbill/redskull/redskull-agent/rpcclient"
	"github.com/urfave/cli"
	//rsclib"github.com/therealbill/redskull/redskull-agent/lib"
)

var (
	app *cli.App
)

func main() {
	app = cli.NewApp()
	app.Name = "redskull-agent-client-example"
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
	app.Commands = []cli.Command{
		{
			Name:   "showPod",
			Usage:  "show given pod",
			Action: showPod,
		},
	}
	app.Run(os.Args)
}

func showPod(c *cli.Context) error {
	//rsSvc,err :=  rsclib.NewRemoteService("redskull-agent")
	client, err := rsclient.NewClient("127.0.0.1:11000", time.Duration(15)*time.Second)
	if err != nil {
		return err
	}
	pn := c.Args()[0]
	token, err := client.GetPodAuth(pn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Pod '%s'\n", pn)
	fmt.Printf(" Auth: %+v\n", token)
	return err
}
