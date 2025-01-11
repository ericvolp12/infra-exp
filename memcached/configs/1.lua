verbose(true)
local_zone("mc1")
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
