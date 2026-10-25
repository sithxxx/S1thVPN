#!/usr/bin/env bash
set -euo pipefail
cd /opt/sibtunnel/deploy/server
mkdir -p /var/backups/sibtunnel
F=/var/backups/sibtunnel/sibcp-$(date +%F).sql.gz
docker compose exec -T postgres pg_dump -h 127.0.0.1 -U sibcp sibcp | gzip > 
"$F"
find /var/backups/sibtunnel -name '*.sql.gz' -mtime +14 -delete
scp -q "$F" backup@second-server:/backups/     
# или rclone copy в облак 

# /etc/systemd/system/sibtunnel-backup.service
[Service]
Type=oneshot
ExecStart=/opt/sibtunnel/deploy/server/backup.sh
# /etc/systemd/system/sibtunnel-backup.timer
[Timer]
OnCalendar=*-*-* 04:30:00
Persistent=true
[Install]
WantedBy=timers.target