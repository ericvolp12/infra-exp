# Dockerfile for Mcrouter

# Use Ubuntu 24.04 as the base image
FROM ubuntu:24.04 as build

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
RUN apt-get update && apt-get install -y \
    git \
    curl \
    sudo \
    build-essential \
    python3 \
    python3-pip \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Symlink python3 to python
RUN ln -s /usr/bin/python3 /usr/bin/python

# Clone the Mcrouter repository
RUN git clone https://github.com/facebook/mcrouter.git /mcrouter

# Set working directory
WORKDIR /mcrouter

# Run the Ubuntu 24.04 install script
RUN chmod +x mcrouter/scripts/install_ubuntu_24.04.sh

RUN mkdir -p mcrouter-install/install

# Install Mcrouter dependencies
RUN ./mcrouter/scripts/install_ubuntu_24.04.sh "$(pwd)"/mcrouter-install deps

# Build Mcrouter
RUN ./mcrouter/scripts/install_ubuntu_24.04.sh "$(pwd)"/mcrouter-install mcrouter

# Copy Mcrouter binary to /usr/local/bin
RUN cp ./mcrouter-install/install/bin/mcrouter /usr/local/bin/mcrouter

# Expose default Mcrouter port
EXPOSE 5000

# Command to run Mcrouter
CMD ["mcrouter"]
