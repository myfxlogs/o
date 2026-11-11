---
schema_version: 2
id: standing-beat-fact-blind-spot
kind: pitfall
risk: high
status: verified
keywords: [零命中, 恒值信号, 盲区, venue接入, pair_zero, okx.symbols, 通配, UniverseCatalog, escEdgeUV, ops_event]
scope: [internal/venue/onboard, cmd/core, 探测薄壳, 感知环, okx.symbols]
summary: 「如实记」beat 是感知环死端——恒零事实每拍落行但非升件，无会话工作队看得见（mt5:3001157 草案 57 连零 28h 业主点破方知）；根因=okx.symbols=['*'] 通配被字面消费当 base 配对，根修=AttachUniverse 缝接 T-0416 供给件+pairZeroEscCheck 恒零探补盲区
first_seen: 2026-11-11
last_verified: 2026-11-11
skills: [mt-accounts, okx-api-spec]
evidence_refs:
  - type: ops_event
    ref: "venue接入 beat=草案 key=mt5:3001157 零命中 ×57 (2026-11-10 03:4x ~ 2026-11-11 07:5x UTC)"
---

## 现象或反模式

两层病灶同案并发：

**①通配字面消费（因）**：onboard `okxUniverse()` 裸读 `okx.symbols` 册得
`["*"]`——草案循环把 `"*"` 当 inst 名 `SplitN("*","-",2)[0]` 得 base `"*"`，
`MatchSymbol` 对 366 枚真 broker 符号（BTCUSDTm/BTCUSDm…）结构性永零。
后缀归一器（USDTM/USD/PERP…）一直在位无罪——宇宙侧根本没供给真标集。
T-0416 `UniverseCatalog` 供给件（/public/instruments 实读+TTL 缓存）本就是
'*' 钦定展开源，hypothesis 提名闸在同进程 dcd.go 装配——onboard 没接线，
第二套裸读消费面滋生。

**②恒值 beat 无会话入口（病）**：beatDraft 零命中「如实记不跃迁」每拍落
venue接入 行——行在账但在**会话扫描面之外**：进场协议只扫 `event='升件'`
+工单队，esc_class='接入败' 的 beat 行无人消费。恒零=常量不是事件→
边缘触发探针无盯→57 拍 28h 结构死局业主人工点破。与
standing-condition-as-event-churn 是镜像病灶：彼=常亮告警复火 churn，
此=常死事实零火盲区——常设态两侧（误报/漏报）都须探针化。

## 根因或策略解释

感知环设计裂缝：探针写事实行与探针升件是两条路。「如实记」哲学保证账真，
但没保证账被看——会话面=事件面，恒值事实要有**第二枚探针**把「同条件
持续 N 拍跨 T 时」折算成事件，盲区才闭合。规则：凡「如实记不跃迁」的
beat 类，写侧须自问「连记多少拍=该升的结构性」并配恒值探；单点修个案
（如本工单直接配对）=治标，盲区原样留给下一 scope。

## 处置

- `Runner.universeFn` 缝+`AttachUniverse`（onboard.go）：`okxUniverse()` 遇
  册含 `'*'` 元素→缝展开；显式清单=钦定子集直用不展；缝 nil/目录读败=
  空集 fail-closed（beatDraft 让位同语义——不虚构标的集）
- `cmd/research/dcd.go`：uniCat 提出变量共享，`onboard.New(...).AttachUniverse(uniCat.Symbols)`
  ——同供给件禁第二套（T-0416 律）
- `pairZeroEscCheck`（cmd/core/supply_esc.go，escLoop 载）：36h 窗零命中
  草案 beat ≥10 ∧ 首零>12h ∧ 末零<2h ∧ 末零后无非零 beat=升件
  `数据缺/pair_zero`；fp="@" 回合纹（standing-condition-as-event-churn
  三纹型之三——同回合销后不火,复绿再死=新回合照火）
- 桩测 `TestOkxUniverseWildcard`：通配缝 nil→空集/缝挂→展开/显式清单→
  直用/目录败→空集 四锚

## 证据

- check: O_TEST_PG_DSN go test onboard（新锚+全链桩测）
- result: 2 passed；生产对拍 mt5:3001157 57 连零 evidence 链
- ref: 本批 commit；probed_symbols 366 行（BTCUSDTm 等在册）实证

## 反证条件

- check: venue接入 beat=草案 连零 ≥10 跨 12h 后升件面查 `数据缺/pair_zero` 行
- falsifies_when: 恒零达阈无升件（探针被旁路/谓词漂）∨ 通配册再被字面消费
  （新消费面绕 universeFn 直读 okx.symbols）

## 适用边界

venue 配对面+一切「如实记」beat 类。显式 okx.symbols 清单不展缝（owner
钦定子集语义不动——'*' 之外元素即全集）。通配判据=册含 '*' 元素即整册
按全谱展（混写 ['*','X'] 视同通配——'*' 语义即全，显式项被吞属册面自洽）。

## 伴生量表（T-0332 强制）

- gauge_name: pair_zero_stall
- class: 探针健康
- query: SELECT esc_scope,count(*) FROM ops_event WHERE event='venue接入' AND note LIKE 'beat=草案 %零命中%' AND ts>now()-interval '36 hours' GROUP BY 1 HAVING count(*)>=10
- threshold: ≥10 连零 36h=达阈应升件；升件面 24h 内无 pair_zero 行=探针失效人工复核
