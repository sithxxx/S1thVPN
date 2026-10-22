#!/usr/bin/env bash
# Сквозной тест MVP: control plane + data plane + клиент в network
namespaces.
# Запуск: make e2e (нужны root, собранные ./bin/* и uv).
set -euo pipefail
cd "$(dirname "$0")/.."
BIN=$PWD/bin; W=$(mktemp -d); export W
cleanup() {
    kill $(jobs -p) 2>/dev/null || true
    [ -f "$W/cp.pid" ] && kill "$(cat "$W/cp.pid")" 2>/dev/null || true
    pkill -x sibtund || true; pkill -x sibtun || true
    ./lab/down.sh
}
trap cleanup EXIT
./lab/down.sh; ./lab/up.sh

# 1. Data plane
$BIN/sibtun genkey > "$W/server.key"
cat > "$W/server.toml" <<EOF
[interface]
private_key_file = "$W/server.key"
listen_port = 51900
address = "10.77.0.1/22"
[uapi]
socket = "$W/uapi.sock"
EOF
start_server() { ip netns exec srv "$BIN/sibtund" -config "$W/server.toml"
>>"$W/srv.log" 2>&1 & echo $! > "$W/srv.pid"; }
start_server; sleep 1

# 2. Control plane (SQLite вместо PostgreSQL — только для теста)
export SIBCP_DATABASE_URL="sqlite+aiosqlite:///$W/cp.db" SIBCP_SECRET_KEY=e2e
SIBCP_COOKIE_SECURE=false \
Ф3 · Мультиклиентность, MVP
Page 20 of 23
    SIBCP_UAPI_SOCKET="$W/uapi.sock" SIBCP_SERVER_ENDPOINT=192.0.2.1:51900 \
    SIBCP_SERVER_PUBLIC_KEY="$($BIN/sibtun pubkey "$(cat "$W/server.key")")" \
    SIBCP_PUBLIC_BASE_URL=http://127.0.0.1:8000 SIBCP_RECONCILE_INTERVAL_S=2
cd controlplane
uv run alembic upgrade head
uv run sibctl create-admin e2e --password e2e-pass
uv run uvicorn --factory sibcp.main:app --port 8000 >>"$W/cp.log" 2>&1 & echo
$! > "$W/cp.pid"
sleep 3

# 3. Админ создаёт пользователя и устройство, «пользователь» скачивает конфиг
uv run python - <<'PY'
import httpx, os
c = httpx.Client(base_url="http://127.0.0.1:8000")
c.post("/api/auth/login", json={"username": "e2e", "password": "e2epass"}).raise_for_status()
u = c.post("/api/users", json={"name": "e2e"}).json()
d = c.post(f"/api/users/{u['id']}/devices", json={"name": "lab"}).json()
cfg = c.get(d["invite_url"] + "/config"); cfg.raise_for_status()
open(os.environ["W"] + "/client.toml", "w").write(cfg.text)
assert c.get(d["invite_url"] + "/config").status_code == 410 # второй раз
нельзя
PY
cd ..

# 4. Клиент подключается и выходит в «интернет»
ip netns exec cli "$BIN/sibtun" up "$W/client.toml" >>"$W/cli.log" 2>&1 &
sleep 2
ip netns exec cli ping -c3 -W2 203.0.113.2 >/dev/null && echo "OK: интернет
через туннель"

# 5. Перезапуск датаплейна: reconciler возвращает пира, клиент
переподключается сам
kill "$(cat "$W/srv.pid")"; sleep 1; start_server; T0=$(date +%s)
for _ in $(seq 1 60); do
    if ip netns exec cli ping -c1 -W1 203.0.113.2 >/dev/null 2>&1; then
        echo "OK: восстановление за $(( $(date +%s) - T0 )) с"; exit 0
    fi
done
echo "FAIL"; tail -n 20 "$W"/*.log; exit 1