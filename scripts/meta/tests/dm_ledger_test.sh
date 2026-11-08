#!/usr/bin/env bash
# dm_ledger_test.sh —— T-0414 机械轨桩测：真 git 仓+桩 psql/esc/check_all/
# git_push。场景谱：
#   ①B 补登：done/ 票无完成行+状态未翻→INSERT META-DM 完成行+状态翻完成
#     （「补登+票翻」同拍）；
#   ②A-完成面：latest=完成 票在 tasks/→归档 done/+翻状态+commit+push；
#   ③A-裁决面：裁决通过+收口 SHA 证并→归档；④裁决未证并+龄超→升件不翻；
#   ⑤写径锁忙→零 mv 零 commit（竞态互斥桩）；
#   ⑥饿殍面（T-0417）：40 件 rc2 古票吃满批帽→新完成票仍入集翻页，
#     rc2 票零写零翻；
#   ⑦GW-001 段B3 两源扩证：裁决执行件（无票支∧范围节无码件）与外审归宗
#     （票面「外审裁过归宗@<sha>」∧sha∈master）翻完成+清单留证；
#     有票支/有码件负例不翻。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT
mkdir -p "$TMP/bin" "$TMP/repo/docs/tasks/done"

# —— 真 git 仓（无 origin：MASTER_REF=master 本支面；fetch 哑降级在案）——
REPO="$TMP/repo"
git -C "$REPO" init -q
git -C "$REPO" config user.email t@t && git -C "$REPO" config user.name t
cat > "$REPO/docs/tasks/T-9002-完成行归档件.md" <<'EOF'
# T-9002 x

- 状态：待验（施工收口——移交决策席审计）
EOF
cat > "$REPO/docs/tasks/T-9003-裁决归档件.md" <<'EOF'
# T-9003 y

- 状态：待验
EOF
cat > "$REPO/docs/tasks/T-9004-裁决未并件.md" <<'EOF'
# T-9004 z

- 状态：待验
EOF
cat > "$REPO/docs/tasks/T-9005-锁忙待翻件.md" <<'EOF'
# T-9005 l

- 状态：待验
EOF
cat > "$REPO/docs/tasks/T-9100-饿殍新票件.md" <<'EOF'
# T-9100 s

- 状态：待验
EOF
# 段B3(a)：裁决执行件——范围节在且全档皆文档（零码件路径词）
cat > "$REPO/docs/tasks/T-9007-裁决执行件.md" <<'EOF'
# T-9007 e

- 状态：待验

## 范围

1. 纯诊断定性——产出判词文档 docs/reports/t9007-verdict.md 备查
2. 零码件零票支——交付物=裁决判词本身
EOF
# 段B3(b)：外审归宗件——票面标记 sha 在 init commit 后回填（base-ref 祖先实证）
# 段B3 负例：范围节含码件路径+票支在案→两源皆不中不翻
cat > "$REPO/docs/tasks/T-9009-有支有码件.md" <<'EOF'
# T-9009 n

- 状态：待验

## 范围

1. 改 scripts/meta/dm_ledger.sh 补证逻辑
2. 范围外发现不自修
EOF
# 饿殍景桩：40 件 done/ 古票 rc2 异形态（| 状态 | 表形——status_done 不
# 识）恒入动作集吃满批帽 MAX_BATCH=40，升序排在新完成票前→新票永不进集
for i in $(seq -w 1 40); do
	printf '# T-00%s 古票\n\n| 状态 | 完成 |\n' "$i" > "$REPO/docs/tasks/done/T-00$i-古票rc2件.md"
done
cat > "$REPO/docs/tasks/done/T-9001-台账漂移件.md" <<'EOF'
# T-9001 w

## 状态

**施工中**（devin-t9001 @ 2026-11-06）
EOF
cat > "$REPO/docs/tasks/done/T-9006-作废态件.md" <<'EOF'
# T-9006 v

> 状态：作废（superseded——清账定性）
EOF
git -C "$REPO" add -A && git -C "$REPO" commit -qm init
SHA_CLOSE=$(git -C "$REPO" rev-parse HEAD) # 收口 SHA 桩——在 master 链上=已并证
git -C "$REPO" branch base-ref # 远端快照桩——HEAD 领先 base-ref=待推

# —— psql 桩：场景行集注 A/B 扫描+收口 note；INSERT 全落账 ——
cat > "$TMP/bin/psql" <<'EOF'
#!/usr/bin/env bash
last="${*: -1}"
if [ -f "$last" ]; then query=$(cat "$last"); else query="$last"; fi
case "$query" in
	*'审计中'*|*"c.event IN ('收口'"*)
		# B 面扫描（收口类存在谓词钉）——按场景回集
		case "${LEDGER_SCENARIO:-empty}" in
			brecon) printf 'T-9001\x1f收口\nT-9006\x1f裁决\nBASESWP\x1f升件\n' ;;
			adone|verdict|stale) printf 'T-9001\x1f收口\nT-9006\x1f裁决\n' ;;
		esac ;;
	*"SELECT DISTINCT ON (l.ticket)"*)
		case "${LEDGER_SCENARIO:-empty}" in
			adone) printf 'T-9002\x1f完成\x1f2026-11-07\x1f100\n' ;;
			starve) for i in $(seq -w 1 40); do printf 'T-00%s\x1f完成\x1f2026-11-01\x1f99999\n' "$i"; done
				printf 'T-9100\x1f完成\x1f2026-11-07\x1f100\n' ;;
			verdict) printf 'T-9003\x1f裁决\x1f2026-11-07\x1f100\n' ;;
			stale) printf 'T-9004\x1f裁决\x1f2026-11-01\x1f99999\n' ;;
			lockbusy) printf 'T-9005\x1f完成\x1f2026-11-07\x1f100\n' ;;
			execproof) printf 'T-9007\x1f裁决\x1f2026-11-07\x1f100\n' ;;
			waiproof) printf 'T-9008\x1f裁决\x1f2026-11-07\x1f100\n' ;;
			execfail) printf 'T-9009\x1f裁决\x1f2026-11-07\x1f100\n' ;;
		esac ;;
	*"SELECT note FROM ops_event"*)
		case "${LEDGER_SCENARIO:-empty}" in
			verdict) printf 'auto@%s\n' "$STUB_CLOSE_SHA" ;;
		esac ;;
	*"SELECT 1 FROM ops_event WHERE event='升件'"*) ;;
	*INSERT*|*BEGIN*|*SELECT*|*COMMIT*)
		printf '%s\n' "$query" >> "$LEDGER_SQL_LOG"
		grep -q 'RETURNING ticket' <<<"$query" && printf 'T-9001\n' ;;
	*) printf '%s\n' "$query" >> "$LEDGER_SQL_LOG" ;;
esac
EOF
cat > "$TMP/bin/ops-esc" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$ESC_LOG"
EOF
cat > "$TMP/bin/check-all" <<'EOF'
#!/usr/bin/env bash
echo called >> "$CHECKALL_LOG"
EOF
cat > "$TMP/bin/git-push" <<'EOF'
#!/usr/bin/env bash
echo called >> "$PUSH_LOG"
EOF
chmod +x "$TMP/bin/"*

printf 'FIN_PG_DSN=fake\n' > "$TMP/env"
export PATH="$TMP/bin:$PATH"
export DM_ENV_FILE="$TMP/env"
export O_LEDGER_REPO="$REPO" O_LEDGER_MASTER_REF=base-ref
export O_LEDGER_LOCK_SH="$ROOT/scripts/meta/dm_repolock.sh"
export O_REPO_LOCK_DIR="$TMP/dm-repo.lock"
export O_LEDGER_CHECK_ALL="$TMP/bin/check-all" O_LEDGER_GIT_PUSH="$TMP/bin/git-push"
export O_LEDGER_OPS_ESC="$TMP/bin/ops-esc" O_LEDGER_WT_ROOT="$TMP/wt"
export O_LEDGER_LOCK_WAIT=1 O_LEDGER_VERDICT_STALE=7200
export LEDGER_SQL_LOG="$TMP/ledger.sql" ESC_LOG="$TMP/esc.log"
export CHECKALL_LOG="$TMP/checkall.log" PUSH_LOG="$TMP/push.log"
export STUB_CLOSE_SHA="$SHA_CLOSE"

run() { LEDGER_SCENARIO=$1 bash "$ROOT/scripts/meta/dm_ledger.sh" >"$TMP/out.$1" 2>&1 || echo "rc=$?" >>"$TMP/out.$1"; }

# ①B 补登：done/ 票无完成行+状态未翻→补登行+状态翻+commit+push 同拍
run brecon
grep -qF "SELECT 'CTO','完成','T-9001','机械轨——台账对账补登" "$LEDGER_SQL_LOG"
grep -q 'pg_advisory_xact_lock' "$LEDGER_SQL_LOG"
grep -qF '**完成**（机械轨台账对账补登' "$REPO/docs/tasks/done/T-9001-台账漂移件.md"
git -C "$REPO" log --format=%s -1 | grep -q '票翻完成'
[ -s "$PUSH_LOG" ] && [ -s "$CHECKALL_LOG" ]
# 作废态不翻字但完成行照补（done/ 在案=闭管铁证）；非票键不入补登集
grep -q '作废' "$REPO/docs/tasks/done/T-9006-作废态件.md"
! grep -q "'完成','BASESWP'" "$LEDGER_SQL_LOG"
grep -q "'完成','T-9006'" "$LEDGER_SQL_LOG"

# ②A-完成面：latest=完成+票在 tasks/→归档（零合并证——台账权威直翻）
run adone
[ -f "$REPO/docs/tasks/done/T-9002-完成行归档件.md" ]
[ ! -f "$REPO/docs/tasks/T-9002-完成行归档件.md" ]
grep -qF '状态：完成（ops_event 完成行 2026-11-07 在案' "$REPO/docs/tasks/done/T-9002-完成行归档件.md"

# ③A-裁决面：裁决通过+收口 note 内 SHA 在 master 链→归档+判词模板
run verdict
[ -f "$REPO/docs/tasks/done/T-9003-裁决归档件.md" ]
grep -qF '状态：完成（决策席裁决通过 2026-11-07' "$REPO/docs/tasks/done/T-9003-裁决归档件.md"

# ④裁决未证并+龄超阈→升件不翻（合并未证=防未并件标完成）
: > "$ESC_LOG"
run stale
[ -f "$REPO/docs/tasks/T-9004-裁决未并件.md" ] # 票未动
grep -q 'LEDGER-MERGE-UNPROVEN' "$ESC_LOG"

# ⑦a 段B3 裁决执行件：无票支∧范围节全档皆文档→判词即交付物翻完成
run execproof
[ -f "$REPO/docs/tasks/done/T-9007-裁决执行件.md" ]
[ ! -f "$REPO/docs/tasks/T-9007-裁决执行件.md" ]
grep -qF '取证=裁决执行件:无票支∧范围节无码件' "$REPO/docs/tasks/done/T-9007-裁决执行件.md"
grep -q '票翻动作集' "$TMP/out.execproof" # 首拍清单留证——不静默翻
grep -q '票翻 T-9007' "$TMP/out.execproof"

# ⑦b 段B3 外审归宗：票面「外审裁过归宗@<sha>」∧sha∈base-ref→merge 证据翻完成
# （范围节含码件路径→exec 证先否落 waipai 证；标记置回填节全文 grep 面）
cat > "$REPO/docs/tasks/T-9008-外审归宗件.md" <<EOF
# T-9008 w

- 状态：待验

## 范围

1. 外审件归宗——merge 证据回填
2. 码件面 scripts/meta/dm_ledger.sh

## 回填

外审裁过归宗@${SHA_CLOSE}——归宗锚在案
EOF
git -C "$REPO" add docs/tasks/T-9008-外审归宗件.md && git -C "$REPO" commit -qm t9008-fixture
run waiproof
[ -f "$REPO/docs/tasks/done/T-9008-外审归宗件.md" ]
grep -qF '取证=外审归宗' "$REPO/docs/tasks/done/T-9008-外审归宗件.md"
grep -q '∈master' "$REPO/docs/tasks/done/T-9008-外审归宗件.md"
grep -q '票翻 T-9008' "$TMP/out.waiproof"

# ⑦c 段B3 负例：票支在案+范围节有码件→两源皆不中不翻（龄新压下拍续）
git -C "$REPO" checkout -qb t/9009-side
printf 'side\n' > "$REPO/side.txt"; git -C "$REPO" add side.txt; git -C "$REPO" commit -qm side
git -C "$REPO" checkout -q -
: > "$ESC_LOG" # stale 升件行清场——execfail 断言只看本场景
run execfail
[ -f "$REPO/docs/tasks/T-9009-有支有码件.md" ] # 不翻——证不出票支+码件双否
! grep -q '票翻 T-9009' "$TMP/out.execfail"
! grep -q 'LEDGER-MERGE-UNPROVEN' "$ESC_LOG" # 龄新非 stale——压下拍续不升件

# ⑤写径锁忙→零 mv 零 commit 零 push（竞态互斥桩：会话写轨持锁中）
mkdir "$TMP/dm-repo.lock"
printf 'token=session-tok\nholder=决策席-merge\n' > "$TMP/dm-repo.lock/holder"
HEAD_BEFORE=$(git -C "$REPO" rev-parse HEAD)
PUSH_BEFORE=$(wc -l <"$PUSH_LOG")
run lockbusy
[ -f "$REPO/docs/tasks/T-9005-锁忙待翻件.md" ] # 锁忙不翻——让位下拍
[ "$(git -C "$REPO" rev-parse HEAD)" = "$HEAD_BEFORE" ]
[ "$(wc -l <"$PUSH_LOG")" = "$PUSH_BEFORE" ]
grep -q '写径锁忙' "$TMP/out.lockbusy"

# ⑥饿殍面（T-0417 钦定判 A 桩）：40 rc2 古票吃满批帽下新完成票仍入集
# 翻页；rc2 票零写零翻（不识形不碰——异形态史录原样保）
rm -rf "$TMP/dm-repo.lock" # ⑤ 桩锁清场——复自由写径
PUSH_BEFORE=$(wc -l <"$PUSH_LOG"); CHECKALL_BEFORE=$(wc -l <"$CHECKALL_LOG")
run starve
[ -f "$REPO/docs/tasks/done/T-9100-饿殍新票件.md" ]
[ ! -f "$REPO/docs/tasks/T-9100-饿殍新票件.md" ]
grep -qF '状态：完成（ops_event 完成行 2026-11-07 在案' "$REPO/docs/tasks/done/T-9100-饿殍新票件.md"
! grep -q '批帽' "$TMP/out.starve"
git -C "$REPO" log --format=%s -1 | grep -q 'T-9100'
git -C "$REPO" show --name-only --format='' HEAD | grep -q 'T-9100'
! git -C "$REPO" show --name-only --format='' HEAD | grep -q 'T-00'
grep -qF '| 状态 | 完成 |' "$REPO/docs/tasks/done/T-0001-古票rc2件.md"
[ "$(wc -l <"$PUSH_LOG")" -gt "$PUSH_BEFORE" ] && [ "$(wc -l <"$CHECKALL_LOG")" -gt "$CHECKALL_BEFORE" ]

echo "dm ledger flip/reconcile/lock/starve six-scene dry run PASS"
