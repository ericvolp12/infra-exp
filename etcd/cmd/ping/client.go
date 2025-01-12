package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	cli "github.com/urfave/cli/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var clientCmd = &cli.Command{
	Name:  "client",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		// Flags
		opt := struct {
			DiscoveryKeyPrefix string
			DiscoveryAddresses []string
		}{
			DiscoveryKeyPrefix: cctx.String("discovery-key-prefix"),
			DiscoveryAddresses: cctx.StringSlice("discovery-addresses"),
		}

		c, err := NewClient(opt.DiscoveryKeyPrefix, opt.DiscoveryAddresses)
		if err != nil {
			return err
		}

		// Ping every second
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			pc := c.GetClient()
			if pc == nil {
				slog.Error("No backend available")
				continue
			}
			if err := pc.Ping(ctx); err != nil {
				slog.Error("Failed to ping backend", "err", err)
			} else {
				slog.Info("Pinged backend", "addr", pc.serviceAddr)
			}
		}

		return nil
	},
}

type Client struct {
	DiscoveryKeyPrefix string
	DiscoveryAddresses []string
	clients            map[string]*PingSubclient
	lk                 sync.RWMutex
}

type serverPair struct {
	id      string
	address string
}

func NewClient(discoveryKeyPrefix string, discoveryAddresses []string) (*Client, error) {
	c := Client{
		DiscoveryKeyPrefix: discoveryKeyPrefix,
		DiscoveryAddresses: discoveryAddresses,
		clients:            make(map[string]*PingSubclient),
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   discoveryAddresses,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		slog.Error("Failed to create etcd client", "err", err)
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	keyPrefix := discoveryKeyPrefix + "/backend/"

	// Add existing backends
	resp, err := cli.Get(context.Background(), keyPrefix, clientv3.WithPrefix())
	if err != nil {
		slog.Error("Failed to get existing backends", "err", err)
		return nil, fmt.Errorf("failed to get existing backends: %w", err)
	}
	for _, kv := range resp.Kvs {
		s := serverPair{
			id:      string(kv.Key)[len(keyPrefix):],
			address: string(kv.Value),
		}
		slog.Info("Adding existing backend", "id", s.id, "addr", s.address)
		c.AddServers([]serverPair{s})
	}

	// Start a goroutine to watch for changes in the discovery service
	go func() {
		defer cli.Close()

		// Watch for changes, start at the current revision from our initial get
		wch := cli.Watch(context.Background(), keyPrefix, clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision+1))
		for wresp := range wch {
			for _, ev := range wresp.Events {
				if ev.Type == clientv3.EventTypePut {
					s := serverPair{
						id:      string(ev.Kv.Key)[len(keyPrefix):],
						address: string(ev.Kv.Value),
					}
					slog.Info("Adding new backend", "id", s.id, "addr", s.address)
					c.AddServers([]serverPair{s})
				} else if ev.Type == clientv3.EventTypeDelete {
					id := string(ev.Kv.Key)[len(keyPrefix):]
					slog.Info("Removing backend", "id", id)
					c.RemoveServers([]string{id})
				}
			}
		}
	}()

	return &c, nil
}

func (c *Client) AddServers(servers []serverPair) {
	c.lk.Lock()
	defer c.lk.Unlock()
	for _, server := range servers {
		// Skip if already added
		if _, ok := c.clients[server.id]; ok {
			continue
		}
		c.clients[server.id] = &PingSubclient{
			serviceAddr: server.address,
			httpClient:  &http.Client{Timeout: 5 * time.Second},
		}
	}
}

func (c *Client) RemoveServers(ids []string) {
	c.lk.Lock()
	defer c.lk.Unlock()
	for _, id := range ids {
		delete(c.clients, id)
	}
}

func (c *Client) GetClient() *PingSubclient {
	c.lk.RLock()
	defer c.lk.RUnlock()
	numClients := len(c.clients)
	if numClients == 0 {
		return nil
	}

	i := 0
	clientNum := rand.IntN(numClients)
	for _, pc := range c.clients {
		if i == clientNum {
			return pc
		}
		i++
	}
	return nil
}

type PingSubclient struct {
	serviceAddr string
	httpClient  *http.Client
}

func (c *PingSubclient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.serviceAddr+"/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
