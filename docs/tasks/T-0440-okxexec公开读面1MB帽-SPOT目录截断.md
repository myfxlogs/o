# T-0440 okxexec 公开读面 1MB 帽——SPOT 目录截断族级让位

- 状态：已批（业主令 2026-11-07——「自选报价条编辑搜不到XRP现货」实证）
- 归口：外派外审径（工程面——行情读径容量，不碰钱径判词）

## 实证链（2026-11-07 实拍）

- 前端「自选报价条编辑」搜 XRP 无现货候选——候选域=md_ticks 活符∪md_tick_scan 目录符；
- md_tick_scan 实拍：`inst_type` 分布 swap=528 futures=6 **spot=0**；
- OCI collector 日志每拍：`[okxscan] 粗扫族让位: SPOT 族: 宇宙目录读败——fail-closed 不虚构谱域: resp parse: unexpected end of JSON input`；
- OKX 实测：`/public/instruments?instType=SPOT`=**1,306,838B** vs SWAP=582,841B；
- 帽位：`internal/trading/okxexec/client.go:122` `io.ReadAll(io.LimitReader(resp.Body, 1<<20))`——
  公开读面 1MB 帽，SPOT 目录必截断→族级 fail-closed 每拍让位（SWAP/FUTURES 不触发）；
- 传染面：同一 PublicGet 还喂 `scanFam` 的 `/market/tickers?instType=SPOT`（ticker 响应更大）——
  即使目录闸过了 tickers 也会截；一处帽修两径同愈。

## 钦定活

① `okxexec/client.go` 公开读帽升到安全水位——`1<<23`（8MB）起步；帽保留（防失控体）
   不写死无限；注释钉「OKX instruments SPOT 2026-11 已 1.3MB 增长中」；
② 截断语义补强（可选同件）：`io.ReadAll(io.LimitReader(r, N))` 截断静默——
   读满帽仍 parse 败才报错，建议在帽处改 `LimitReader(N+1)`+len>N 显式「响应超帽」错，
   让下次同类病不再以 JSON 语法错伪装；
③ 验收三断言：check_all 绿戳对口 + OCI collecto 日志 `粗扫拍 spot=N>0` 实证 +
   `md_tick_scan` spot>0 且含 `XRP-USDT`；前端自选搜索 XRP 出现现货候选为终验。

## 顺手账（写票面回填）

- `internal/platform/notify/calendar.go:115` 同款 1MB 帽——经济日历响应远小暂不炸，
  不扩面；只在坑条目记「LimitReader 截断静默」模式。
