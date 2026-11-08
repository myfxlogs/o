# T-0506 ops_event 写面语义闸——(seat,event) 合法性矩阵落库执行绝根修

- 状态：已批（业主令「从根因解决，下次不会再出现」）
- 级别：P2 机械轨写面完整面——T-0499 键形闸的同族姊妹件

## 病灶（2026-11-09 实证在案）

- COS 清账批 seat='COS' 写派单 5 票（T-0443/0479/0481/0482/0484）→
  `ops_ticket_phase` 视图照翻 dispatched（映射不看 seat），而
  `dm_queue.sh` 补拉候选滤 `latest_seat IN ('决策席','CTO')`——
  票呈「已派」而拉取窗全盲 **4h 零拉起**；
- 根因断层：相位投影 seat 宽 × 消费面 seat 窄，写面**零校验**——
  键形闸（T-0499 ops_event_key_gate.sh）只管 ticket 形，不管
  (seat,event) 组合合法性；裸 psql INSERT 更是一切闸全绕；
- 经验条目在案：dispatch-seat-whitelist-phase-lie。

## 判据

1. **合法性矩阵落库执行**：新迁移建 (seat,event) 白名单约束面——
   推荐 BEFORE INSERT trigger/约束表层（拒裸 psql，不靠写径自觉）；
   矩阵内容=现行钦定语义枚举（派单∈决策席/CTO、裁决∈决策席/CTO/
   裁定席/CEO/外审席、领单∈施工席族…），枚举依据=消费面既有 SQL
   判据逐条对拍成表，**不发明新语义**；
2. 违规写=loud 拒（ERROR 明文 seat/event/ticket 三元）——
   非 phase 驱动词（升件/处置/候水/送达等说明类）宽容放行，
   先钉「相位驱动词×seat」子集，全量收紧留观察窗；
3. 桩测：seat='COS'+派单→拒、seat='决策席'+派单→过、
   说明类事件任意 seat→过、历史合法写径零回归；
4. 机械轨/脚本写径全量过闸不炸（闸上线即真跑 mech_tick 一拍零拒——
   矩阵漏枚举=自残，桩测须先枚举在产全部 (seat,event) 存量组合
   `SELECT DISTINCT seat,event FROM ops_event` 对拍矩阵全覆盖）；
5. 全谱零新增 FAIL/SKIP。

## 禁线

- 矩阵以「消费面判据+存量组合」为准绳，禁凭直觉添删 seat；
- 不碰钱径；不改相位映射表本身（映射=钦定面）；
- 说明类事件（非相位驱动）不设闸——闸只罩会骗相位的写。
