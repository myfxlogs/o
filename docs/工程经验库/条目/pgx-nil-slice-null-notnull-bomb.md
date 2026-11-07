---
schema_version: 2
id: pgx-nil-slice-null-notnull-bomb
kind: pitfall
risk: medium
status: verified
keywords: [pgx, nil切片, NULL, NOT-NULL, 空集coerce]
scope: [go, pg, quotes_face, 写径]
summary: pgx v5 nil []string 编码 NULL 非空数组——NOT NULL text[] 列 INSERT 必炸；写库前显式 coerce 空切片
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: provenance
    ref: T-0442 TestQuoteWatchlistSetGetAudit 500→修后 200（cmd/core/quotes_watch_face.go venues coerce）
---

## pgx nil 切片=NULL——NOT NULL text[] 列写径炸弹

- **实证**：T-0442 `fin_quote_watchlist.venues text[] NOT NULL` 写径，成员无源约束时
  Go 侧 `[]string(nil)` 直传——pgx v5 把 nil 切片编码为 SQL NULL（非 `'{}'`），
  INSERT 撞 NOT NULL 违例→RPC 500（`SetQuoteWatchlist` 全链 500，测试
  `Set 应 200: code=500` 实锚）。
- **教训**：pgx 对 nil 切片/nil map 的编码语义=NULL 而非零值集合；「NOT NULL DEFAULT
  '{}'」列的写径必须在应用层显式 coerce（`if v == nil { v = []string{} }`）——
  DEFAULT 只管列缺席不管显式 NULL。
- **收口**：写径 coerce 空切片后测试复绿（TestQuoteWatchlistSetGetAudit PASS）。
