---
schema_version: 2
id: dedupe-key-missing-version-bit-silent-stall
kind: pitfall
risk: high
status: verified
keywords: [dedupe_key, ON CONFLICT DO NOTHING, 幂等, 版本位, 观察日, FRED, 增量水印, 断供假象]
scope: [research, ctx, 数据面幂等设计, 增量消费]
summary: dedupe_key 不带数据版本位（观察日/行 id）时，append-only 表对同一逻辑对象恒首拍占坑——后续采集成功也全走 ON CONFLICT DO NOTHING 零新行，下游 id 水印增量消费恒枯竭=「断供」假象且零日志不可辨（transport 死与结构静默同形）
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: ticket
    ref: T-0429
  - type: commit
    ref: t/0429-ctx-inlet
---

## 定案

md_ctx_fact FRED 行 dedupe_key=序列名（md_1084 幂等键设计）——首拍成功
（2026-11-03 id973-976 四序列）后，**后续任何成功拉取都命中唯一键
DO NOTHING**：新观察日不落新行、payload 永停在首拍值。下游分析席
`id>(evidence->>'ctx_max_id')` 增量水印恒读不到 FRED 新件。外象与
transport 断供完全同形（零新行+零日志——键/端点/transport 实测全通），
生产「止 id976 断供」主病灶即此。

## 判据（幂等键设计律）

append-only+增量水印消费的表，dedupe_key 必须含**数据版本位**：

- 时序数据 → 版本位=数据自身日期（FRED 现形 `序列:观察日`）
- 批报告 → 版本位=上游行 id（CFTC 现形，本就正确）
- 快照日更 → 版本位=日戳（OKX OI 现形，本就正确）

版本位缺=首拍占坑恒静默——「幂等」做成「只此一拍」。

## 排障教训

transport 死与结构静默同象不可辨时，按「**消费视角**」验：下游水印是否
长期不进=结构病先验，再验 transport/凭据/端点（本轮三实测全通反推结构病）。
DB-only 视角（「行停了」）会误引去查 Akamai 族（skill 界桩在册病）白绕。

## 证据

- check: 生产 md_ctx_fact fred 恒 4 行（id973-976，11-03 13:42）；键/端点/
  transport/凭据 OCI 实测全通（Go 400 业务响应/curl 200/KEY_LEN 32）
- result: 测试 TestFredDedupeNewObsDate 修前红（n=0）→ 键含观察日后绿
  （新观察日=新行）
- ref: T-0429 收口 commit 链；skills/ctx-sources FRED 行

## 反证条件

- check: FRED 部署后行数随观察日推进（日级新行）
- falsifies_when: 序列停更（上游真断供）致零行=探针（CTX-FRED 96h）接管，
  本条只辖「成功仍零行」形
