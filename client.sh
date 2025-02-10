#!/bin/bash
## This script helps regenerating multiple client instances in different network namespaces using Docker
## This helps to overcome the ephemeral source port limitation
## Usage:   ./client.sh <number of clients> <server addr> <connections> <number of goroutines> <enable tls>
## Example: ./client.sh 10 1m-server:8000 30000 1000 false
## Example: ./client.sh 10 10.89.3.10:8000 30000 1000 false
## Number of clients helps to speed up connections establishment at large scale, in order to make the demo faster

REPLICAS=$1
ADDR=$2
CONNECTIONS=$3
GOROUTINES=$4
TLS=${5:-false}

go build --tags "static netgo" -o build/client client.go

docker rm -vf $(docker ps -aq --filter "label=1m-client") 2>/dev/null

for (( c=0; c<REPLICAS; c++ )); do
    docker run -l 1m-client \
        --sysctl net.ipv4.ip_local_port_range="1024 65535" \
        --ulimit nofile=655350:655350 \
        --network 1m-tcpserver \
        -v "$(pwd)/build/client:/client" \
        --name 1m-client-$c \
        -d alpine \
        /client -addr="$ADDR" -conn="$CONNECTIONS" -timeout=10m -goroutines="$GOROUTINES" -tls="$TLS"
done
