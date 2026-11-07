---
schema_version: 2
id: connect-wire-protojson-camelcase-vs-go-tag-snake
kind: pitfall
risk: medium
status: verified
keywords: ["connect", "protojson", "json.Unmarshal", "camelCase", "snake_case", "多词字段静默零", "测试断言假绿", "protoc-gen-go json tag"]
scope: [测试面, connect RPC 桩测, proto 载荷断言, dashboardv1]
summary: connect 线上响应=protojson marshal（lowerCamelCase 键），而 protoc-gen-go 生成 Go 结构的 json tag=snake_case 原名——测试里用 encoding/json.Unmarshal 解 wire 字节，多词字段（venue_symbol/ct_val/spec_hash…）静默全零、单词字段（base/digits/score…）恰好命中，断言面「半绿」极难肉眼辨（venue_symbol 丢但 base 在——像字段漏设非解码病）。wire 载荷反序列化唯一正径=protojson.Unmarshal（同 proto 类型），json.Unmarshal 只许用于非 proto 结构
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: ticket
    ref: T-0432
  - type: commit
    ref: t/0432-symbol-dialect
---

# wire protojson 与 Go json tag 命名分叉——测试反序列化用错解码器=多词字段静默零

## 病灶

T-0432 方言对表面桩测：connect HTTP 径回包（SymbolCandidate/SymbolVerifyEvidence）用
`json.Unmarshal` 解进 proto 生成型。病形=**多词字段静默零、单词字段恰好命中**：

- 线上键（protojson）= `venueSymbol`/`contractSize`/`specHash`（lowerCamelCase）
- Go 结构 json tag（protoc-gen-go）= `venue_symbol`/`contract_size`/`spec_hash`（snake 原名）
- `base`/`digits`/`score`（单词）两边同名——恰好解码成功

结果=断言「半绿」：venue_symbol 丢但 base 在、ct_val 丢但 bid/ask 在——像字段漏设
非解码病，肉眼极难辨（T-0432 TestListSymbolCandidatesRPC/TestSubmitSymbolMappingFullLoop
双测试同中）。Go tag 形参证：解码失败的恰是全部下划线字段。

## 对策

1. **wire 载荷反序列化唯一正径=`protojson.Unmarshal`**（同 proto 生成型，CamelCase 键
   正读）；`json.Unmarshal` 只许用于非 proto 结构（本地 map/DTO）。
2. 断言面写锚时先打 `%+v` 全形——只见单词字段=第一嫌疑解码器错配，不是业务缺。
3. proto 载荷「跨面契约 proto 唯一源」（架构律）的对称面=**解码也 proto 唯一源**——
   marshal 用 protojson、unmarshal 也 protojson，半套混用=本病灶。

## 关联

- skills/mt5-quotes-history（venue 方言坑同票——后缀两层剥实勘同 commit）
