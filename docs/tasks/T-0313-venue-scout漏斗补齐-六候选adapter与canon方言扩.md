# T-0313 venue-scout 漏斗补齐——六候选 adapter 接入+canon 方言扩

> 状态：派单中 | 立票 2026-11-03 | [角色:决策]

## 背景

venue-scout 度量面 10 候选中 6 家 metrics 空（state='candidate' 无 adapter
不采）：bitget / gate / kucoin-futures / mexc / dydx-v4 / deribit。度量窗口
30d 滚动——每晚接一天少一天，漏斗口径=唯一信息增益来源。bybit 单日
8.24% 已量，候补池扩 4→10 是同一窗口内机会面 2.5× 的最便宜路径。

## 范围

1. collect.go adapters 册 +6 件（fetcher 签名同 collectBinance 先例）
2. 各端点字段方言以**实测响应为准**（协议面取证纪律——禁猜字段名）；
   种子 endpoints 在 fin_venue_candidate.endpoints 可查作起点，端点不对
   允许实测后换更好端点并在 evidence 记
3. interval_h 如实填（端点有取端点值/无则 nil——禁虚构；dydx=1h、
   deribit 连续结算模型注意口径换算或如实 nil）
4. canon() 方言扩：kucoin M 尾（XBTUSDTM→XBT）+XBT→BTC 别名族、
   deribit PERPETUAL 尾——每扩一条附测试用例
5. 升 measuring：新 md 迁移 UPDATE 六 venue state='measuring'（OCI 正本
   →SG 投影自动跟——不改 fin_0098/md_1082 原种子件）
6. 六 adapter 各配 scout_test.go 桩测（照现存三例）

## 验收硬迹

- OCI 部署后 md_venue_funding 六 venue 实落行（贴 count+max(ts)）
- 下一度量拍后 fin_venue_candidate.metrics 六家非空 boards 实迹
- 单 venue 挂不拖全局（venue_down 计数+log 已有制式）
- gates：check_migrations/check-env-reads/go build/相关 test 全过
