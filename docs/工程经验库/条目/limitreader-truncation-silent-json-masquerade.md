---
schema_version: 2
id: limitreader-truncation-silent-json-masquerade
kind: pitfall
risk: high
status: verified
keywords: [LimitReader, 截断静默, 响应超帽, JSON语法错伪装, unexpected end of JSON input, 容量帽, fail-closed, 粗扫让位, SPOT目录, T-0440, httpread]
scope: [internal/trading/okxexec, internal/platform/httpread, internal/platform/notify/calendar.go, internal/research/ctx, internal/research/scout, scripts/meta/gauges_tick.sh]
summary: io.ReadAll(io.LimitReader(r,N)) 读满帽静默截断零报错，下游 JSON parse 以「unexpected end of JSON input」伪装成解析病——容量病被误诊；okxexec 1MB 帽截断 OKX SPOT instruments（1.31MB）→SPOT 族 fail-closed 每拍让位、spot=0 靠业主人工对拍发现。对治=LimitReader(N+1)+len>N 显式「响应超帽」错（httpread.Body 共享件），新 venue 客户端禁各抄 LimitReader
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: server_fact
    ref: 2026-11-07 实拍——OKX /api/v5/public/instruments?instType=SPOT=1,306,838B（SWAP=582,841B）；OCI collecto 每拍「[okxscan] 粗扫族让位：SPOT 族：宇宙目录读败——fail-closed 不虚构谱域——resp parse——unexpected end of JSON input」；md_tick_scan inst_type 分布 swap=528 futures=6 spot=0；前端自选搜 XRP 无现货候选
  - type: code_ref
    ref: internal/trading/okxexec/client.go doOnce 原 io.ReadAll(io.LimitReader(resp.Body, 1<<20))——T-0440 改 httpread.Body(PublicCap=8MB)+ErrOverCap 显式错；残留同款静默位：internal/platform/notify/calendar.go:115(1MB)/internal/research/ctx/ctx.go:113(8MB)/internal/research/scout/collect.go:37(8MB)
---

# LimitReader 截断静默——容量病伪装成 JSON 解析病

## 坑形

`io.ReadAll(io.LimitReader(r, N))` 读满帽**静默截断**：不报错、不提示，
截断后残缺体流入下游 JSON parse，报 `unexpected end of JSON input`——
表象=解析病，根因=容量病。错误面与真实病灶错位，排查绕远。

T-0440 实证链：okxexec 公开读面 1MB 帽 → OKX SPOT instruments 增长至
1.31MB 必截断 → 目录读败 fail-closed（不虚构谱域——闸设计正确）→ SPOT 族
**每拍让位零行落表**，spot=0 且前端自选搜不到现货候选。SWAP/FUTURES 未
超帽不触发——族级差异=唯一线索。发现靠业主人工对拍（自选编辑搜 XRP），
非告警面——「部分族让位」仅 docker 日志一行（无人值守面事实失明同族病）。

## 判据

- 见「unexpected end of JSON input」/JSON 语法错先问**体量**：响应本应
  多大？读径有无 LimitReader 帽？帽值 vs 实测响应大小——别先修解析；
- 同一读径部分族/部分端点炸、部分好——超帽族增长的信号（目录类响应
  随上市数增长，SWAP/SPOT 体量可差 2 倍+）。

## 处置律

1. 读面帽必须配**截断显式错**：`LimitReader(N+1)` 多读一字节探测，
   `len>N` 即显式「响应超帽」错（errors.Is 可判）——共享件
   `internal/platform/httpread.Body`（T-0440③b）；
2. 新 venue 客户端**禁各抄 LimitReader**——消费共享件继承同构防御
   （抄=病跟着抄；抄一份静默截断=埋一颗同类雷）；
3. 帽保留不写死无限（防失控体），帽值按公开目录增长水位定（8MB 起步
   ——SPOT 2026-11 已 1.31MB 且增长中）；
4. 族级 fail-closed 让位须配**水位闸**兜底（`internal/platform/scanwm`
   ——注册面派生期望族集，零鲜落持续 ≥10min 升件）——让位可观测，
   靠日志/人肉对拍发现=缺席无感族。

## 残留面（在案不扩）

calendar.go:115（1MB，经济日历响应远小暂不炸）、research/ctx:113 与
research/scout:37（8MB，同款静默形态）——不扩面同票修；下次触碰时改
消费 httpread.Body。
