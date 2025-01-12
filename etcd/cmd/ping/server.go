package main

import (
	_ "net/http/pprof"

	cli "github.com/urfave/cli/v2"
)

var serveCmd = &cli.Command{
	Name: "serve",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "listen-address",
			Usage:   "Listen address",
			Value:   "0.0.0.0:8200",
			EnvVars: []string{"PING_LISTEN_ADDRESS"},
		},
		&cli.StringFlag{
			Name:    "metrics-listen-address",
			Usage:   "listen endpoint for metrics and pprof",
			Value:   "0.0.0.0:8201",
			EnvVars: []string{"PING_METRICS_LISTEN_ADDRESS"},
		},
	},
	Action: func(cctx *cli.Context) error {
		// ctx := cctx.Context
		// // Flags
		// opt := struct {
		// 	ListenAddress        string
		// 	MetricsListenAddress string
		// 	DiscoveryKeyPrefix   string
		// 	DiscoveryAddresses   []string
		// }{
		// 	ListenAddress:        cctx.String("listen-address"),
		// 	MetricsListenAddress: cctx.String("metrics-listen-address"),
		// 	DiscoveryKeyPrefix:   cctx.String("discovery-key-prefix"),
		// 	DiscoveryAddresses:   cctx.StringSlice("discovery-addresses"),
		// }

		return nil
	},
}
