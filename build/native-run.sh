#!/bin/sh
# Test-only container entrypoint. Wait for actual native API report, not a fake GUI.
set -eu
export PROTOSVPN_SMOKE_REPORT=/out/native-report.json
rm -f "$PROTOSVPN_SMOKE_REPORT"
/out/ProtosVPN-native-smoke > /out/native-smoke.log 2>&1 &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' EXIT INT TERM
tries=0
while [ ! -s "$PROTOSVPN_SMOKE_REPORT" ]; do
 if ! kill -0 "$pid" 2>/dev/null; then printf 'Native process exited; inspect native-smoke.log\n'; exit 1; fi
 tries=$((tries+1))
 if [ "$tries" -ge 60 ]; then printf 'Native API report not received; inspect native-smoke.log\n'; exit 1; fi
 sleep 1
done
# The report confirms real API round trips. Allow the following UI reload to paint.
sleep 2
/tmp/capture-x11 /out/native.ppm
printf 'Native API report and real X11 capture written.\n'
