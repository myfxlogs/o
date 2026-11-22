#!/usr/bin/env bash
# regimedigest.sh —— LLM 宏观体制读数器官·机械层（工单O-32603）。
# 两职责（dispatch.sh 每拍调，细胞件不立 resident_units——同拍距同宿主）：
#   A 对拍段：horizon 已满的 regime_digest 读数→取同期 md_klines BTC 1D
#     确认棒实际走向对拍 risk 轴命中→量表/regime_score 行（命中率同帧
#     载——LLM 幻觉/漂移病在案，无对拍=不可信输入，spec 证伪度件钉死）。
#   B 到期段：判「体制读数该产了」→量表/regime_due 标件行（唤知会话
#     消费面=dispatch.sh trigger 携 regime:due→按 regime_prompt.md 契约
#     产 regime_digest 行；标件在末读数落地前幂等不重复）。
# 到期判据（写死常量）：末 regime_digest 龄>43200s(12h) ∨ 指纹异∧龄>7200s
#   (2h)；指纹=md5(BTC/ETH/SOL 末确认1D收盘|md_funding 7d均值)——市场面
#   输入变了才提前唤，ops_event 计数不入指纹（24h 事件流恒变=永到期）。
# 去重窗：同标签+同置信 24h 内不重复落行=契约层（模板 NOT EXISTS 判据
#   自带，唤起侧由「末读数落地→标件消」天然节流）；horizon=读数自带
#   龄界，超窗读数=陈账不算证（对拍段只对龄满者评分）。
# fail-open：库写败=本拍无产（digestspawn/mine 同例，调用侧 ||true）。
# 用法：dispatch.sh 每拍调；桩测 regimedigest_test.sh。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0

# ── A 对拍段：horizon 满+宽限(24h→3d/7d→9d 棒结算)后评分，未评分行只评一次
psql "$DSN" -qAt <<'SQL' 2>/dev/null || true # tolerate: 对拍写败=下拍补,唤起面不受阻
WITH cand AS (
  SELECT d.id, d.ts, d.note,
    (regexp_match(d.note,'regime=([a-z_+]+)'))[1] regime,
    (regexp_match(d.note,'conf=(low|mid|high)'))[1] conf,
    (regexp_match(d.note,'horizon=(24h|7d)'))[1] horizon
  FROM ops_event d
  WHERE d.event='量表' AND d.ticket='regime_digest'
    AND d.ts + CASE WHEN (regexp_match(d.note,'horizon=(24h|7d)'))[1]='7d'
                    THEN interval '9 days' ELSE interval '3 days' END < now()
    AND NOT EXISTS (SELECT 1 FROM ops_event s
      WHERE s.event='量表' AND s.ticket='regime_score'
        AND s.note LIKE '%regime_digest#'||d.id||' %')
), px AS (
  SELECT c.id, c.regime, c.conf, c.horizon,
    (SELECT k.c FROM md_klines k WHERE k.venue='okx' AND k.inst_id='BTC-USDT-SWAP'
       AND k.bar='1D' AND k.confirm=true AND k.ts<=c.ts ORDER BY k.ts DESC LIMIT 1) p0,
    (SELECT k.c FROM md_klines k WHERE k.venue='okx' AND k.inst_id='BTC-USDT-SWAP'
       AND k.bar='1D' AND k.confirm=true
       AND k.ts<=c.ts+CASE WHEN c.horizon='7d' THEN interval '7 days' ELSE interval '24 hours' END
       ORDER BY k.ts DESC LIMIT 1) p1
  FROM cand c
)
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '探测席','量表','regime_score',
 'regime_digest#'||p.id||' hit='||COALESCE(h.hit,'na')
 ||' ret='||COALESCE(round(((p.p1/p.p0)-1)*100,2)::text,'?')
 ||'% label='||COALESCE(p.regime,'?')||' conf='||COALESCE(p.conf,'?')
 ||' horizon='||COALESCE(p.horizon,'?')
 ||'——对拍md_klines BTC-USDT-SWAP 1D确认棒 risk轴(薄=|ret|≤3%);'
 ||'hit_rate30d='||COALESCE((SELECT round(avg((s.note ~ 'hit=1')::int)*100,1)::text
     FROM ops_event s WHERE s.event='量表' AND s.ticket='regime_score'
       AND s.note ~ 'hit=[01]' AND s.ts>now()-interval '30 days'),'?')||'%'
FROM px p
LEFT JOIN LATERAL (
  SELECT CASE
    WHEN p.p0 IS NULL OR p.p1 IS NULL OR p.p0=0 THEN 'na'
    WHEN (regexp_match(COALESCE(p.regime,''),'risk_(on|off|thin)'))[1]='on'
         AND (p.p1/p.p0-1)>0 THEN '1'
    WHEN (regexp_match(COALESCE(p.regime,''),'risk_(on|off|thin)'))[1]='off'
         AND (p.p1/p.p0-1)<0 THEN '1'
    WHEN (regexp_match(COALESCE(p.regime,''),'risk_(on|off|thin)'))[1]='thin'
         AND abs(p.p1/p.p0-1)<=0.03 THEN '1'
    WHEN (regexp_match(COALESCE(p.regime,''),'risk_(on|off|thin)'))[1] IS NOT NULL THEN '0'
    ELSE 'na' END hit
) h ON true
RETURNING note;
SQL

# ── B 到期段：判据 SELECT 出 flag|fp，再条件 INSERT 标件（幂等=末读数后零标件）
DUE=$(psql "$DSN" -qAt -c "
WITH lastd AS (
  SELECT ts,note FROM ops_event WHERE event='量表' AND ticket='regime_digest'
  ORDER BY id DESC LIMIT 1
), fp AS (
  SELECT md5(concat_ws('|',
    (SELECT string_agg(k.inst_id||'='||round(k.c::numeric,4)::text,',' ORDER BY k.inst_id)
     FROM (SELECT DISTINCT ON (inst_id) inst_id,c FROM md_klines
           WHERE venue='okx' AND bar='1D' AND confirm=true
             AND inst_id IN ('BTC-USDT-SWAP','ETH-USDT-SWAP','SOL-USDT-SWAP')
           ORDER BY inst_id, ts DESC) k),
    (SELECT COALESCE(round(avg(funding_rate)::numeric,8)::text,'null') FROM md_funding
     WHERE venue='okx' AND ts>now()-interval '7 days'
       AND inst_id IN ('BTC-USDT-SWAP','ETH-USDT-SWAP'))
  )) h
)
SELECT CASE WHEN lastd.ts IS NULL THEN 1
  WHEN extract(epoch from now()-lastd.ts)>43200 THEN 1
  WHEN fp.h IS DISTINCT FROM (regexp_match(lastd.note,'fp:([0-9a-f]{6,32})'))[1]
       AND extract(epoch from now()-lastd.ts)>7200 THEN 1
  ELSE 0 END||'|'||fp.h
FROM fp LEFT JOIN lastd ON true" 2>/dev/null)
FLAG=${DUE%%|*}; FP=${DUE#*|}
[ "$FLAG" = 1 ] && [ -n "$FP" ] && [ "$FP" != "$DUE" ] || exit 0
psql "$DSN" -qAt -v fp="$FP" <<'SQL' 2>/dev/null || true # tolerate: 标件写败=下拍补
INSERT INTO ops_event(seat,event,ticket,note)
SELECT '探测席','量表','regime_due',
 format('regime读数到期 fp=%s——唤知会话按 scripts/cognition/regime_prompt.md 契约产 regime_digest 行(evid 须载 fp:%s)', :'fp', left(:'fp',8))
WHERE NOT EXISTS (
  SELECT 1 FROM ops_event d WHERE d.event='量表' AND d.ticket='regime_due'
    AND d.ts > COALESCE((SELECT max(ts) FROM ops_event
      WHERE event='量表' AND ticket='regime_digest'),'-infinity'::timestamptz))
RETURNING note;
SQL
