# T-0391 自选报价条——NOX QuoteStrip/WatchlistModal 语义移植

- 状态：已批（2026-11-06 业主令——「订阅某1个symbol的实时报价，
  这个symbol需要我来定义，要支持搜索；NOX归档总览页就有，
  我要的是这个效果，前端要适合现有前端」）
  → 施工中（2026-11-06 devin-t0391 领单翻）→ 待验（收口同拍）
  → 完成（决策席裁决通过 merge d210e69a——迁移 0121→0122 顺延；
  全链绿戳对口）
- 领取：devin-t0391 @ 2026-11-06
- 优先级：P1——业主点名功能面
- 收口移交归：决策席
- 参照真源：/opt/nox/web/src/components/QuoteStrip.tsx+
  WatchlistModal.tsx（语义复刻，数据源/栈换 O 形）
- **形态决策（业主 11-06 二令）：实现的是 NOX 的功能，前端
  用 O 的前端**——QuoteStrip/WatchlistModal 仅作语义参照，
  视觉/组件/样式零拷贝，全走 O 现有 Card/Modal/表单件系

## 钦定语义（NOX 一比一）

1. **自选报价条**：QuoteBoard 卡内加「自选报价」段——tiles 显
   watchlist 成员的实时 bid/ask/last+鲜度（数据=md_ticks 最新，
   复用 ListPriceAlerts 同类 JOIN 或 ListQuoteWatchlist 响应携价）；
   零成员/零行情如实缺态不假零；
2. **「自选✎」浮层**：搜索框子串过滤候选（大小写不敏）→点选
   加入→↑↓排序+✕删→保存=全量替换即生效；
3. **候选域**=md_ticks DISTINCT inst_id（有行情才可选——NOX
   ListLatestQuotes 同义）；
4. **建提醒表单收口**：symbol 选择件=**可过滤输入**（input+
   datalist 式——敲子串即筛,几百品种裸 select 不可用——业主
   11-06 三令「缺少搜索功能」）；候选=watchlist 成员优先
   （成员空则退回全候选——不 fail-closed 卡死建单）。

## 范围

1. 迁移 `fin_0121_quote_watchlist.sql`：fin_quote_watchlist
   (symbol text PK, ord int, created_by, created_at)——fin 域
   正本（同 fin_price_alert 域钉）；migrate_table 挂 {121}；
2. RPC：GetQuoteWatchlist/SetQuoteWatchlist（origin 闸+seat=
   dashboard-ui 审计同 CreatePriceAlert 式；Set=全量替换）；
   候选面可并入 ListPriceAlerts.symbols 既有字段零新增；
3. 前端：QuoteBoard 内嵌报价条段+自选浮层（搜索/排序/删/存），
   形随现有 Card/Modal 件系——不引入 NOX 样式；ev_ticks 帧
   重取既有径；
4. DOM 桩测：搜索过滤/点选/排序/保存全量替换/tile 渲染缺态。

## 验收锚

① 搜索「trx」→候选含 TRX-USDT-SWAP 断言；
② 选存后报价条瓦片显成员价+非成员不建瓦片；
③ 保存全量替换真值重读断言（seat=dashboard-ui 审计行）；
④ 提醒表单候选=watchlist 成员断言；
⑤ check_all+前端测绿戳对口；并入发窗批。

## 红线

- 报价条=纯展示面——不喂判定/策略/采集域零动；
- 不动真单径；候选/持久皆在采域，不开任意 symbol 输入。

## 结果回填

实现摘要：
- 迁移 `fin_0122_quote_watchlist.sql`（fin 域正本：symbol PK/ord/
  created_by/created_at——ord=全量替换写序即展示序）+domain.tsv 册行+
  migrate_table.go {122} 登记；
- proto：GetQuoteWatchlist/SetQuoteWatchlist+QuoteWatchRow（成员×最新价
  携 has_quote/last/bid/ask/quote_ts/quote_age_s；候选域=md_ticks
  DISTINCT inst_id 并入响应 symbols）——buf 全谱正径再生（npm ci 后
  buf generate，非快照）；
- 后端 cmd/core/quotes_face.go：Get=成员 ord 序 LEFT JOIN md_ticks
  最新行（DISTINCT ON 同 alertRowSQL 口径；零成员/零行情如实缺态）；
  Set=earnOriginGate 同源闸→归一（大写 trim）去重→逐员在采校验（非
  在采=InvalidArgument 拒——候选域闸不开任意输入）→单事务 DELETE+
  INSERT ord=数组序→ops_event 审计行 seat=dashboard-ui event=
  quote_watchlist_set→真值重读返回（回显禁）；
- 前端 web/src/react/components/QuoteBoard.tsx：卡首「自选报价」段
  tiles（成员价/鲜度服务端派生龄直译；零成员/成员零行情如实缺态；
  成员序后端 ord 直渲零前端 sort；非成员不建瓦片）+「自选✎」浮层
  （.modal-overlay/.modal+watchlist-* 既有 CSS 件系——搜索子串过滤
  大小写不敏/点选加入/↑↓排序/✕删/保存=全量替换生效）+建提醒表单
  symbol 收口=可过滤输入 input+datalist（候选=watchlist 成员优先，
  成员空退回全在采——不 fail-closed）；
- 容器双取 listPriceAlerts+getQuoteWatchlist 合 useLive ev_ticks 帧
  重取（合并窗内——零定时器）。

测试证据：
- Go 桩测 TestQuoteWatchlistSetGetAudit（cmd/core/quotes_face_test.go）：
  零成员 rows 空+候选域=在采 2 员/Set 全量替换 ord 序真值重读/成员×
  最新价投影/审计行 seat=dashboard-ui/二次 Set 全量覆盖/非在采拒/
  Origin 缺席拒——PASS；
- DOM 桩测 quoteboard-domtest.mjs 23/23 PASS（搜索「trx」→候选含
  TRX-USDT-SWAP+已选剔出；瓦片渲价/零行情显缺/非成员不建瓦片；表单
  datalist 候选=成员集/零成员退回全在采；排序删件+全量替换文案）；
- check_all 全链绿戳 4dd3a894b1275a50fb4c58eaa58fbfa86212e5ac
  （go build/vet/gofmt/migrations 七断言/ops-chain/go test 全谱/npm
  typecheck+domtest 全绿）。

偏离说明：
- 票面钦 `fin_0121_quote_watchlist.sql` 挂 {121}——0121 已被
  fin_0121_ops_event_chain.sql 占（T-0390 后 master 现状），顺延
  fin_0122/{122}（check_migrations 链内唯一律强制，非自选）；
- 候选域未并入 ListPriceAlerts.symbols 而并入 GetQuoteWatchlist.
  symbols（票面「可并入……既有字段零新增」为可选径——浮层数据源
  本就是 GetQuoteWatchlist，候选随成员同帧刷新零新增 RPC）；提醒表单
  datalist 候选仍用 ListPriceAlerts.symbols 既有字段零改动；
- 瓦片无 venue 维（O md_ticks inst_id 单源——NOX venue 组语义天然
  不适用，语义等价=成员 tile 直列）；
- ✎ 浮层零 venue chips 子集（同上——md_ticks 无 venue 概念面）。
