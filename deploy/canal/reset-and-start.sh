#!/usr/bin/env sh
set -e

RESET_CANAL="${RESET_CANAL:-0}"

if [ "${RESET_CANAL}" = "1" ]; then
  echo "Resetting Canal position/TSDB..."
  find /home/admin/canal-server/conf -maxdepth 2 \( -name meta.dat -o -name h2.mv.db \) -type f -print -delete || true
else
  echo "Skipping Canal reset. Set RESET_CANAL=1 to force reset."
fi

# Remove stale pid to avoid "found canal.pid" on restart.
rm -f /home/admin/canal-server/bin/canal.pid

/home/admin/canal-server/bin/startup.sh

# Keep container in foreground.
tail -F /home/admin/canal-server/logs/canal/canal_stdout.log
