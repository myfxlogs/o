# T-0391 自选报价条——NOX QuoteStrip/WatchlistModal 语义移植

- 状态：已批（2026-11-06 业主令——「订阅某1个symbol的实时报价，
  这个symbol需要我来定义，要支持搜索；NOX归档总览页就有，
  我要的是这个效果，前端要适合现有前端」）
  → 施工中（2026-11-06 devin-t0391 领单翻）
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

（施工写：实现摘要 / 测试证据 / 偏离说明）
