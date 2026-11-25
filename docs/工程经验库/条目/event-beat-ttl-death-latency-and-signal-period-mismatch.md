---
schema_version: 2
id: event-beat-ttl-death-latency-and-signal-period-mismatch
title: 事件拍 TTL 死亡潜伏期=到期+下一事件间隔；等信号窗须按信号周期非拍距定
date: 2026-11-25
source: 工单O-39955（probe-armed_ttl P0——3 probe 全死 armed_ttl_expired）
kind: pitfall
risk: high
status: observed
last_verified: 2026-11-25
evidence_refs:
  - server_fact: fin_strategy_transition 287643/287604/287748 armed_ttl_expired×3（probe 4.1~4.9h 死）
  - server_fact: fin_paper_fill dir_perp_mom 同 inst 相邻 open 间隔 p50=127.5h/p90=170.3h/max=217.5h
  - commit: fix/39955-probe-armed-ttl
  - doc: internal/portfolio/lifecycle/run.go ListenLoop(ev_strategy/ev_eval) 事件化拍
  - doc: internal/portfolio/lifecycle/retire.go signalAlive
---

# 等信号型 TTL 的两层错配（armed_ttl 实证）

**事实/症状**：probe 语义=「晋级后持枪等入场信号」。armed_ttl=4h
纯时间窗杀「armed 永不触发占位」——但 dir_perp_mom 入场信号
（1D 动量 z≥1.5，hold 10d）同 inst 再发间隔 p50=127.5h/p90=170.3h，
窗与信号周期错配 ~30-40 倍：confGate 修通晋升的 probe 必死在
等信号途中（CRCL/MEME/CASHCAT 4.1~4.9h 三连实证）。

**第二层错配（取证副产）**：lifecycle scan=PG LISTEN/NOTIFY 事件拍
（ev_strategy/ev_eval 驱动+boot 首拍），非周期环——TTL 执法时刻=
「到期后下一 ev_* 事件」，静默窗内 probe 可超龄存活任意长。
COST-USDT-SWAP 活 12.8h **零活动锚**（exec_drill/execution/
position/eval_reject 全空）——票面假设「锚续命」证伪，真因=事件
真空。读 TTL 闸实测死亡龄时必须先问「拍跑没跑」。

**根因**：休眠判据用「时间」量「等信号」——信号周期是市场量，
拍距/TTL 是工程量，两量纲不可比；且「等信号」本身是活动不是休眠。

**处置（O-39955 径B）**：休眠候选先过信号死判——signalAlive 复用
entryGate 族判词同函数（dirEntryGate=本向 eval 仍 proposed<72h /
fundingEntryGate=率同号 / volEntryGate=报价守立项半值），活着=豁免
本轮退役，死=armed_ttl_expired 照退。判据一源律：禁第二套信号数学。
**非 perp 向平凡放行≠信号证**——default 臂仅 direction 带
short_perp/long_perp 前缀时采信 fundingEntryGate，否则不豁免
（否则非 perp 族 probe 永生=墓碑锁复现，spec 禁线直违）。

**证据**：桩测五锚绿+变异证明（摘豁免判据两锚转红）；lifecycle
全包 33 测绿；S0 三实据报数见收口判词行。

**反证条件**：①信号存续锚若被证「eval proposed 陈行但信号实际已死」
（臂停>72h 误判活）——72h 界是臂拍距量级，dirmom 扫 240min 下
eval 每日棒刷新，陈行=臂真停非信号强；②若未来某族 entryCondition
判词语义改为「有条件延开」而非「信号死判」，signalAlive 直调会把
「暂时市况拦」误读为「信号死」——宏观爆点闸已显式排除（市况≠死），
新增同类闸须同法处理。

**适用边界**：仅 armed_ttl 策略面（live/probe 休眠候选）；提案面
armed_ttl_h 24h 卫生径、paper_ttl 168h、confGate/entryGate 判据本体
零触。非 perp 向 default 臂维持旧行为（无信号存续证面=照退）。
