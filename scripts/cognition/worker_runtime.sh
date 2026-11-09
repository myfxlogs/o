#!/usr/bin/env bash

o_worker_runtime_dir() {
	printf '%s\n' "${O_WORKER_RUNTIME_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/o-worker}"
}

o_worker_ensure_runtime_dir() {
	local dir
	dir=$(o_worker_runtime_dir)
	umask 077
	mkdir -p "$dir"
	chmod 700 "$dir"
	mkdir -p "$dir/sessions"
	chmod 700 "$dir/sessions"
}

o_worker_parallel_limit() {
	# T-0509 白名单动作面：界内微调覆写档优先（whitelist_actions.sh
	# 规则#5 写；有效值 1|2 才采信——坏档/缺档回落 env 缺省律）。
	local ovf ov
	ovf="$(o_worker_runtime_dir)/parallel-limit-override"
	if [ -r "$ovf" ]; then
		ov=$(head -n1 "$ovf" 2>/dev/null | tr -d '[:space:]')
		case "$ov" in 1|2) printf '%s\n' "$ov"; return 0 ;; esac
	fi
	local limit=${O_WORKER_PARALLEL_LIMIT:-1}
	case "$limit" in
		1|2) printf '%s\n' "$limit" ;;
		*) echo "O_WORKER_PARALLEL_LIMIT must be 1 or 2" >&2; return 2 ;;
	esac
}

o_worker_quota_until() {
	local file epoch now
	file="$(o_worker_runtime_dir)/quota-reset-epoch"
	[[ -e "$file" ]] || return 1
	[[ -r "$file" ]] || return 2
	read -r epoch < "$file" || return 2
	[[ "$epoch" =~ ^[0-9]+$ ]] || return 2
	now=$(date +%s)
	(( epoch > now )) || return 1
	printf '%s\n' "$epoch"
}

o_worker_session_state() {
	local devin_bin=$1 output
	output=$("$devin_bin" list --format json 2>/dev/null) || return 2
	python3 -c '
import json, sys
try:
    value = json.load(sys.stdin)
except Exception:
    raise SystemExit(2)
def extract_sessions(item):
    if isinstance(item, list):
        return item
    if isinstance(item, dict):
        for key in ("sessions", "items", "data"):
            if key in item:
                return extract_sessions(item[key])
        if item.get("session_id") or item.get("id"):
            return [item]
    return None
sessions = extract_sessions(value)
if not isinstance(sessions, list):
    raise SystemExit(2)
print("resume" if sessions else "new")
' <<< "$output"
}

o_worker_is_rate_limited() {
	grep -Eqi 'reached .*rate limit|rate limit.*reset|quota.*reset' "$1"
}

o_worker_record_quota_reset() {
	local log=$1 runtime minutes now until tmp current
	runtime=$(o_worker_runtime_dir)
	minutes=$(sed -nE 's/.*reset in ([0-9]+) minutes?.*/\1/ip' "$log" | tail -1)
	[[ "$minutes" =~ ^[0-9]+$ ]] || minutes=${O_WORKER_RATE_LIMIT_FALLBACK_MINUTES:-15}
	[[ "$minutes" =~ ^[0-9]+$ ]] || minutes=15
	now=$(date +%s)
	until=$((now + minutes * 60 + 60))
	o_worker_ensure_runtime_dir
	exec 8>"$runtime/quota-reset.lock"
	flock -x 8
	if current=$(o_worker_quota_until) && (( current > until )); then
		until=$current
	fi
	tmp=$(mktemp "$runtime/.quota-reset.XXXXXX")
	chmod 600 "$tmp"
	printf '%s\n' "$until" > "$tmp"
	mv -f "$tmp" "$runtime/quota-reset-epoch"
	flock -u 8
	exec 8>&-
	printf '%s\n' "$until"
}
