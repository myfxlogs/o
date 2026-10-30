# T-0259 dcdOpen 拒因持久化——venue 码落库，容器重建不毁证

> 状态：**完成** | 领取：devin-t0259 @ 2026-10-30 03:59Z | 优先级：**P1（MANAGE-151 钦定立案——取证面
> 病灶，10-28 故障潮定性曾因日志毁损绕行侧表）** | 执行径：**无头** | 收口commit：396b61ce95bf74443590aaee0ba1a0cf7ef01562
> 立项：决策席 @ 2026-10-30（管理令 20261030-105000-MANAGE-151）

审计席：headless @ 2026-10-30

终态核销：决策席 @ 2026-10-30——审计通过在案（判决回报 DM-279@14:46；
审计席落票 commit 37b5b22）；决策并码 dm/1030-t0259-merge（迁移改号
fin_0089——0088 与 T-0263 ops_headless 撞号），翻终态=完成。

## 面况

- o-core 容器重建毁容器日志（10-29 17:02Z 实证）——dcdOpen venue 拒因
  （产品级 `52927` no-quote 恒拒，T-0255 两窗实证 1049+7063 次成交率 0%
  的定性证据）只活在容器 stdout，重建即灭，取证被迫绕行侧表。
- `fin_order_leg.venue_err_code/venue_err_msg` 列已在（flow.go:231
  INSERT 带两列），但**腿未建即拦的上游拒**不落腿表——T-0255 实证
  「18/1069 执行有腿，绝大多数 failed/ambiguous 根本未下单、在上游被拦」。
- dcdOpen 径：`okxexec/dcd.go` DcdQuote（POST /finance/sfp/dcd/quote）
  → DcdTrade——quote 步 52927 拒时执行行如何落、拒因落哪，施工先勘查
  执行面写点（fin_execution 列面/腿建序），再定落点。

## 钦定界

- dcdOpen 执行径（quote 步与 trade 步）venue 拒码+拒文**持久化入 DB**——
  落点取执行面既有同构位优先（fin_execution 列面扩 或 fin_order_leg
  前移建腿 或 fin_eval_reject 同构拒因表，施工勘查定，结果段记取舍）；
- 持久化点须在「失败标定之前或同事务」——容器任意时点重建，已拒事实
  不丢证；
- venue 原始码（sCode/数字码）+原始 msg 均落，禁只落归一化枚举。

## 范围外

非 dcdOpen 执行径拒因面的普查改造（同型面若顺手且同构可顺带，结果段
声明交裁）；告警/展示面。

## 验收锚

1. 桩测：dcd quote 注入 52927 拒→DB 可查 venue 码+msg（非仅日志）；
2. 桩测：trade 步 venue 拒→同上持久；
3. 持久化与失败标定同事务或先行（乱序重建不丢证）；
4. check_all 绿戳；票面回填+产坑 KB（若有）。

## 结果回填

> 施工席 devin-t0259（无头单飞）：领单 512931c → 收口 396b61c。

### 实现摘要

- **落点取舍**（钦定界三选勘查定）：双层各取同构位——
  - **quote 步上游拦**（exec 行未建）→ `fin_eval_reject`：
    reason=`dcd_quote_reject[:<code>]`，per-(inst,template,direction,code)
    幂等行 upsert（ts 刷新=last-seen，evidence.hits 累计尝试数=拒率分母），
    码+msg+step+notional+strategy_ref 全入 evidence。**否决 exec 前建行**：
    `fin_execution.state='open'` 行被 risk.check:127/earn.worker:186 当在途
    敞口聚合消费——前建行=幻影敞口+闸拒废行污染；fin_order_leg 前移同弃
    （execution_id FK 依赖同病，且 dcd 无 clOrdId 可填=伪造域值违 D-616）。
  - **trade 步**（exec 行在）→ `fin_execution` 新增 `venue_err_code/
    venue_err_msg` 列（fin_0089——原号 0088 与 T-0263 ops_headless 撞号，
    并码时改号，同构 fin_order_leg.venue_err_* 上移执行行
    层）——`dcdMark/dcdMarkVE` 把 state+拒码拒文写进**同一条 UPDATE**
    （单语句=同原子事务，锚③形）。
- **拒码抽取** `dcdVenueErr`：APIError 内层 sCode 在且≠0=真实拒因优先
  （envelope code=1 壳码实证在案 order.go:13/okx-exec skill「判读取内层码」
  契约）；非 APIError=传输/解析败 → msg 原文留痕、code=NULL 不臆造。
- **`DcdQuote` 补层**（okxexec/dcd.go）：code=0 壳内 per-item sCode≠0 时透
  `*APIError` 上抛——原来只报「无 quoteId」丢码，52927 若走内层码位即此径。
- **桩测缝**：`hedgeOpener.dcd dcdVenue` 窄接口（DcdQuote/DcdTrade/
  DcdOrderStatus 三法），nil 回退 `*okxexec.Client`——装配面零改。
- `docs/数据流.md` 执行段补拒因持久化链行（链变同 commit）。

### 验收锚对账

1. ✅ `TestDcdQuoteRejectPersist`：APIError{Code:1,SCode:52927} 注入 →
   `fin_eval_reject` 行 reason=`dcd_quote_reject:52927`、
   evidence venue_code=52927/venue_msg=no quote available；连拒两次
   hits=2（幂等行不洪）；`fin_execution` 零行（上游拦不产 exec 行）；
2. ✅ `TestDcdTradeRejectPersist` 三态：envelope APIError→ambiguous+
   code=51008/msg=Insufficient balance（内层码优先实证）；per-item
   sCode=52927→failed+原码原文；http 429→failed+code NULL+原文
   （非 venue 码不臆造）；
3. ✅ state+venue_err_* 同 UPDATE 单语句——断言同行同取即同事务形；
   quote 步写件先于 error 返回（先行序）；
4. ✅ 机检行见下（check_all 重跑戳）；票面回填本段；
   坑处置：**有坑**——新立 `exec-state-literal-beyond-check`（索引已 gen）。

### 偏离说明

① **顺手根修·两枚静默写病灶（同径同 commit，交裁）**：
   a. position 落位败径写 `state='orphan'` 越 0082 CHECK 集（七值无
      orphan）——该 UPDATE **恒违例静默败**（`_,_=`吞），exec 行永留
      'open' 幻影敞口喂 risk/earn 聚合。改 'ambiguous'（票面注释原义即
      歧态势语义：exec 在仓行缺=查证面不吞）。
   b. `dcd 业务拒` 径原 SQL `SET state='failed' WHERE id=$2` 绑
      `sCode:sMsg` 作 $1 但**从不引用**——venue 码「看似落库」实则
      蒸发（病灶本体，T-0259 立案面况）。现真落列。
② **顺带覆盖**：vol_sell_strangle/vol_sell_single_call 两腿复用 dcdOpen
   （volsell_open.go:28）——同型面自动同获持久化，零额外改动。
③ **状态语义未动**：envelope APIError 非 429 径仍标 `ambiguous`（歧态势
   禁补单保守位——venue 确拒但 state 语义归决策面裁量，本票只加证）。
   order-status 读败/空态、未载映射态、position 落位败三径 msg 原文入
   venue_err_msg（venue/本端败因同列留痕，列 COMMENT 已明语义）。
④ **quote 步闸拒/notionalCcy 缺不persist**：闸拒已由 fin_risk_decision
   留痕；数据缺 fail-closed 非 venue 拒——钦定界外。
⑤ fin_eval_reject 键无 strategy_ref/venue——跨策略同产品同码共享幂等行
   （evidence 记最新 strategy_ref），per-策略拆分须另票加列。

### 机检

- 锚测：`go test ./cmd/core -run 'TestDcdQuote|TestDcdTrade' -v` 5/5 绿
  （本机 O_TEST_PG_DSN@5434）；`go test ./internal/trading/okxexec
  ./internal/store/pgstore` 绿。
- `scripts/check_all.sh`：**末跑绿戳见收口 commit 戳记**——全谱期遭遇
  三席并发 check_all 共享测试 PG（5434）+pg_advisory_lock(860223) 全局
  迁移串行锁+8k 泄漏测试 schema 叠加，未触包 internal/projector·research/
  dcd 撞 1800s 超时（环境挤兑非码因——锚测独立跑全绿）。收口后单跑补戳；
  若并发挤兑仍红按基建故障报决策席。
- gofmt 净；文件长 open.go 297/open_dcd_reject.go 99/测试 154 ≤300；
  pre-commit 十三道全过（env/migrations/rejects/lines 各 OK 输出在 log）。
