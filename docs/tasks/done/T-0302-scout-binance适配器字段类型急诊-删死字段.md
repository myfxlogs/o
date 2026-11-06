# T-0302 scout binance 适配器字段类型急诊——删死字段

## 状态

**完成（2026-11-03，决策席急诊自施 typo 级＋合 @bdf7a292）**

## 生产实证（三锚全绿，05:55 UTC 实拍）

| 锚 | 实证 |
|---|---|
| binance 行落库 | `md_venue_funding` GROUP BY：binance-usdm=896（恰=真载荷 USDT 尾数 896——协议面互证）、bybit=3720、hyperliquid=940 |
| 拍败清零 | 重启后 journal grep `拍败`=0；metrics `measured_at=05:54:48` 刷新 |
| SG 镜像 boards | `jsonb_array_length(metrics->'boards')`：binance=2、bybit=2、hyperliquid=2、okx=1 |

次生发现一并处置：projector 每拍 `permission denied for table fin_venue_candidate`——
fin_0098 补 `GRANT ... TO projector_rw`（同 fin_opp_eval 权集）已入迁移并手工
授权 SG 先行（T-0301 锚③由此达成）。二进制部署坑：`ExecStart=%h/o-research/o-research`
（目录式布局，非 `~/o/` 旧径）——投件照 unit 文准。

待办（出本票面，后续票）：scout rejects Door 无 Snapshot 消费面（venue_down
运行期不可查）；`interval_h` 归真（fundingInfo 端点）；adapter 测试换实测载荷 fixture。

## 背景（生产实证，非推断）

T-0301 部署后实测：scout 两拍跑完，bybit/hyperliquid 行入 `md_venue_funding` 正常（930×2/235×2），
**binance-usdm 恒零行且无错误日志**——`metrics.boards=[]`。盒上探针复现 `http.DefaultClient`
取 `fapi/v1/premiumIndex` 全量响应 200/209KB 正常。

根因（`internal/research/scout/collect.go` collectBinance unmarshal struct）：

- `IntervalH int ` + tag `json:"interestRate"` —— 真实响应 `interestRate` 是**字符串**
  `"0.00000000"`，`json.Unmarshal` 遇 string→int 类型冲突破整次 unmarshal → adapter 返回 err →
  `venue_down` 拒 → continue → 零行。该字段**声明后从未被读**（interval 用的是 FundingIntH）。
- 次生事实（不阻锚，记入）：`fundingIntervalHours` **不在** premiumIndex 响应——FundingIntH
  恒 0 → 落表 `interval_h=0`。该列未被度量消费（settles_per_day 走 config
  `research.funding.settles_per_day`），真值源是 `fapi/v1/fundingInfo`（盒上已验 820 行带
  fundingIntervalHours）——归真属后续票，本票不扩。

## 钦定目标（急诊最小面）

1. collectBinance unmarshal struct **删除 `IntervalH int ` + `json:"interestRate"` 死字段**（一行）。
   不改其它字段、不加第二端点、不动度量。
2. `go test ./internal/research/scout/` 绿 + 真响应手工核（盒上已存 /tmp/prem.json 可对）。
3. 交叉编 arm64 → 投盒 → 重启 o-research → 一拍内 `md_venue_funding` 出 binance-usdm 行 +
   metrics.boards 非空。

## 部署后验（本票锚，全部已半验）

- [x] 拓扑桥：PG-md `fin_venue_candidate` 10 行 + measure() 产出 metrics（05:49 实拍）
- [x] projector：env 补齐 SG_PG_DSN 后 active，boot 首拍投影 venue_candidate
- [ ] binance-usdm 行入 md_venue_funding ← 本票修复锚
- [ ] SG 镜像 metrics 非空（projector 下一拍上推）

## 钉注（防再议）

- **scout rejects Door 不可见**：`NewRunner` 内建 `rej` 无任何生产面调 `Snapshot()`——
  venue_down 计数运行期不可查，本次只能靠探针反推。观察面缺口，后续票补。
- **协议面取证坑**：adapter 桩响应未带真实字段类型（测试 stub 漏 interestRate 字符串）→
  假绿。规条：adapter 测试 fixture 须用实测截获载荷，禁手写理想形。
- `fundingIntervalHours` 归真 → `fapi/v1/fundingInfo` 第二拉（公开零凭据）——后续票。
- 决策席自施：删除死字段=typo 级（collab「typo 级除外」），超面不回本票扩。
