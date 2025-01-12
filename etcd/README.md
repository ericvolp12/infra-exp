# `etcd` Experiments

This folder contains some experiments to use `etcd` as a service discovery database.

Clients watch a key prefix to determine which servers are available to connect to and to determine when to remove a server from their configuration and stop sending it requests.

Servers spin up, create a lease with a short TTL for their `etcd` keys, and create a key/value pair once ready within the prefix clients are watching to let clients identify and connect to the server.

When the server crashes, it unregisters itself via lease TTL expiration. When the server exits cleanly, it revokes its lease and deletes its keys before shutting down so clients stop sending requests.

## Running the Experiment

A docker compose file is provided to spin up an `etcd` cluster locally and a handful of demo ping servers.

```bash
$ docker compose up -d
```

To run a test client, use Go:
```bash
$ go run ./cmd/ping client
```

You can run mutiple clients in different terminals and watch their behavior.

## Simulating Failures

You can drop servers from the cluster cleanly with:

```shell
$ docker stop pingserver1
```

You can kill servers (unclean exit) with:
```shell
$ docker kill pingserver1
```

You can bring the servers back online with:
```shell
$ docker compose up -d
```

You can run `etcdctl` to inspect keys and prefixes with:
```shell
$ docker exec -it etcd-1 etcdctl --endpoints=localhost:2480 watch --prefix pingservice/backend
```
