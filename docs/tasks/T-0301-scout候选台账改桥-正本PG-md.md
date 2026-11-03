# T-0301 scout 候选台账改桥——fin_venue_candidate 正本 PG-md

| 字段 | 值 |
|---|---|
| 状态 | **已批**（决策席立单 2026-11-03——部署实证触发，scope 钦定如「钦定目标」） |
| 优先级 | P1（scout 臂当前 24h 一拍一拍败——不炸但度量全断；钱径无碍） |
| 关联 | T-0300（骨架票）/ D-0043 / 部署实证 2026-11-03 @o-core:339082400054 |

## 背景（部署实证，非推测）

o-core SG 部署+OCI 盒 o-research 换臂重启后（2026-11-03 05:26 UTC），研究域
验项全绿——fin 评估器新口径在产（`net_apr −0.33 < 阈 0.06，cost_apr=0.389，
cost_rt_bps=32`）、basis 两模板 `enabled=t`、md_1079/1080/1081 已落——
**唯 scout 臂每拍败**：`relation "fin_venue_candidate" does not exist`。

根因：`fin_venue_candidate` 在 T-0300 时被钉 fin 域（fin_0098 建表+seed 在
SG 生效），但 scout 的 `pool` 是研究主池（盒本机 PG-md :5435——"SG_PG_DSN"
名残）——研究臂惯例是**研究域表正本在 PG-md**（fin_opp_eval/fin_opp_template
同款桥表），fin 域正本表经双池（FIN_PG_DSN）或桥投影达 SG。scout 票施工时
未意识到台账表须走研究域桥规制——一票域判错位。

## 钦定目标（照 fin_opp_eval 桥范式，勿另创径）

1. **domain.tsv**：`fin_venue_candidate` `fin`→`bridge`（正本 PG-md / SG 镜像——
   同 fin_opp_eval 位格）。
2. **新 md 迁移**（md_1082 序取用）：PG-md 建 `fin_venue_candidate` 同形 DDL
   （列集/约束/注释照 fin_0098）+**同款 seed 10 行**（evidence/fee 值照抄——
   两端正本同源，幂等 `ON CONFLICT (venue) DO NOTHING`）。
3. **projector 加桥频道**：`fin_venue_candidate` md→SG 上行投影（upsert on
   venue 键——metrics/state/last_request_at 三列须随行；仿 projectEvals 形，
   水印键用 ts 或 venue 自然键自定——钦定约束：SG 侧 upsert 不得丢列）。
4. **scoutv1 Board 面零改**（SG 侧表仍在 fin_0098 落——成为投影终表）。
5. **验收锚**：
   - OCI 盒 `md_venue_funding` 出现 ≥3 venue 实采行（采集径自通即证）；
   - scout journal `拍败` 行消失、`fin_venue_candidate`（盒 :5435）出现
     metrics jsonb 写回行；
   - SG 侧 `fin_venue_candidate` 镜像行 metrics 非空（投影贯通实证）；
   - `check_bridge_registry`+迁移七断言+import 边全绿；复跑 scout 4 桩绿。

## 范围外

- 不改 scout 度量/判据/推荐口径（T-0300 已验收面）；
- 不动 SG fin_0098 已落表（作投影终表原位留）；
- 不动 o-collecto/o-projector 既有频道口径。

## 返修注记

施工席回填面含：迁移文件/DDL 对齐核验、projector 频道码、部署后三锚实证。
