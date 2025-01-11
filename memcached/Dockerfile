FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    build-essential \
    libevent-dev \
    wget && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /memcached_build

RUN wget https://memcached.org/files/memcached-1.6.34.tar.gz

RUN tar -zxvf memcached-1.6.34.tar.gz

WORKDIR /memcached_build/memcached-1.6.34

RUN ./configure --enable-proxy && make && make install

CMD ["memcached"]
