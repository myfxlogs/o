#!/usr/bin/env bash
# unit_boundary_test.sh —— 工单O-32638 词面闸普查B档①回归锚：
# 唤起 unit= 消费边全谱必须带右界 ([^0-9A-Za-z_-]|$)——sessionpulse/
# doing_face 已带界=对照范式；无界 LIKE %unit=X% = unit=o-x 前缀互吞
# +散文提及误消费同族面（colon/aseat/dedup 三案同族第四发防复发）。
# 判据=逐点钉：文件×谓词形双断言（界在场 ∧ 旧无界形缺席）。
set -u
cd "$(dirname "$0")/../.." || exit 1
PASS=0; FAIL=0
ok()  { PASS=$((PASS+1)); printf '  ok   %s\n' "$1"; }
bad() { FAIL=$((FAIL+1)); printf '  FAIL %s\n' "$1"; }

# $1=文件 $2=必含锚串 $3=必不含旧形（空=跳）
chk() {
	local f="$1" anchor="$2" stale="${3:-}"
	[ -f "$f" ] || { bad "$f 缺席"; return; }
	grep -qF "$anchor" "$f" || { bad "$f 缺界锚: $anchor"; return; }
	if [ -n "$stale" ] && grep -qF "$stale" "$f"; then
		bad "$f 残留无界形: $stale"; return
	fi
	ok "$f 界锚在"
}

SUB="substring(e.note from 'unit=[^ ]+')"
BOUNDSH="||'([^0-9A-Za-z_-]|"
STALE="x.note LIKE '%'||${SUB}||'%'"

# ── 案① substring 提取族（5 shell + 3 Go = 8 处）──
for f in scripts/cognition/dispatch.sh scripts/cognition/dispatch_research.sh \
         scripts/cognition/dispatch_worker.sh scripts/cognition/dispatch_self.sh \
         scripts/audit/dispatch.sh; do
	chk "$f" "${SUB}||'([^0-9A-Za-z_-]|" "$STALE"
done
for f in cmd/core/selfaware_beat.go cmd/core/panelv2_face.go cmd/core/audit_esc.go; do
	chk "$f" "${SUB}||'([^0-9A-Za-z_-]|" "$STALE"
done

# ── 案② dispatch_worker 兜底收口配对（quoted-heredoc 形，$ 无需转义）──
chk scripts/cognition/dispatch_worker.sh "'unit='||u.unit||'([^0-9A-Za-z_-]|$)'" \
	"x.note LIKE '%unit='||u.unit||'%'"

# ── 案③ run.sh 壳兜底自查（double-quote -c 形，\$ 转义在案）──
chk scripts/cognition/run.sh '([^0-9A-Za-z_-]|\$)'"'"'' \
	"note LIKE '%unit="

# ── 案④ 对照范式不回退：sessionpulse/doing_face 已带界须仍在 ──
chk scripts/cognition/sessionpulse.sh "unit=\$unit([^0-9]" ""
chk cmd/core/doing_face.go "unit='||e.ticket||'([^0-9]" ""

# ── 案⑤ selfgap 锚位契约：消费谓词=研究票消费锚同形 ^(求知|研究席) ──
chk cmd/core/selfgap_esc.go "t.note ~ '^(求知[:：]|研究席[:：])'" "t.note ~ '求知[:：]'"

# ── 案⑥ land_queue aseat 支名右界（散文支名直引不命中）──
chk scripts/land_queue.sh "('\$bre' || '([@[:space:]=]|" ""

# ── 案⑦ closeoutprobe U+3000 方言宽容双侧 ──
chk internal/portfolio/hunger/closeoutprobe.go "[[:space:]　]*完工回执" ""
chk internal/portfolio/hunger/closeoutprobe.go "工单#(\\d+)[\\s　]*完工回执" ""

printf '== 结果 PASS=%d FAIL=%d ==\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
