#!/usr/bin/env bash
# unit_watch.sh —— 建制单元本机查件（工单#26444②判一供数臂——systemctl
# 实查必须在宿主本机：o-self.timer disable 静默躺尸 2.5d 实证坑根修）。
# 每拍：读注册表 ops.unit_registry（canon 单源——本机单元表）→ 逐单元
# `systemctl --user is-enabled/is-active` 实查 → 态变才写 ops_event
# 「单态」行（seat=建制机；常态静默零写防账洪——差分才落账即审计轨）
# + ops_headless beat 覆写（查件死活自证——探针侧 stale 判据此）。
# 用法: systemd timer 周期驱（o-unit-watch.timer 建议 5min）或手跑。
# 只报不动——复跳/enable 归处置席非本件（探针族只述事实同契）。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || { echo "[unit-watch] FIN_PG_DSN 读不出($ENVF)——fail-closed不猜" >&2; exit 1; }

REG=$(psql "$DSN" -qAt -c "SELECT value::text FROM app_config WHERE key='ops.unit_registry'" 2>/dev/null)
if [ -z "$REG" ]; then
	echo "[unit-watch] 注册表 ops.unit_registry 读不出/缺——让位本拍（探针侧 fail-closed 升件覆盖本静默）" >&2
	exit 0
fi

# 查拍参数+单元表（python3 解 JSON——注册表=canon，本件零内置名单）。
read -r BEAT_TICKET FACT_SEAT FACT_EVENT PERIOD <<< "$(python3 -c '
import json,sys
r=json.loads(sys.stdin.read())
w=r.get("watch",{})
print(w.get("beat_ticket","UNITW"), w.get("fact_seat","建制机"),
      w.get("fact_event","单态"), w.get("period_sec",300))' <<< "$REG" 2>/dev/null)"
[ -n "${BEAT_TICKET:-}" ] || { echo "[unit-watch] 注册表 watch 段解析败" >&2; exit 1; }

# 逐单元实查+差分落账（python3 出「unit|expect_en|expect_act」表逐行喂）。
python3 -c '
import json,sys
r=json.loads(sys.stdin.read())
for u in r.get("units") or []:
    print(u.get("unit",""), u.get("expect_enabled",""), u.get("expect_active",""))' \
	<<< "$REG" | while read -r UNIT EXP_EN EXP_ACT; do
	[ -n "$UNIT" ] || continue
	EN=$(systemctl --user is-enabled "$UNIT" 2>/dev/null || true) # tolerate: 查败/单元不在=落 missing 字面——差分判据如实非静默
	[ -n "$EN" ] || EN="missing"
	ACT=""
	if [ -n "$EXP_ACT" ]; then
		ACT=$(systemctl --user is-active "$UNIT" 2>/dev/null || true) # tolerate: 同上——查败落 missing 如实载
		[ -n "$ACT" ] || ACT="missing"
	fi
	# 期望判：enabled 必比；expect_active 钉才比（oneshot 常态 inactive 不钉）。
	DRIFT=0
	[ "$EN" != "$EXP_EN" ] && DRIFT=1
	[ -n "$EXP_ACT" ] && [ "$ACT" != "$EXP_ACT" ] && DRIFT=1
	# 差分契：实态≠期望→必落「单态」行；实态=期望但末行在案为异态
	# →补一行复常账（审计轨闭环——异→常转换不可无账）。同态不重写。
	LAST=$(psql "$DSN" -qAt -c "SELECT substring(note from 'enabled=[^ ]+ active=[^ ]*')
		FROM ops_event WHERE event='$FACT_EVENT' AND note LIKE 'unit=$UNIT %'
		ORDER BY ts DESC LIMIT 1" 2>/dev/null || true) # tolerate: 末行查败=空→按「无行」判（差分契下拍重查，beat 探针兜查件死）
	WANT="enabled=$EN active=$ACT"
	if [ "$DRIFT" = 1 ] && [ "$LAST" != "$WANT" ]; then
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('$FACT_SEAT','$FACT_EVENT','','unit=$UNIT enabled=$EN active=$ACT want_en=$EXP_EN want_act=${EXP_ACT:--} drift=1')" \
			>/dev/null 2>&1 || true # tolerate: 差分写径=尽力件——写败下拍重试，beat 探针兜查件死
	elif [ "$DRIFT" = 0 ] && [ -n "$LAST" ] && [ "$LAST" != "$WANT" ]; then
		psql "$DSN" -qAt -c "INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('$FACT_SEAT','$FACT_EVENT','','unit=$UNIT enabled=$EN active=$ACT want_en=$EXP_EN want_act=${EXP_ACT:--} drift=0')" \
			>/dev/null 2>&1 || true # tolerate: 同上
	fi
done

# beat 覆写（查件死活自证——ops_headless upsert 有界零增长；探针 stale 判据此）。
psql "$DSN" -qAt -c "INSERT INTO ops_headless(ticket,seat,state,last_activity)
	VALUES('$BEAT_TICKET','$FACT_SEAT','alive',now())
	ON CONFLICT (ticket) DO UPDATE SET seat=EXCLUDED.seat,
	  state=EXCLUDED.state,last_activity=now(),updated_at=now()" \
	>/dev/null 2>&1 || true # tolerate: beat=尽力径——写败不阻主径，下拍覆写自愈
exit 0
