---
schema_version: 2
id: active-set-dual-owner-arm-drift
kind: pitfall
risk: high
status: verified
keywords: [活跃集, 谓词对拍, 双端同语义, 漏臂, 静默滤, UNION 臂, refreshSets, activeSet, 归一臂]
scope: [谓词集, projector 投影臂, okxws 订阅面, dcd/basis 现货腿归一, 跨域表 JOIN]
summary: 同一逻辑集（活跃集标的）在采集订阅谓词与投影谓词各持一份 SQL——单端补归一臂而另端漏配=该臂标的数据被谓词静默滤在桥外（零错零痕，靠下游缺数反查才显形）。同语义集多持有方变更必须全方对拍；跨域表 JOIN 臂移植前须先对账目标域表可用性与镜像覆盖面（覆盖面=喂集谓词域须自洽）。
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: task
    ref: T-0250（projector refreshSets 归一臂补齐——T-0216 根因②根修）
  - type: doc
    ref: internal/projector/projector.go refreshSets
  - type: doc
    ref: internal/data/okxws/subscribe.go:75-83（collector activeSet 四臂）
  - type: task
    ref: T-0216（collector 侧 activeSet ③④臂返修——同语义 projector 侧漏配为审计残差）
---

## 事实/症状

T-0216 给 collector `activeSet`（internal/data/okxws/subscribe.go）补了第三/四
UNION 臂（dcd 策略 uly 归一+basis 两族现货腿归一），但**同语义的另一份实现**
——projector `refreshSets`（internal/projector/projector.go）——没有同步补臂。
serve 谓词下 basis/dcd 策略的现货腿（BTC-USDT 等）不在投影活跃集，md_ticks
现货 tick 被滤在桥外：SG 镜像零行、零错零日志，直到 basis paper 记账价源
断供排查才显形（T-0216 根因②=审计残差未捕段）。

## 根因或策略解释

- **同一逻辑集两份 SQL、零机检对拍**：活跃集标的的语义是「需要实时行情的
  标的集」，但采集端（订什么）与投影端（搬什么）各写各的 UNION——臂增删
  只改一端，另一端静默漂移。补臂工单只审了被改端的 diff，未把「同语义集
  全持有方」列为对账面。
- **跨域 JOIN 臂移植有域可用性前置**：collector 臂 JOIN `md_dcd_product`
  是 PG-md 域表；projector 侧查询打 SG fin 库——盲抄 SQL 会 relation does
  not exist（T-0187 fin_position 同坑型）。本例 SG 侧有桥面对应物（fin_0049
  同形镜像表，projector 自写），但**覆盖面=喂集（r.dcd 谓词域）**：JOIN 臂
  谓词域 `dcd%` 而喂集原为 `='dcd_call'`——域不齐=覆盖面缺口，同型静默滤
  换个姿势重演。移植 JOIN 臂必须核「目标域表在否+行覆盖面=谓词域否」两点。
- **静默滤=可观测性黑洞同族**：谓词集少一个元素不报错不计数，只有下游
  「价源陈/缺」症状反推——同 proposal-dedup 条静默吞类。谓词集变更的验收
  锚应是「归一产物入集断言」（桩测断集内/集外两端），非「查询不报错」。

## 复用面

凡「同一逻辑集在 ≥2 处各有 SQL/代码实现」的面（订阅集/投影谓词/风控名单/
保留账）：①补臂/删臂工单的对账面必列全部持有方，diff 对 diff 逐臂核；
②跨域移植含 JOIN 的臂先查目标域表可用性（桥面对应物/覆盖面与谓词域自洽），
禁盲抄；③谓词集桩测断「归一产物入集+离集不入」双端——只断输入行不算
验过归一变换。
