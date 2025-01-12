verbose(true)
local_zone("mc1")
pools {
    set_read = {
        mc1 = {
            backends = { "127.0.0.1:11211" }
        },
        mc2 = {
            backends = { "127.0.0.1:11212" }
        },
        mc3 = {
            backends = { "127.0.0.1:11213" }
        },
        mc4 = {
            backends = { "127.0.0.1:11214" }
        },
        mc5 = {
            backends = { "127.0.0.1:11215" }
        }
    },
    set_write = {
        mc1 = {
            options = { iothread = true },
            backends = { "127.0.0.1:11211" }
        },
        mc2 = {
            options = { iothread = true },
            backends = { "127.0.0.1:11212" }
        },
        mc3 = {
            options = { iothread = true },
            backends = { "127.0.0.1:11213" }
        },
        mc4 = {
            options = { iothread = true },
            backends = { "127.0.0.1:11214" }
        },
        mc5 = {
            options = { iothread = true },
            backends = { "127.0.0.1:11215" }
        }
    }
}


routes {
    cmap = {
        get = route_zfailover {
            children = "set_read",
            stats = true,
            miss = true,       -- failover on miss
            shuffle = true,    -- try the list in a randomized order
            failover_count = 2 -- retry at most 2 times. comment out to try all
        },
        gets = route_zfailover {
            children = "set_read",
            stats = true,
            miss = true,       -- failover on miss
            shuffle = true,    -- try the list in a randomized order
            failover_count = 2 -- retry at most 2 times. comment out to try all
        }
    },
    -- by default, send commands everywhere. ie; touch/set/delete
    default = route_allsync {
        children = "set_write"
    }
}
