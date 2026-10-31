---
schema_version: 2
id: grep-m1-o-multimatch-newline-tid
kind: pitfall
risk: high
status: verified
keywords: [grep, -m1, -o, 多匹配, 换行串, TID 提取, ops_event, 幻影行, split_part 归一]
scope: [/opt/meta/bin/dm-spawn.sh, TID 提取壳写法, ops_event 写径, pipeline 读径]
summary: grep -m1 -oE 的 -m1 限「第一个匹配行」非「第一个匹配」——一行多票号时 -o 行内逐匹配各吐一行，$() 捕获=换行拼接串原样灌库（ops_event.ticket 幻影值案）；单值提取须补 | head -n1；且管道后 grep 败=head 退 0，原 || echo 兜底失效，空值兜底改 ${VAR:-默认}；既有脏行读径以 split_part(col,E'\n',1) 归一兜（append-only 禁 UPDATE）
first_seen: 2026-10-31
last_verified: 2026-10-31
skills: []
evidence_refs:
  - type: server_fact
    source: postgres
    ref: ops_event id 722/733/736/739/740/743/745 ticket=换行串（dm-wake 批量件 prompt 一行三票号，2026-10-31）
  - type: code
    ref: /opt/meta/bin/dm-spawn.sh :22/:27（head -n1+${TID:-} 兜底修——PIPEFIX）
  - type: code
    ref: cmd/core/pipeline_face.go GetPipeline 活跃票聚合（读径 split_part(ticket,E'\n',1) GROUP BY/WHERE/SELECT 三处一致归一）
---

# grep -m1 -oE 多匹配吐换行串——TID 幻影行灌库

## 症状

驾驶舱管道幻影行占满 ≤20 行帽：ops_event.ticket 出现 `T-0276␤T-0276␤T-0276`、
`T-0270␤T-0277` 等换行拼接值，GROUP BY 按原值分组各立一行，真票挤出屏。

## 根因

`TID=$(grep -m1 -oE 'T-[0-9]{4}' "$PROMPT")` 两处误读：

1. **-m1 限「第一个匹配行」非「第一个匹配」**——-o 对命中行内每个匹配各吐
   一行；dm-wake 批量件 prompt 一行三票号 → $() 捕获三行换行串，report.sh
   原样灌库。
2. 若顺手补 `| head -n1` 而留 `|| echo 兜底`，管道后 `||` 绑的是 head 退
   0——无匹配时兜底静默失效，TID 变空串（次生病）。

## 对策

- **写径**：单值提取固定式 `$(grep -m1 -oE 'PAT' "$F" 2>/dev/null | head -n1)`
  + 空值兜底 `VAR=${VAR:-默认}`（两段式，勿用管道后 `|| echo`）。
- **读径兜既有脏行**：聚合 SQL 对 ticket 全用
  `split_part(ticket, E'\n', 1)` 归一（GROUP BY/WHERE/SELECT 三处一致）——
  append-only 库禁 UPDATE，脏行不清洗由读面归位。

## 回改点

- /opt/meta/bin/dm-spawn.sh :22/:27（写径根修现场，PIPEFIX 2026-10-31）。
- cmd/core/pipeline_face.go GetPipeline（读径归一+TestGetPipelinePhantomRowNormalized 回归锚）。
- 同病排查口径：仓内其余 `grep -m1 -oE 'T-'` 无 head 提取点（devin-relay.sh
  :119/:183、mbus-dispatch.sh :698/:735/:747 形同，票外未动——交决策裁）。
