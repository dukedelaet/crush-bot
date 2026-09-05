#!/usr/bin/env bash
set -euo pipefail

bin_dir="$(cd "$(dirname "$0")" && pwd)"
state="${FAKE_CRUSH_STATE:-$bin_dir/state}"
mkdir -p "$state"

version() { echo "crush version v0.91.2"; }

# Shift root flags.
yolo=0
debug=0
cwd=""
datadir=""
session=""
args=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version|-v) version; exit 0 ;;
    --help|-h)
      echo "USAGE crush [--yolo] [--cwd] [--data-dir] [--session] [--debug] [command]"
      echo "  -y --yolo"
      exit 0
      ;;
    --yolo|-y) yolo=1; shift ;;
    --debug|-d) debug=1; shift ;;
    --cwd|-c) cwd="$2"; shift 2 ;;
    --data-dir|-D) datadir="$2"; shift 2 ;;
    --session|-s) session="$2"; shift 2 ;;
    --host|-H) shift 2 ;;
    --quiet) shift ;;
    --model) shift 2 ;;
    --json) args+=("$1"); shift ;;
    --continue|-C) shift ;;
    *) args+=("$1"); shift ;;
  esac
done
set -- "${args[@]+"${args[@]}"}"

cmd="${1:-}"
shift || true

case "$cmd" in
  "" )
    echo "fake crush tui"
    exit 0
    ;;
  models)
    echo "fake/test-model"
    exit 0
    ;;
  server)
    echo $$ > "$state/server.pid"
    trap 'exit 0' TERM INT
    while true; do sleep 0.2; done
    ;;
  run)
    uuid="11111111-1111-4111-8111-111111111111"
    echo "$uuid" > "$state/last-uuid"
    echo "online: ${*:-}"
    exit 0
    ;;
  session)
    sub="${1:-}"
    shift || true
    case "$sub" in
      last)
        uuid="$(cat "$state/last-uuid" 2>/dev/null || echo "11111111-1111-4111-8111-111111111111")"
        title="$(cat "$state/last-title" 2>/dev/null || echo "Untitled Session")"
        printf '{"meta":{"id":"abc123","uuid":"%s","title":"%s"}}\n' "$uuid" "$title"
        ;;
      rename)
        id="${1:-}"
        title="${2:-}"
        echo "$title" > "$state/last-title"
        echo "renamed $id $title"
        ;;
      show)
        printf '%s\n' '{"messages":[{"role":"user","parts":[{"type":"text","text":"hi"}]},{"role":"assistant","parts":[{"type":"text","text":"online"}]}]}'
        ;;
      *)
        echo "unknown session $sub" >&2
        exit 1
        ;;
    esac
    ;;
  *)
    echo "unknown $cmd" >&2
    exit 1
    ;;
esac
