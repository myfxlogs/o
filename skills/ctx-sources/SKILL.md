---
name: ctx-sources
description: ctx 外部上下文采集面——ffcal/FRED/CFTC COT/OKX OI 四源端点账/节奏/鉴权形态，FRED v1/v2 界桩实证与自造 UA 禁令（Akamai 拉黑定案）
domain: 工程
---

# ctx 外部上下文源能力账（实测 2026-11-03，证据=docs/能力账/ctx-sources-probe-2026-11-03.md）

## 四源端点账

| 源 | 端点 | 鉴权 | 自然节奏 | 落库 | 坑 |
|---|---|---|---|---|---|
| ffcal 日历 | forexfactory weekly XML（ffcal_week_this.xml） | 无 | 周文件滚动 | md_ctx_fact(source=ffcal)+md_macro_event（桥表） | **机房 ASN 限流狠**——自造 UA 被 429/tarpit；密拍无益 |
| FRED | `api.stlouisfed.org/fred/series/observations`（v1） | `api_key` query 参数（32 位，存 SG app_secrets `fred_api_key` 密文） | 日/周/月参差 | md_ctx_fact(source=fred, dedupe_key=序列名) | **禁自造 UA**；err 脱 URL 防泄 key；密钥 finCfg.Secret() 每拍自取自愈 |
| CFTC COT | Socrata | app token 可无 | 周更（周五 3:30pm ET） | md_ctx_fact(source=cot) | 字段方言照 ark 抄件 |
| OKX OI | OKX v5 REST | 无（公开面） | 连续实时 | md_ctx_fact(source=oi) | 口径见 okx-api-spec skill |

## FRED v1/v2 界桩（探针实证，勿再重探）

| 面 | 鉴权 | 端点 | 语义 | 判 |
|---|---|---|---|---|
| v1 | `api_key` query | `/fred/series/observations` | 按序列取点 | **我方正道**——4 序列×最新 2 点，对口 |
| v2 | `Authorization: Bearer <key>` 头 | `/fred/v2/release/observations?release_id=N` | **整 release 批发**（全序列全历史+cursor 翻页+序列元数据） | 整史回填/vintage 训练才用；序列散落多 release 下按点取数不对口 |
| v2 series | — | `/fred/v2/series/observations` | **不存在**（Bearer/query 双径 404 实证） | 死端勿探 |

- v2 Bearer 头的结构性好处：**key 永不进 URL**——从结构上消灭 URL 泄键类坑；将来重写 fetcher 可参照
- v2 按 release 取数示例：release_id=18=H.15 利率（DGS10/DGS2 所在）、50=就业报告（UNRATE）、10=CPI——我方 4 序列散在三 release，批发不对口
- 上游参照件 ark-intelligent `internal/service/fred/fetcher.go` 同走 v1 series 径——v1 非将就件

## 通用坑（跨源钦定）

- **自造 UA 禁令**：`o-ctx/1.0`/`o-research` 自命名 UA 被 Akamai 边池拉黑——h1 tarpit 挂死+h2 stream INTERNAL_ERROR（四 UA 隔离实验定案：curl/Go 默认 UA 秒通）。**一律栈默认 UA，禁自造**。ffcal 429 同根。详见 KB `fred-akamai-edge-go-transport-reject`
- 排障纪律：**UA/请求头列进变量矩阵第一轮**——客户端栈/协议/节点全排完才查 UA=鬼打墙实录
- 采集臂 TickOnce 串行序 cal→fred→cot→oi：单源失败记 fetch_err 不阻塞；ffcal 若 tarpit 会拖住后排（现 6h 共拍，分源自适应拍=将来优化件）
- 日志红线：永不打含 `api_key` 的 URL/query——err 只打 `http get: <err>` 脱敏文
