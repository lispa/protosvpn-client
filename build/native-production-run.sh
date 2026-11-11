#!/bin/sh
# Smoke the exact distributable binary, with no test-only entrypoint or fixtures.
set -eu
/src/build/bin/ProtosVPN-linux-amd64 > /out/native-production.log 2>&1 &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' EXIT INT TERM
tries=0
until /tmp/capture-x11 /out/native-production.ppm; do
 if ! kill -0 "$pid" 2>/dev/null; then printf 'Production binary exited; inspect native-production.log\n'; exit 1; fi
 tries=$((tries+1)); if [ "$tries" -ge 30 ]; then exit 1; fi
 sleep 1
done
# Window-map is verified; allow embedded application assets to paint, then capture.
sleep 2
kill -0 "$pid"
/tmp/capture-x11 /out/native-production.ppm
printf 'Exact production Linux executable remained running and rendered an X11 window.\n'
