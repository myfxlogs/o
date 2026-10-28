# T-0213 basis 通用评估器永判 unevaluable——evalBasis 不发 apr，撞 T-0205 硬闸

> 状态：待施工 | 优先级：P1（basis 族评估永不可 proposed，卡链事实死锁）
> 立项：决策席 @ 2026-10-28（假想卡 adjudication 取证实证）

## 病灶

`internal/research/grammar/run.go` `evalBasis`：基差超阈时写
`verdict=proposed` + evidence `{fut,spot,basis_pct}`——**无 apr/net_apr**。
`writeEval` 的 T-0205 硬闸把无 APR 的 proposed 全量翻成
`unevaluable/apr_est_missing`。SG 实证：BTC-USD-261225/
ETH-USD-261225 三模板各落 apr_est_missing 行（2026-10-28）——
basis 族结构上不可能产出 proposed。

数据面已就绪（非病灶）：盒 PG-md `md_mark_index` 有 BTC/ETH/SOL/
XAU-USD[-UM]-261225 季度期货 mark-price 行且 <1min 鲜度，现货腿
BTC-USD/ETH-USD 同表有行。评估查询 15min 窗内两条腿都在。

## 钦定修法

evalBasis 在 evidence 里补 `net_apr`——年化基差
（basis_pct / 距到期天数 × 365，到期日解析自 inst_id `-YYMMDD` 尾；
无到期日解析=不写 proposed）。阈判改用年化 APR 口径对齐
funding-hedge 的判据维（而不是裸 basis_pct>0.5%）。

## 验收锚

- 桩测：fut/spot 行齐+基差超阈→verdict=proposed 且 evidence 含
  net_apr；基差不足→rejected/basis_below_min。
- 生产：grammar 下一拍后 `fin_opp_eval` basis 模板出现非
  unevaluable 判词行。
