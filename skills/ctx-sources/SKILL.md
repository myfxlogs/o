---
name: ctx-sources
description: ctx 外部上下文采集面——ffcal/FRED/frankfurter/CFTC COT/OKX OI 五源端点账/节奏/鉴权形态+断供探针阈账，FRED v1/v2 界桩实证与自造 UA 禁令（Akamai 拉黑定案）
domain: 工程
---

# ctx 外部上下文源能力账（实测 2026-11-03 初账；T-0429 2026-11-07 断供根修+补维扩面）

## 五源端点账（T-0429 后）

| 源 | 端点 | 鉴权 | 自然节奏 | 落库 | 坑 |
|---|---|---|---|---|---|
| ffcal 日历 | forexfactory weekly XML（nfs.faireconomy.media/ff_calendar_thisweek.xml） | 无 | 周文件滚动 | md_ctx_fact(source=ffcal)+md_macro_event（桥表） | **机房 ASN 限流狠**——自造 UA 被 429/tarpit；密拍无益；**XML 头声明 windows-1252——Go xml 无 CharsetReader 拒读**（2026-11-07 定案修复，KB go-xml-charset-reader-non-utf8） |
| FRED | `api.stlouisfed.org/fred/series/observations`（v1） | `api_key` query 参数（32 位，存 SG app_secrets `fred_api_key` 密文；2026-11-07 OCI 实测键活） | 日/周/月参差（DGS10/DGS2/SP500/NASDAQCOM/DCOILWTI/DHHNGSP 营业日；UNRATE/CPIAUCSL 月） | md_ctx_fact(source=fred, kind=macro\|equity\|commodity, **dedupe_key=序列:观察日**——T-0429 根修，原键=序列名首拍占坑恒静默，KB dedupe-key-missing-version-bit-silent-stall) | **禁自造 UA**；err 脱 URL 防泄 key；密钥 finCfg.Secret() 每拍自取自愈；逐列 fail-soft（单列败不拖余列） |
| frankfurter FX | `api.frankfurter.dev/v1/latest?base=USD&symbols=EUR,JPY,GBP,CNY` | 无（ECB 公共面） | 日更（~16:00 CET 营业日） | md_ctx_fact(source=frankfurter, kind=fx, dedupe_key=USD:率日) | T-0429 新源（fx 补维）；独立上游——FRED 断供不连坐；OCI 实测 200 快速 |
| CFTC COT | Socrata | app token 可无 | 周更（周五 3:30pm ET） | md_ctx_fact(source=cftc, dedupe_key=行 id) | 字段方言照 ark 抄件 |
| OKX OI | OKX v5 REST | 无（公开面） | 连续实时 | md_ctx_fact(source=okx_public, dedupe_key=日戳) | 口径见 okx-api-spec skill |

## equity/commodity 补维取舍（T-0429 实测案）

- **stooq `q/l` CSV：机房 IP 全 404**（单/多符号、裸/编码 caret 全试）——死端勿再探；
- **Yahoo `query1.finance.yahoo.com/v8/finance/chart`：机房 IP 429**（Edge: Too Many Requests 首触即拒）——死端勿再探；
- 定案：equity/commodity 走**既有 FRED 臂扩列**（SP500/NASDAQCOM/DCOILWTI/DHHNGSP）——键/端点/transport 2026-11-07 OCI 实测全通，免鉴权优先让位「可用性实证优先」（机房 ASN 出网采公开面先天打折——fred-akamai 条目同判）。

## 断供探针阈账（T-0429——feedstale.go 钦定形，ops_event 观测盲/CTX-<源>）

| scope | 源 | 阈 | 节律判据 |
|---|---|---|---|
| CTX-FFCAL | ffcal | 10d | 周历滚动（新事件 slug 周级） |
| CTX-FRED | fred | 96h | 营业日级（**周末缺口 72h——48h 每周末假报**，故 96h） |
| CTX-FX | frankfurter | 96h | ECB 营业日 |
| CTX-COT | cftc | 9d | 周五周报（缺席一周+2d 富余） |
| CTX-OI | okx_public | 48h | 日戳行（缺一日即报） |

- 探针挂 ctx TickOnce 尾槽（采集后验鲜）；升件写 ops_event（fin 池——review_rw INSERT 实证）；败=log 自报非 fatal（errc=岗终径）；零行史=断供极端形同报；24h 同窗去重；只报不修。
- 2026-11-07 死因案（FRED 止 id976）：**主病灶=dedupe 键无版本位**（成功也零新行零日志）；诱因=11-03 边池 h2 stream INTERNAL_ERROR（fred-akamai 条目族，间歇性）；凭据期排除（KEY_LEN 32+实测 200）。

## FRED v1/v2 界桩（探针实证，勿再重探）

| 面 | 鉴权 | 端点 | 语义 | 判 |
|---|---|---|---|---|
| v1 | `api_key` query | `/fred/series/observations` | 按序列取点 | **我方正道**——8 序列×最新 2 点，对口（T-0429 扩列：macro 4+equity 2+commodity 2） |
| v2 | `Authorization: Bearer <key>` 头 | `/fred/v2/release/observations?release_id=N` | **整 release 批发**（全序列全历史+cursor 翻页+序列元数据） | 整史回填/vintage 训练才用；序列散落多 release 下按点取数不对口 |
| v2 series | — | `/fred/v2/series/observations` | **不存在**（Bearer/query 双径 404 实证） | 死端勿探 |

- v2 Bearer 头的结构性好处：**key 永不进 URL**——从结构上消灭 URL 泄键类坑；将来重写 fetcher 可参照
- v2 按 release 取数示例：release_id=18=H.15 利率（DGS10/DGS2 所在）、50=就业报告（UNRATE）、10=CPI——我方序列散落多 release，批发不对口
- 上游参照件 ark-intelligent `internal/service/fred/fetcher.go` 同走 v1 series 径——v1 非将就件

## 通用坑（跨源钦定）

- **自造 UA 禁令**：`o-ctx/1.0`/`o-research` 自命名 UA 被 Akamai 边池拉黑——h1 tarpit 挂死+h2 stream INTERNAL_ERROR（四 UA 隔离实验定案：curl/Go 默认 UA 秒通）。**一律栈默认 UA，禁自造**。ffcal 429 同根。详见 KB `fred-akamai-edge-go-transport-reject`
- 排障纪律：**UA/请求头列进变量矩阵第一轮**——客户端栈/协议/节点全排完才查 UA=鬼打墙实录
- 采集臂 TickOnce 串行序 cal→fred→fx→cot→oi+探针尾槽：单源失败记 fetch_err 不阻塞；ffcal 若 tarpit 会拖住后排（现 6h 共拍，分源自适应拍=将来优化件）
- 日志红线：永不打含 `api_key` 的 URL/query——err 只打 `http get: <err>` 脱敏文
