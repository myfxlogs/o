# T-0168 cmd/projector 投影臂——活跃集谓词七镜像表+fin_proposal 双向（D-0030 §三/§四）

## 态：可领（T-0167 ✅）
## 立项：决策席 @ 2026-10-26
## 决策档：D-0030

## 施工
1. `cmd/projector`：LISTEN PG-md 本地 ev_*（零轮询）→活跃集谓词查最新行→
   写 SG 镜像表（ON CONFLICT DO UPDATE WHERE ts>EXCLUDED.ts——per-table
   键照 domain.tsv；md_macro_event 无 ts 用 event_date 键语义，D-0032）。
2. 活跃集来源：SG 侧 `fin_active_set`（fin_0069 已建名——fin_position
   closed_at IS NULL ∪ fin_strategy 非 retired 触发器维护，ev_active_set
   NOTIFY）——projector 订阅帧刷谓词；**启动/重连全表 reconciler**（D-0032：
   谓词自算 fin_position/fin_strategy 不赖种子——同径兼治 0069 无回填）。
3. **八桥表**镜像谓词照 D-0030 §三+domain.tsv bridge 册（md_macro_event
   第八桥含）。
4. fin_proposal 双向：PG-md→SG 投影全表；SG state/retire_reason/retired_at
   变更（**ev_proposal 双侧帧**——dev insert 与 SG 状态组更新各发，D-0032
   §三裁）→回写 PG-md；SG 属三列禁投影覆盖（唯一双向面——桩重点）。
5. projector_silent 告警（镜像 max(ts) 超阈）+ roster 注册。
6. 失效语义（D-0030 §七）：projector 死→SG 镜像 ts 陈→sizing/MTM 拒新动作
   fail-closed；既有仓平仓=SG 本地 MT5 feed+OKX 私有 WS 不赖镜像。

## 桩锚
- 活跃集外 inst 行→零投影；活跃集内→投影；SG 改 proposal state→PG-md 回写。
- projector 停→SG 镜像 ts 陈→sizing 拒开仓（T-0128 闸复用）实证。
## 验收锚
- 隧道载荷 ≤300 行/分（实拍）；镜像表 ts 新鲜 ≤60s；双向回写零丢；
  reconciler 重启自愈实证（杀连→全表对账→集齐）。
