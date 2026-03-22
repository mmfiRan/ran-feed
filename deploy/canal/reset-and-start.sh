#!/usr/bin/env sh
set -e

# Always reset Canal position/TSDB so it won't get stuck on bad offsets.
find /home/admin/canal-server/conf -maxdepth 2 \( -name meta.dat -o -name h2.mv.db \) -type f -print -delete || true

# Remove stale pid to avoid "found canal.pid" on restart.
rm -f /home/admin/canal-server/bin/canal.pid

/home/admin/canal-server/bin/startup.sh

# Keep container in foreground.
tail -F /home/admin/canal-server/logs/canal/canal_stdout.log
