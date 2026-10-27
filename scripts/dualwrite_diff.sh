#!/usr/bin/env bash
# dualwrite_diff.sh —— 双写对拍脚本（T-0169 ①：逐表比 count/max(ts)/抽样行）。
# 口径注记：PG-md 正本与 SG 旧库的 count 差=SG legacy 历史+SG 本地采集共写行
# （设计内差异，非病灶）；对拍判据=max(ts) 时滞+近窗增量 count 比+抽样键行形。
# 用法：MD_DSN=postgres://... SG_DSN=postgres://... scripts/dualwrite_diff.sh
# （SG_DSN 经隧道口；psql 取自 timescale 镜像 docker run——本机零宿主依赖）。
set -uo pipefail

MD_DSN="${MD_DSN:-postgres://o:o@127.0.0.1:5435/o?sslmode=disable}"
SG_DSN="${SG_DSN:?SG_DSN 缺——SG 侧经隧道 DSN 必填}"
WINDOW="${WINDOW:-10 minutes}"   # 近窗增量比对窗
FRESH="${FRESH:-300}"            # max(ts) 时滞告警秒（对拍期口径宽——非钱径闸）

PSQL="docker run --rm --network host timescale/timescaledb:2.30.2-pg18 psql"

q() { # q <dsn> <sql> → -tA 输出
	$PSQL "$1" -tA -v ON_ERROR_STOP=1 -c "$2" 2>&1
}

fail=0
# 表集：投影搬运八桥+config 副本（fin_data_retention 分库独立账不比对）。
TABLES="md_ticks md_mark_index md_signal md_dcd_product md_funding md_macro_event fin_proposal fin_opp_eval app_config"
LIVE_TABLES="md_ticks md_mark_index md_signal md_dcd_product"
# md_funding 不判时滞（结算拍键纪律——键静默期=正确行为非断流，T-0169 实证）

printf '%-18s %12s %12s %10s %10s %8s\n' TABLE MD_CNT SG_CNT MD_MAXLAG SG_MAXLAG SAMPLE
for t in $TABLES; do
	tscol="ts"; [ "$t" = "md_macro_event" ] && tscol="event_date"
	[ "$t" = "fin_proposal" ] && tscol="created_at"
	[ "$t" = "md_funding" ] && tscol="recv_ts" # funding ts=未来结算拍（帧语义）——新鲜度判 recv_ts
	md_cnt=$(q "$MD_DSN" "SELECT count(*) FROM $t" | tail -1)
	sg_cnt=$(q "$SG_DSN" "SELECT count(*) FROM $t" | tail -1)
	md_lag=$(q "$MD_DSN" "SELECT COALESCE(extract(epoch FROM (now()-max($tscol)))::int,-1) FROM $t" | tail -1)
	sg_lag=$(q "$SG_DSN" "SELECT COALESCE(extract(epoch FROM (now()-max($tscol)))::int,-1) FROM $t" | tail -1)
	md_win=$(q "$MD_DSN" "SELECT count(*) FROM $t WHERE $tscol>now()-interval '$WINDOW'" | tail -1)
	sg_win=$(q "$SG_DSN" "SELECT count(*) FROM $t WHERE $tscol>now()-interval '$WINDOW'" | tail -1)
	# 抽样行形：近窗键集全语句（镜像同 DDL 前提下逐字对形）
	case "$t" in
		md_ticks)        sample="SELECT COALESCE(string_agg(inst_id||':'||COALESCE(last::text,''),','),'∅') FROM (SELECT inst_id,last FROM $t WHERE $tscol>now()-interval '$WINDOW' ORDER BY $tscol DESC LIMIT 50) s" ;;
		md_mark_index)   sample="SELECT COALESCE(string_agg(inst_id||':'||channel||':'||COALESCE(px::text,''),','),'∅') FROM (SELECT inst_id,channel,px FROM $t WHERE $tscol>now()-interval '$WINDOW' ORDER BY $tscol DESC LIMIT 50) s" ;;
		md_signal)       sample="SELECT COALESCE(string_agg(kind||':'||key,';'),'∅') FROM (SELECT kind,key FROM $t WHERE $tscol>now()-interval '$WINDOW' ORDER BY $tscol DESC LIMIT 50) s" ;;
		md_dcd_product)  sample="SELECT COALESCE(string_agg(product_id||':'||COALESCE(annualized_yield::text,''),','),'∅') FROM (SELECT product_id,annualized_yield FROM $t WHERE $tscol>now()-interval '$WINDOW' ORDER BY $tscol DESC LIMIT 20) s" ;;
		md_funding)      sample="SELECT COALESCE(string_agg(inst_id||':'||COALESCE(funding_rate::text,''),','),'∅') FROM (SELECT inst_id,funding_rate FROM $t WHERE $tscol>now()-interval '$WINDOW' ORDER BY ts DESC LIMIT 50) s" ;;
		md_macro_event)  sample="SELECT COALESCE(string_agg(event_date::text||':'||kind,','),'∅') FROM md_macro_event" ;;
		fin_proposal)    sample="SELECT COALESCE(string_agg(id::text||':'||state,','),'∅') FROM (SELECT id,state FROM $t ORDER BY id DESC LIMIT 50) s" ;;
		fin_opp_eval)    sample="SELECT COALESCE(string_agg(id::text||':'||verdict,','),'∅') FROM (SELECT id,verdict FROM $t ORDER BY id DESC LIMIT 50) s" ;;
		app_config)      sample="SELECT COALESCE(string_agg(key,';' ORDER BY key),'∅') FROM app_config" ;;
	esac
	mdh=$(q "$MD_DSN" "$sample")
	sgh=$(q "$SG_DSN" "$sample")
	stamp="PASS"
	[ "$mdh" != "$sgh" ] && stamp="SAMPLE-DIFF"
	# max(ts) 时滞只判活写表（proposal/eval/macro/app_config 低频——lag 仅报告）
	case " $LIVE_TABLES " in *" $t "*)
		if [ "$md_lag" -ge 0 ] && [ "$md_lag" -gt "$FRESH" ]; then stamp="MD-STALE"; fail=1; fi
		if [ "$sg_lag" -ge 0 ] && [ "$sg_lag" -gt "$FRESH" ]; then stamp="SG-STALE"; fail=1; fi
		;; esac
	printf '%-18s %12s %12s %10s %10s %8s  win:%s/%s\n' "$t" "$md_cnt" "$sg_cnt" "${md_lag}s" "${sg_lag}s" "$stamp" "$md_win" "$sg_win"
	[ "$stamp" = "SAMPLE-DIFF" ] && {
		echo "    md:  ${mdh:0:180}"
		echo "    sg:  ${sgh:0:180}"
	}
done
if [ "$fail" -eq 1 ]; then
	echo "❌ dualwrite_diff: 时滞超阈（$FRESH s）——投影/旁写断流排查" >&2
	exit 1
fi
echo "OK: dualwrite_diff（时滞阈内；count 差=legacy/共写设计内差异——见头注）"
