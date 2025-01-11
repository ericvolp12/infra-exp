-- NOTE: See REPLICATE.md in the parent directory for a longer
-- explanation of how replicated setups work and when to use them.
--
-- This is an example of a very basic "cache replication" setup.
--
-- Memcached proxy thinks in terms of "pools", not individual backends. This
-- may seem wrong if your goal is to "replicate data to all memcached
-- servers". However doing this is a nonstandard antipattern: The design of
-- memcached is that adding servers _increases the available memory to cache_
-- Thus a "pool" of servers have a key hashed against a list of servers.
--
-- In some cases you may still want to replicate a subset of the cache to
-- multiple servers, or multiple pools in different racks, regions, zones,
-- datacenters, etc.
--
-- In this example we set up two pools in a set with a single backend in each,
-- and then tell the routes below to copy keys to all pools.
local_zone("mc5")

pools {
    set_all = {
        mc1 = {
            backends = { "memcached1:11211" }
        },
        mc2 = {
            backends = { "memcached2:11211" }
        },
        mc3 = {
            backends = { "memcached3:11211" }
        },
        mc4 = {
            backends = { "memcached4:11211" }
        },
        mc5 = {
            backends = { "memcached5:11211" }
        }
    }
}

routes {
    cmap = {
        get = route_zfailover {
            children = "set_all",
            stats = true,
            miss = true,       -- failover on miss
            shuffle = true,    -- try the list in a randomized order
            failover_count = 2 -- retry at most 2 times. comment out to try all
        },
        gets = route_zfailover {
            children = "set_all",
            stats = true,
            miss = true,       -- failover on miss
            shuffle = true,    -- try the list in a randomized order
            failover_count = 2 -- retry at most 2 times. comment out to try all
        }
    },
    -- by default, send commands everywhere. ie; touch/set/delete
    default = route_allsync {
        children = "set_all"
    }
}
