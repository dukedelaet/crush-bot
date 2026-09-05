#!/usr/bin/env bash
# Two-bot mesh smoke using the fake Crush in testdata (no provider).
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$root/internal/crush/testdata:$PATH"
chmod +x "$root/internal/crush/testdata/fake-crush.sh"
ln -sfn "$root/internal/crush/testdata/fake-crush.sh" /tmp/crushbot-e2e-bin/crush 2>/dev/null || true
tmpdir="$(mktemp -d)"
mkdir -p "$tmpdir/bin"
cp "$root/internal/crush/testdata/fake-crush.sh" "$tmpdir/bin/crush"
chmod +x "$tmpdir/bin/crush"
export PATH="$tmpdir/bin:$PATH"
export FAKE_CRUSH_STATE="$tmpdir/st"
export CRUSHBOT_HOME="$tmpdir/home"
export XDG_CONFIG_HOME="$tmpdir/cfg"
cd "$root"
go build -o "$tmpdir/crushbot" ./cmd/crushbot
cb="$tmpdir/crushbot"
"$cb" init
"$cb" spawn alpha --title Alpha
"$cb" spawn beta --title Beta
pend="$CRUSHBOT_HOME/bots/beta/inbox/pending"
mkdir -p "$pend"
cat > "$pend/e2edm.json" <<EOF
{"id":"e2edm","kind":"dm","from":"alpha","to":"beta","hop":1,"body":"hello from e2e","trace":["user","alpha"],"attribution":"Message from alpha (@alpha):","created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
EOF
"$cb" daemon start
sleep 0.8
"$cb" daemon stop || true
if ! test -f "$CRUSHBOT_HOME/bots/beta/inbox/archive/e2edm.json"; then
  echo "FAIL: expected archived envelope on beta" >&2
  "$cb" inbox
  exit 1
fi
if ! grep -q '"kind": "receipt"' "$CRUSHBOT_HOME/bots/alpha/inbox/archive/"*.json "$CRUSHBOT_HOME/bots/alpha/inbox/pending/"*.json 2>/dev/null; then
  echo "FAIL: expected receipt on alpha" >&2
  "$cb" inbox
  exit 1
fi
echo "ok e2e mesh"
