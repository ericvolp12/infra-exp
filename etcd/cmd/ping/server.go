package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	cli "github.com/urfave/cli/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
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
			Name:    "external-address",
			Usage:   "External address for clients to connect to",
			Value:   "http://localhost:8200",
			EnvVars: []string{"PING_EXTERNAL_ADDRESS"},
		},
		&cli.StringFlag{
			Name:    "metrics-listen-address",
			Usage:   "listen endpoint for metrics and pprof",
			Value:   "0.0.0.0:8300",
			EnvVars: []string{"PING_METRICS_LISTEN_ADDRESS"},
		},
	},
	Action: func(cctx *cli.Context) error {
		// Flags
		opt := struct {
			ListenAddress        string
			ExternalAddress      string
			MetricsListenAddress string
			DiscoveryKeyPrefix   string
			DiscoveryAddresses   []string
		}{
			ListenAddress:        cctx.String("listen-address"),
			ExternalAddress:      cctx.String("external-address"),
			MetricsListenAddress: cctx.String("metrics-listen-address"),
			DiscoveryKeyPrefix:   cctx.String("discovery-key-prefix"),
			DiscoveryAddresses:   cctx.StringSlice("discovery-addresses"),
		}

		s, err := NewServer(opt.DiscoveryKeyPrefix, opt.DiscoveryAddresses, opt.ListenAddress, opt.ExternalAddress, opt.MetricsListenAddress)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		// Start the server.
		serverClosed := make(chan struct{})
		go func() {
			if err := s.Run(context.Background()); err != nil {
				slog.Error("server run failed", "err", err)
			}
			close(serverClosed)
		}()

		quit := make(chan struct{})
		exitSignals := make(chan os.Signal, 1)
		signal.Notify(exitSignals, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			select {
			case sig := <-exitSignals:
				slog.Info("received OS exit signal", "signal", sig)
			case <-serverClosed:
				slog.Info("server closed, exiting")
			}

			// Create a context with a 5 second timeout.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Shutdown the RPC server.
			if err := s.Shutdown(ctx); err != nil {
				slog.Warn("rpc server shutdown failed", "err", err)
			}

			// Trigger the return that causes an exit.
			close(quit)
		}()

		<-quit
		slog.Info("graceful shutdown complete")
		return nil

	},
}

type Server struct {
	DiscoveryKeyPrefix   string
	DiscoveryAddresses   []string
	ListenAddress        string
	ExternalAddress      string
	MetricsListenAddress string
	discoveryClient      *clientv3.Client
	shutdown             chan chan struct{}
}

func NewServer(discoveryKeyPrefix string, discoveryAddresses []string, listenAddress, externalAddress, metricsListenAddress string) (*Server, error) {
	s := Server{
		DiscoveryKeyPrefix:   discoveryKeyPrefix,
		DiscoveryAddresses:   discoveryAddresses,
		ListenAddress:        listenAddress,
		ExternalAddress:      externalAddress,
		MetricsListenAddress: metricsListenAddress,
		shutdown:             make(chan chan struct{}),
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   discoveryAddresses,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	s.discoveryClient = cli

	return &s, nil
}

func (s *Server) Run(ctx context.Context) error {
	errGrp := errgroup.Group{}

	// Start metrics server
	errGrp.Go(func() error {
		mux := http.DefaultServeMux
		mux.Handle("/metrics", promhttp.Handler())
		slog.Info("Starting metrics server", "address", s.MetricsListenAddress)
		err := http.ListenAndServe(s.MetricsListenAddress, mux)
		if err != nil {
			slog.Error("Failed to start metrics server", "err", err)
		}
		return err
	})

	// Start ping server
	errGrp.Go(func() error {
		mux := http.NewServeMux()
		mux.HandleFunc("/ping", s.HandlePing)
		slog.Info("Starting ping server", "address", s.ListenAddress)
		err := http.ListenAndServe(s.ListenAddress, mux)
		if err != nil {
			slog.Error("Failed to start ping server", "err", err)
		}
		return err
	})

	// Register with discovery service
	errGrp.Go(func() error {
		slog.Info("Registering with discovery service", "address", s.ExternalAddress)
		// Create a 5 second TTL lease
		leaseResp, err := s.discoveryClient.Grant(ctx, 5)
		if err != nil {
			return fmt.Errorf("failed to grant lease: %w", err)
		}

		// Generate a random ID for this server
		id := fmt.Sprintf("%d", time.Now().UnixNano())

		// Register the server with the discovery service
		key := s.DiscoveryKeyPrefix + "/backend/" + id
		_, err = s.discoveryClient.Put(ctx, key, s.ExternalAddress, clientv3.WithLease(leaseResp.ID))
		if err != nil {
			return fmt.Errorf("failed to register with discovery service: %w", err)
		}

		// Keep the lease alive
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case res := <-s.shutdown:
				// Revoke the lease
				slog.Info("Revoking lease on shutdown")
				_, err := s.discoveryClient.Revoke(ctx, leaseResp.ID)
				if err != nil {
					slog.Error("Failed to revoke lease", "err", err)
				}
				close(res)
				return nil
			case <-ticker.C:
				_, err := s.discoveryClient.KeepAliveOnce(ctx, leaseResp.ID)
				if err != nil {
					slog.Error("Failed to keep lease alive", "err", err)
				}
			}
		}
	})

	return errGrp.Wait()
}

func (s *Server) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	s.shutdown <- done
	<-done
	return nil
}

func (s *Server) HandlePing(w http.ResponseWriter, r *http.Request) {
	// Echo back the request
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
