SRC_FILE=$(realpath -m "${1:-s2_epoll}/server.go")
go build -tags=static,netgo,poll_opt,gc_opt -o build/server "$SRC_FILE"

docker network rm 1m-tcpserver 2>/dev/null
docker network create --subnet 10.89.3.0/24 1m-tcpserver

docker run -l 1m-server \
    --sysctl net.core.somaxconn=10240 \
    --sysctl net.ipv4.tcp_max_syn_backlog=10240 \
    --ulimit nofile=655350:655350 \
    --network 1m-tcpserver \
    --ip 10.89.3.10 \
    -v "$(pwd)/build/server:/server" \
    --name 1m-server \
    -p 8112:8112 \
    --rm \
    alpine /server
