[角色:施工] T-0313——venue-scout 六候选 adapter 接入+canon 方言扩

## 任务单
/opt/o/docs/tasks/T-0313-venue-scout漏斗补齐-六候选adapter与canon方言扩.md
工作树 /opt/o-wt/t0313 自切分支 t/0313-scout-adapters。

## 范围与文件清单（清单外不动）

1. internal/research/scout/collect.go：adapters 册加 6 件——
   bitget / gate / kucoin-futures / mexc / dydx-v4 / deribit，
   签名 func(ctx, f Fetcher) ([]row, error) 照 collectBinance/
   collectBybit/collectHyperliquid 三先例
2. 端点起点=fin_venue_candidate.endpoints 种子（md_1082/fin_0098 可查），
   **字段名以实测响应为准**——先 curl 实拉看 JSON 再写映射（协议面取证
   纪律），种子端点不够用允许换并记 evidence
3. interval_h：端点有取端点值，无则 nil（禁虚构）；kucoin 混合档
   （1h/4h/8h 并存）按 per-inst 实取；deribit 连续结算若端点无小时档
   如实 nil（settles 会走 config 回退）
4. canon()（series.go）扩方言，每条附用例：
   - 尾缀表加 "PERPETUAL"（deribit BTC-PERPETUAL→BTC）
   - kucoin 永约 M 尾（XBTUSDTM→剥 M→XBT）+XBT→BTC 别名
   - 剥序注意：M 尾在 quote 剥除之后处理
5. 升 measuring：新 md 迁移（md_1085）UPDATE 六 venue
   state='measuring'——不改 fin_0098/md_1082 种子原件
6. scout_test.go 每 adapter 一桩测（照三先例 url-switch stub）

## 禁线
- 零凭据原则不变——全公开端点，禁加 header/auth
- fetch_err/parse_err/venue_down 计数照现有 rejects 册，单 venue 挂不拖全局
- 不动 metrics.go 判据口径、不动 fin_venue_candidate 表结构
- 超清单→停手回报

## 验收硬迹（缺一项不予审）
- check_migrations/check-env-reads/go build ./... /scout 包 go test 全过
- OCI arm64 部署后 md_venue_funding 六 venue 实落行：贴
  select venue,count(*),max(ts) group by 1
- 下一度量拍 fin_venue_candidate.metrics 六家 boards 非空实迹
- canon 新方言用例测试实录

## 自审记录
| 编号 | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 种子端点个别为历史/单符号径（deribit history、kucoin 单 symbol） | 端点完备性缺口 | 明示允许换端点+evidence 记——不钦定死 URL |
| 2 | canon 扩有剥序耦合（M 尾 vs quote 剥序） | 顺序陷阱 | 明示剥序约束+测试用例硬门 |
| 3 | deribit 连续结算模型与 8h 常态不同 | 口径坑 | 允许 interval nil 走回退——如实优于虚构 |
| 4 | measuring 升态放 md 链新迁移 | 部署锚 | 钦定 md_1085 新件——正本侧升，投影自动 |
| 5 | 验收=落行+boards 非空双实迹 | 硬门 | 零实迹不予审 |
