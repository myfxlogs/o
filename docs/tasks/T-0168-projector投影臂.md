# T-0168 cmd/projector 投影臂——活跃集谓词七镜像表+fin_proposal 双向（D-0030 §三/§四）

## 态：候派（等 T-0167）
## 立项：决策席 @ 2026-10-26
## 决策档：D-0030

## 施工
1. `cmd/projector`：LISTEN PG-md 本地 ev_*（零轮询）→活跃集谓词查最新行→
   写 SG 镜像表（ON CONFLICT DO UPDATE WHERE ts>EXCLUDED.ts）。
2. 活跃集来源：SG 侧 `md_active_set` 小表（fin_position open ∪ fin_strategy
   非 retired 的 inst_id，trigger 维护）——projector 订阅其 ev 帧刷谓词。
3. 七镜像表投影谓词照 D-0030 §三表。
4. fin_proposal 双向：PG-md→SG 投影全表；SG state 变更（ev_strategy）→
   回写 PG-md（唯一双向面——桩重点）。
5. projector_silent 告警（镜像 max(ts) 超阈）+ roster 注册。

## 桩锚
- 活跃集外 inst 行→零投影；活跃集内→投影；SG 改 proposal state→PG-md 回写。
- projector 停→SG 镜像 ts 陈→sizing 拒开仓（T-0128 闸复用）实证。
## 验收锚
- 隧道载荷 ≤300 行/分（实拍）；镜像表 ts 新鲜 ≤60s；双向回写零丢。
