sysctl -w fs.file-max=1073741816
sysctl -w fs.nr_open=1073741816
ulimit -n 655360

sysctl -w net.nf_conntrack_max=2621440
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_max_syn_backlog=10240
sysctl -w net.core.somaxconn=10240
sysctl -w net.core.netdev_max_backlog=10240
