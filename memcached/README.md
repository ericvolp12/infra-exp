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
- Proxy to random replicas if memcached1 is unavailable.
- Proxy sets and deletes to all replicas so all instances should have the same keys available.
