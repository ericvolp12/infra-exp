package main

import (
	"fmt"
	"os"

	cli "github.com/urfave/cli/v2"
)

var (
	version = "unknown" // set by `go build`.
)

func main() {
	app := cli.App{
		Name:  "ping",
		Usage: "Demo ping service for etcd",
	}
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug mode",
			Value:   false,
			EnvVars: []string{"PING_DEBUG"},
		},
		&cli.StringFlag{
			Name:    "discovery-key-prefix",
			Usage:   "discovery key prefix",
			Value:   "pingservice",
			EnvVars: []string{"PING_DISCOVERY_KEY_PREFIX"},
		},
		&cli.StringSliceFlag{
			Name:    "discovery-addresses",
			Usage:   "comma-separated addresses of etcd GRPC servers to connect to for service discovery",
			Value:   cli.NewStringSlice("http://localhost:2480", "http://localhost:2481", "http://localhost:2482"),
			EnvVars: []string{"PING_DISCOVERY_ADDRESSES"},
		},
	}
	app.Commands = []*cli.Command{
		serveCmd,
		clientCmd,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}
}
