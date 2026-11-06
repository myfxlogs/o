#!/usr/bin/env bash
# dm_repolock_test.sh —— T-0414 写径互斥锁桩测：acquire/release/busy/
# 陈锁抢/token 不符/status 全谱（O_REPO_LOCK_DIR 桩面不碰真 .git）。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
LOCK="$TMP/dm-repo.lock"
export O_REPO_LOCK_DIR="$LOCK"

# ① acquire→token→status→release 常径
TOKEN=$(bash "$ROOT/scripts/meta/dm_repolock.sh" acquire test-a 0)
[ -d "$LOCK" ]
grep -q "holder=test-a" "$LOCK/holder"
bash "$ROOT/scripts/meta/dm_repolock.sh" status | grep -q "token=$TOKEN"
bash "$ROOT/scripts/meta/dm_repolock.sh" release "$TOKEN"
[ ! -d "$LOCK" ]

# ② 持锁期二入=忙（rc3——会话 merge/push 在飞让位语义）
T1=$(bash "$ROOT/scripts/meta/dm_repolock.sh" acquire test-b 0)
if bash "$ROOT/scripts/meta/dm_repolock.sh" acquire test-c 1 2>"$TMP/busy.err"; then
	echo "busy 期 acquire 竟得锁" >&2; exit 1
else
	rc=$?; [ "$rc" -eq 3 ]
fi
grep -q "holder=test-b" "$TMP/busy.err"

# ③ token 不符拒释（rc4——防越权解锁）
if bash "$ROOT/scripts/meta/dm_repolock.sh" release "wrong-token"; then
	echo "错 token 竟释锁" >&2; exit 1
else
	rc=$?; [ "$rc" -eq 4 ]
fi
[ -d "$LOCK" ]
bash "$ROOT/scripts/meta/dm_repolock.sh" release "$T1"
[ ! -d "$LOCK" ]

# ④ 陈锁抢（rc6+STOLEN 行——前持者陈死兜底径）
mkdir "$LOCK"
printf 'token=dead-token\nholder=test-dead\npid=1\nts=old\n' > "$LOCK/holder"
touch -d '1 hour ago' "$LOCK"
OUT=$(bash "$ROOT/scripts/meta/dm_repolock.sh" acquire test-d 0 2>"$TMP/steal.err") || rc=$?
rc=${rc:-0}
[ "$rc" -eq 6 ]
[ -n "$OUT" ]
grep -q "STOLEN" "$TMP/steal.err"
grep -q "holder=test-d" "$LOCK/holder"
grep -q "stolen=" "$LOCK/holder"
bash "$ROOT/scripts/meta/dm_repolock.sh" release "$OUT"

# ⑤ release 幂等（锁目已失 rc0——释意已达成）
bash "$ROOT/scripts/meta/dm_repolock.sh" release "any-token"

echo "dm repolock acquire/busy/stale/release dry run PASS"
