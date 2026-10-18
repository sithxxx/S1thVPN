ip -n cli route add 203.0.113.0/24 via 192.0.2.1
ip netns exec cli ping -c2 -W1 203.0.113.2 # 100% потерь
ip netns exec srv nft add rule ip nat postrouting ip saddr 192.0.2.0/24
oifname s-inet masquerade
ip netns exec cli ping -c2 -W1 203.0.113.2 # работает
#!/usr/bin/env bash
# Лаборатория: cli <-> srv <-> inet в трёх network namespaces.
set -euo pipefail
for n in srv cli inet; do ip netns add $n; ip -n $n link set lo up; done
ip link add s-cli netns srv type veth peer name c-srv netns cli
ip link add s-inet netns srv type veth peer name i-srv netns inet
ip -n srv addr add 192.0.2.1/24 dev s-cli; ip -n srv link set s-cli up
ip -n cli addr add 192.0.2.2/24 dev c-srv; ip -n cli link set c-srv up
ip -n srv addr add 203.0.113.1/24 dev s-inet; ip -n srv link set s-inet up
ip -n inet addr add 203.0.113.2/24 dev i-srv; ip -n inet link set i-srv up
ip netns exec srv sysctl -qw net.ipv4.ip_forward=1
ip netns exec srv nft -f - <<'NFT'
table ip nat {
chain postrouting {
type nat hook postrouting priority srcnat; policy accept;
ip saddr 10.77.0.0/22 oifname "s-inet" masquerade
}
}
NFT