# Memcached Tests

Facebook doesn't provide a `mcrouter` docker image so we need to build a local [`mcrouter`](https://github.com/facebook/mcrouter) image with the following:

```shell
$ docker build -f mcrouter.Dockerfile -t mcrouter:latest .
```

_Note building the `mcrouter` image can take 10+ minutes_

Then we can bring up our experiment with

```shell
$ docker compose up -d
```

## Configs

Current `mcrouter1.json` _should_:
- Proxy misses to a random replicas as part of warmup.
- Proxy to random replicas if mcrouter1 is unavailable.
- Proxy sets and deletes to a random replica so there's at least one more copy floating around in the whole replica pool.

Random replica gets/sets are based on `HashRoutes` with a default hashing policy so gets/sets should only have to talk to the node the data would likely exist on in the replica pools.
