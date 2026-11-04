---
schema_version: 2
id: feed-fetch-unchecked-status-no-backoff
kind: pitfall
risk: high
status: verified
keywords: [HTTP StatusCode 未检, 429 错页当 XML, 限流续砸, 指数退避缺席, faireconomy CF 挡, DC-IP 信誉, 外部 feed 拉取]
scope: [notify/calendar, 外部 feed 采集面, Go net/http, SG 出向]
summary: 拉取面不查 resp.StatusCode——429/5xx 错页当 XML 进 decode 报「解包败」假相，真病灶（限流）被掩；且无退避面，15min 拍限流期续砸拉长 CF 封窗盖过事件预推窗=高影响事件漏推无痕
first_seen: 2026-11-03
last_verified: 2026-11-04
skills: [ctx-sources]
evidence_refs:
  - type: ticket
    ref: T-0296
  - type: doc
    ref: internal/platform/notify/calendar.go
---

## 事实/症状

2026-11-03 ~00:20–01:09 UTC SG 出向 IP 遭 faireconomy（CF 挡）间歇限流全
429（宿主 curl/容器 wget/换 UA 皆拒，01:09 自复 200）。calendar.Scan 不检
`resp.StatusCode`：429 HTML 页当 XML 进 decoder→「feed 解包败」log——
真病灶限流被「解包败」假相掩盖；15min ticker 限流期照打续砸（~96 次/日
非频率病灶但封窗期每拍都在给 DC-IP 信誉桶续命），封窗盖过事件预推窗=
当次漏推。`md_calendar_push` 末推 2026-10-30 09:47 UTC，10-30 午后→11-01
间 FOMC/PCE 类疑漏（旧容器日志灭失模式自洽）。

## 根因或策略解释

外部源拉取面缺三件套互为表里：①StatusCode 分支缺席——非 200 体误入
decode 径，错误语义层错位（传输层拒绝被报成载荷层解析败，定性全歪）；
②连败退避缺席——固定 15min 拍对限流源=续砸助封，无自伤止血的收缩面；
③UA 爬虫特征——Go 缺省 `Go-http-client/2.0` 在 CF 边=DC 爬虫低分桶
（fred-akamai 同族：自造名 UA 更糟，真实浏览器串/栈默认是正径）。

## 处置

T-0296 根修（calendar.go）：`!=200`→log 含码+`miss()` 连败 ×2 退避
（15m→30m→1h→2h→4h 封顶 `calBackoffMax`），`nextTry` 窗内 Scan 直退
零出向；200 到手归零回 15min 拍；传输错同桶连败。浏览器形 UA+Accept。
fetchErrs/decErrs/pushErrs 计数随 log 文案出。

## 证据

- check: `TestCalendarBackoffSequence`——429 桩连败退避 15m/30m/1h/2h/
  4h/4h 递增封顶逐档断言，窗内复扫 hits 不增=零出向，零推送；
  `TestCalendarRecoverPush`——403→退避→200 复常归零+事件照推。
- result: 全绿（notify 包 17.4s）。
- ref: T-0296 票面病灶段（2026-11-03 00:20–01:09 UTC 429 实测定时点）。

## 反证条件

- check: 部署后限流窗内 docker logs o-core——只见退避节拍 log（HTTP 4xx/
  5xx 连败 N 退避 Xs 后复拍），非每 15min 拉取败行。
- falsifies_when: 退避窗内仍每 15min 见打源行=退避闸失效；
  错页进 decode 报「解包败」再现=StatusCode 分支失效。

## 适用边界

凡外部 HTTP 拉取面（feed/REST 采集）皆适用「StatusCode 必检+连败退避」
铁律——不限于 ffcal。退避参数（15m 基/4h 顶）按源更新节律定，非通用常数。
