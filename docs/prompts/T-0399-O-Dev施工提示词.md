# T-0399 O-Dev 施工提示词——提醒径鲜度闸

领单唯一权威范围：
/opt/o/docs/tasks/T-0399-提醒径鲜度闸-死符号拒建.md
（AGENTS/collab/施工席SKILL先读；范围红线即全部授权）

改点锚：`cmd/core/quotes_face.go`——ListSymbols 候选滤活+
CreatePriceAlert 校验活性同口径。鲜度判据先读
internal/research/hypothesis freshByKind 源节拍自适实现
复用不新造档（mt5 慢源不一刀切）。桩测红→绿先红桩
（修复类票闸）。

收口：新 wt→bootstrap→check_all 绿戳→收口文+推支不合并不部署。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | QuoteBoard 候选是否同一 RPC 未核 | 票面钦核复用 | 工人查 |
