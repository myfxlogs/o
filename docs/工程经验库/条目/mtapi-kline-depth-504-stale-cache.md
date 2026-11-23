---
schema_version: 2
id: mtapi-kline-depth-504-stale-cache
kind: pitfall
risk: medium
status: verified
keywords: [PriceHistoryExMany, NumBars 504, 史缓存陈化, md_klines venue=mt5, klineBatch, Exmany深度阈, 会话生命周期]
scope: [internal/mt5 klines 面, md_klines venue='mt5' 数据面, mtapi 网关深史拉取]
summary: mtapi PriceHistoryExMany 存在深度超时阈（NumBars≥500→nginx 504，≤200 实测活）且 venue 侧 PriceHistory 史缓存按会话生命周期冻结——会话长存则回棒窗末端停在缓存建窗日，全簿行同截=陈窗签名非逐符缺史。
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: [mt5-quotes-history]
evidence_refs:
  - type: server_fact
    source: postgres
  - type: rpc_probe
    source: mt5grpc2.mtapi.io:443 共享会话消费腿实测
---

# mtapi 日K深拉 504 阈 + 会话史缓存陈窗签名

- **坑**：工单#35162（mt5-dir-dataplane）。两层叠加：
  ①`PriceHistoryExMany NumBars=1000`（生产 klines.go 旧值）→
  mtapi nginx **504 Gateway Timeout**（2 符×500 同死、2 符×200 活——
  深度阈在 200-500 之间，非符号数问题）；批 60 符×1000 棒=6 万棒
  响应体同压超时缘。
  ②2026-11-20 全簿 23458 行 md_klines venue='mt5' 棒齐截
  **2026-04-19**（122 符 max(ts) 同值）而 recv_ts 当日新立——
  网关/终端侧 PriceHistory 史缓存按**会话生命周期**冻结：会话
  token 09:52 重建后同 RPC 立即回新鲜棒（末棒=11-22）。陈窗非
  代码病、非逐符缺史——全簿同截=会话级缓存签名。
- **律**：深拉参数先实测阈再写死——NumBars 取 ≤200（≈10 个月
  交易日≫60d 验收锚）；批量×深度乘积控制在网关超时缘内
  （klineBatch 60→30）；回棒窗末端设陈化阈（>14d=WARN 现形），
  陈窗出现先查会话 token ts——会话重建=史缓存刷新第一嫌疑径，
  不动码面先怀疑 venue 侧。
- **配套**：探针径=共享会话消费腿（SetSharedToken+CheckAlive，
  永不 ConnectEx——双 login 互撞坑）；tools/mt5dirprobe
  （SymbolList+GetQuoteMany+SymbolParamsMany 全目录过滤枚举）
  与 tools/mt5kprobe（PriceHistoryExMany 深度/新鲜度实测）同批
  落仓可复跑。
