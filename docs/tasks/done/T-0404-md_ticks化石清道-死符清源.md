# T-0404 md_ticks 化石清道——死符清源

- 状态：完成（决策席裁决通过 merge 011e7d6c——全链绿戳对口）
- 领取：待施工
- 优先级：P3——数据卫生活件（0399 闸落地后清源才安全）
- 收口移交归：决策席

## 实证

SG md_ticks 514 行 476 陈（>1h）——重启瞬态全谱首跳
化石积案（10-25 17:32 单秒 169 行等大爆簇）；
OCI 144 行 105 陈（sweeper 扫掉更老的残留）；
upsert 最新行表——退订符行永不删。

## 范围

1. 清道件：md_ticks 陈行清扫——口径=ts 陈于阈（钦定
   7 天——远大于任何合法退订回溯窗；mt5 慢源按其节拍
   自适）+**不在活跃集**（activeSet UNION 反查——活跃符
   哪怕暂停流也留行）双条件；
2. 挂入 sweeper 日拍（o-sweeper 同式——SG 侧扫 md_ticks
   陈行删；OCI 侧同理已 partial 补齐口径）；
3. 闸面联动：0399 活性闸先于清源上线（防误删活符行
   后闸误判——序钦定：0399 合→本件清）。

## 验收锚

① 清扫桩测：陈+非活跃删/陈+活跃留/新行留三态对；
② 生产实扫一轮后 md_ticks 总行≈活跃集（对拍断言）；
③ check_all 绿戳对口。

## 红线

- 双条件缺一不可——禁按时间一刀切；
- sweeper 既有清道节律/日志面照式；
- 清源只删 tick 行——信号/评估/提案面零动。

## 结果回填

- 状态：待验
- 实现摘要：
  1. `internal/ledger/retention/sweep.go` `deadTailSweep` 谓词升正——
     离集判据由 `fin_active_set` 窄表（仓∪非 retired 策略触发器维护，
     且 md 链缺席：clean 装 md 库无此表=谓词探针恒 abort 永不删；旧库
     化石残表=谓词陈旧）改 `legset.ActiveSetUnion` 唯一真源反查（提案
     fin_proposed/fin_armed∪live/degraded 策略∪dcd/basis/funding-hedge
     三族派生现货腿∪`fin_sub_request` open 臂——与 okxws 订阅集/
     projector 镜像谓词同链）；「`ts` 陈于 7d ∧ 键不在活跃集 UNION」
     双条件钦定口径落地，谓词探针同步改 UNION 可执行性自检（读败
     fail-safe 整扫 abort 不删）。
  2. 日拍挂载=既有面零动——core `startRetention`→`Sweep`→
     `deadTailSweep`（SG fin 池）、`cmd/sweeper` 同码（OCI md 池），
     双池同件双条件补齐（票件②「OCI 侧同理已 partial」收为全量）。
  3. `scripts/check_import_direction.sh` 白名单补
     `retention→platform/legset` 边（方向评审：消费面→platform 真源，
     与 hunger/okxws/projector 已有 legset 边同形，三条独立红线零触）。
- 测试证据：桩测 `TestDeadTailSweep`（sweep_test.go 锚④）——
  陈+离集删（md_ticks DEAD-USDT/md_mark_index DEADMK-USDT 同式）/
  陈+活跃留（SUB-USDT 走 `fin_sub_request` open 臂——`fin_active_set`
  窄谓词误删面实证；LIVE-USDT 走 live 策略臂）/新行留（FRESH-USDT），
  retention 包全绿；check_all 全链绿戳对口（绿戳
  `check-green-e565681a84ae630f5cc57275679e32153dfadb15`；
  注：三席并发全谱期测试 PG 争用曾现 Migrate IO 等待型 30m 超时
  ×5 包，错峰复跑全绿=环境 flake 非码因）。
- 偏离说明：
  ① `md_tick_scan`（T-0401 粗扫层最新行表）不入清扫面——其「全谱
    非活跃符」行是常态语义非化石，套双条件即误删，范围守票面
    md_ticks+既有 md_mark_index；
  ② 陈阈取 `ts` 字面钦定（0399 的 `recv_ts` 是提醒读径防时钟偏移
    判据——删径同 fin_0079 先例取事件时刻，双条件下活跃符零误删）；
  ③ 验收锚②「生产实扫后总行≈活跃集」属部署后生产对拍面，归决策
    席验收执行（OCI md 池 UNION 各臂表=bridge/bridge-reverse 镜像
    两链皆在，md 侧投影滞延方向=多留非误删 fail-safe）。
