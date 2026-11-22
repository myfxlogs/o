#!/usr/bin/env bash
# digestspawn.sh —— 研究 digest 发现词面→求知票自产闸（工单O-31997）。
# 病灶：处置席只对升件行本身关账——digest 内「达标候选/适配器提案」类
#   发现随销件沉没（O-27488 实案：contract_dca 提案对象被销作「交付载件」
#   无后续票，ops_event#31971；digest 原件 #31714 载「适配器提案对象」
#   词面）。spawn 规则缺席=回流全靠人肉。本件=拍前预扫机械层：
#   dispatch.sh 每拍调（落点取舍：a 拍前预扫优选——判据层钦定机械执行
#   不依赖会话判断；b 写径侧产票被否=digest 来源杂（外派回执/管内臂）
#   写径不唯一难钉全；c runbook 条文被否=会话判断层同病灶语义）。
# 谓词（写死判据，禁 LLM 判断面——发现词面集=机械正则）：
#   ops_event 行 seat='研究席' ∧ event='升件' ∧ note ~* 'digest'
#   ∧ note ~ 发现词面集（初版写死常量，可扩）：
#     达标候选|适配器提案|值得立项|提案对象|重评触发|够线
#     （spec 词面「够线清单≠∅」取词干「够线」——≠∅ 为集合记法非词面）
# 幂等：ops_ticket_desk 无 note 引 'digest#<id>'（数字边界 [^0-9]|$ 同
#   dispatch.sh 升件#id 销账判式）不复产——票=ops_event event='工单'
#   行经 desk 视图同源，INSERT 即入板，dispatch_research.sh 触发④
#   (^求知:|^研究席:) 自然拾取。
# 票形：seat='唤知席' event='工单' ticket=NULL
#   note='求知:<digest ticket|esc_scope|digest>——digest#<id>发现词面派生:<命中词>'
#   (前缀半角冒号=研究机拾取契;读端dispatch_research已宽容[:：]两形,#32020
#    实证全角票死等——写端正形+读端宽容双保险)
#   命中词=regexp_matches 全命中 DISTINCT '/' 连（机读可溯派生源）。
# 边界：digest 销件语义不动——本闸只产票，不触 ops_esc_pending/valence/
#   处置链；desk 视图只读引用不写；全件零 UPDATE/DELETE（append-only）。
# fail-open：库写败=本拍无产（dispatch 主触发面不受阻，调用侧 ||true）。
# 用法：dispatch.sh 每拍调；桩测 digestspawn_test.sh。
set -u
ENVF="${REPORT_ENV_FILE:-/opt/o-research/env}"
DSN=$(grep '^FIN_PG_DSN=' "$ENVF" 2>/dev/null | cut -d= -f2- | tr -d '"')
[ -n "$DSN" ] || exit 0

# 单句 INSERT...SELECT...RETURNING：候选扫描+幂等闸+票产同事务面，
# RETURNING note 回显=唤起 journal 可溯本拍产了哪几票。
psql "$DSN" -qAt <<'SQL' 2>/dev/null || true # tolerate: 自产闸写败=本拍无新票,digest销件链不受阻(dispatch主触发面照跑——fail-open同mine.sh例)
INSERT INTO ops_event(seat,event,note)
SELECT '唤知席','工单',
 '求知:'||COALESCE(NULLIF(e.ticket,''),NULLIF(e.esc_scope,''),'digest')
 ||'——digest#'||e.id||'发现词面派生：'||hit.kw
FROM ops_event e
JOIN LATERAL (
  SELECT string_agg(DISTINCT m[1],'/') AS kw
  FROM regexp_matches(e.note,
    '达标候选|适配器提案|值得立项|提案对象|重评触发|够线','g') m
) hit ON hit.kw IS NOT NULL   -- 聚合LATERAL零命中也返一行kw=NULL，ON真恒过=漏网；IS NOT NULL=词面闸
WHERE e.seat='研究席' AND e.event='升件'
  AND e.note ~* 'digest'
  AND NOT EXISTS (SELECT 1 FROM ops_ticket_desk t
    WHERE t.note ~ ('digest#'||e.id::text||'([^0-9]|$)'))
RETURNING note;
SQL
