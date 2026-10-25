#!/usr/bin/env bash
# Разворачивает SibTunnel на чистой Ubuntu 24.04:  sudo 
./deploy/server/bootstrap.sh vpn.example.org
set -euo pipefail
DOMAIN=${1:?укажите домен: bootstrap.sh vpn.example.org}
REPO=$(cd "$(dirname "$0")/../.." && pwd)
# 1. Пакеты
apt-get update
apt-get install -y nftables iproute2 ca-certificates curl openssl
command -v docker >/dev/null || curl -fsSL https://get.docker.com | sh
# 2. Пользователь датаплейна и каталоги
id sibtunnel >/dev/null 2>&1 || useradd --system --no-create-home --shell 
/usr/sbin/nologin sibtunnel
install -d -m 750 -o root -g sibtunnel /etc/sibtunnel
# 3. Бинарники, ключ и конфиг сервера (ключ создаётся один раз и не покидает 
сервер)
install -m 755 "$REPO/bin/sibtund" "$REPO/bin/sibtun" /usr/local/bin/
[ -f /etc/sibtunnel/server.key ] || (umask 077; sibtun genkey > 
/etc/sibtunnel/server.key)
chgrp sibtunnel /etc/sibtunnel/server.key && chmod 640 
/etc/sibtunnel/server.key
install -m 640 -g sibtunnel "$REPO/deploy/server/server.example.toml" 
/etc/sibtunnel/server.toml
# 4. Ядро и файрвол
install -m 644 "$REPO/deploy/server/sysctl.d/99-sibtunnel.conf" 
/etc/sysctl.d/
sysctl --system >/dev/null
install -m 644 "$REPO/deploy/server/nftables.conf" /etc/nftables.conf
nft -c -f /etc/nftables.conf
systemctl enable --now nftables && nft -f /etc/nftables.conf
Page 11 of 22
Ф4 · Продакшн и деплой
# 5. Датаплейн
install -m 644 "$REPO/deploy/server/systemd/sibtund.service" 
/etc/systemd/system/
systemctl daemon-reload && systemctl enable --now sibtund
# 6. Control plane, HTTPS, мониторинг
cd "$REPO/deploy/server"
if [ ! -f .env ]; then
umask 077
cat > .env <<EOF
POSTGRES_PASSWORD=$(openssl rand -hex 24)
SIBTUNNEL_GID=$(getent group sibtunnel | cut -d: -f3)
SIBCP_SECRET_KEY=$(openssl rand -hex 32)
SIBCP_PUBLIC_BASE_URL=https://$DOMAIN
SIBCP_SERVER_ENDPOINT=$(curl -4 -fsS https://ifconfig.me):51900
SIBCP_SERVER_PUBLIC_KEY=$(sibtun pubkey "$(cat /etc/sibtunnel/server.key)")
EOF
fi
sed -i "s/vpn.example.org/$DOMAIN/" Caddyfile
docker compose up -d --build
docker compose exec sibcp sibctl create-admin admin
echo "Готово. Админка: ssh -L 8000:127.0.0.1:8000 root@$DOMAIN → 
http://localhost:8000/admin"