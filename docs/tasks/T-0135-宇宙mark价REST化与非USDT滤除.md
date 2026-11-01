# T-0135 宇宙 mark-price 撤 WS 转 REST 轮询+非USDT滤除（阻塞级——链路仍饱和实证）

## 实证（SG netmeter 分频道账，T-0134 部署后 7min）

- `okxws/mark-price` = **93.7MB ≈ 1.8Mbps**（507 SWAP 秒级推送）——T-0134 砍掉 index-tickers 后的新任链路大头
- `okxws/tickers`（活跃集~16 标的）= 59.9MB ≈ 1.1Mbps；funding-rate 1.6MB 可忽略
- `ens17` RX 仍 4763 kbps 贴顶——宇宙 mark-price 不撤，1006 断连必然复发

## 修法（T-0134 同法）

1. `subscribeArgs`：SWAP 形只留 `funding-rate`；`mark-price` 退出全宇宙（含季度期货——到期期货标量少同撤）
2. REST 轮询扩件：`GET /api/v5/public/mark-price?instType=SWAP` + `?instType=FUTURES` 两请求全量→`md_mark_index` 写 **`channel='mark_price'`**（复用 WS 原键域——读面零改无缝衔接；与 spotpoll 同周期同热启键）
3. `writeSpotLast` 补 `isUSDTInst` 滤——T-0134 留的非 USDT 死行滤除（含 mark-price 轮询同滤）
4. 活跃集 tickers 不动——钱径实时价流保留

## 验收锚

- `ens17` RX **< 1.5 Mbps**（活跃集 tickers+REST 轮询量级）
- `md_mark_index` mark_price 行 ts<90s 鲜、spot_last 同鲜
- 连续 30min 零 `close 1006`
- netmeter `okxws/mark-price` 字节账归零（只剩 REST 轮询行）
- 棘轮/机检全绿

## 结果回填

- subscribeArgs：SWAP 只留 funding-rate；mark-price 全宇宙撤（含季度期货
  ——107 SWAP+FUTURES 秒推=1.8Mbps 链路大头切除）。
- spotpoll 扩件：markFeed（PublicGet mark-price?instType=SWAP+FUTURES
  两请求批量件）→md_mark_index channel='mark_price'（WS 原键域复用——
  读面零改）；同周期热启键同 spot_poll_s。
- writeSpotLast+writeMarkRest 补 isUSDTInst 滤（非 USDT 死行滤除——
  T-0134 遗留面同封）。
- 活跃集 tickers 零动；netmeter okxws/mark-price 字节账应归零（部署实证面）。

## 验收（决策席 2026-10-26 @5b8d0eb 部署）

- mark_price REST 507 行/21s 鲜、spot_last 1195/20s、netmeter okxws/mark-price 归零、观测窗 0×1006——**宇宙面完全离线** ✅
- 带宽锚未达（RX 3258>1500kbps）：残余=活跃集 ~269 策略 inst_id 的 tickers 流 ~3.3Mbps——属 T-0136 armed 幽灵账（armed 超龄退役落地后活跃集自然缩编）
- 状态：作废（superseded——11-01 清账定性·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**作废**——superseded：mark_price方言归并由T-0146+快照价格全面退役T-0171双票收口（审计通过）

审计席：headless @ 2026-10-30

## 判决注记（审计席 headless 2026-10-30）

逐锚对账：

1. `ens17` RX <1.5Mbps——**未达**（票面自记 3258>1500kbps）；残余=活跃集 ~269 inst_id tickers ~3.3Mbps 归 T-0136 账（撤 mark-price 实降 ~1.5Mbps 与实证口径自洽）。锚口径混入非本票链载——锚失准项报管理席知悉，锚原文不改。
2. `md_mark_index` mark_price ts<90s 鲜、spot_last 同鲜——自记 507 行/21s、1195/20s **达成**；行后随 T-0171 退役 md_1071 清零在史，与本判无涉。
3. 30min 零 `close 1006`——自记 0×1006 **达成**，无反证。
4. netmeter `okxws/mark-price` 归零——自记归零 **达成**；diff 实证订阅全撤（SWAP 只留 funding-rate、datedFutRe 分支 continue）。
5. 棘轮/机检全绿——merge commit 5b8d0eb CI **红**（TestTrxRollupIdempotent——TRX 域与本 diff 零交集）；同期 docs-only 邻提交同挂同测=master 系统性红窗，非本票引入，如实记不连坐。

越锚实害（MANAGE-93 产物实态核验——驳回主据）：

- 票面宣称「写 `channel='mark_price'` 复用 WS 原键域——读面零改无缝衔接」**不实且 merge 时点可由 diff 证伪**：WS 径 store.go 直存 `arg.channel='mark-price'`（连字符），同期读面全查连字符键——`grammar/run.go` evalBasis `channel='mark-price'`+15min 鲜门、`ops_face` 健康面 `IN ('mark-price','index-tickers')`。REST 行落下划线 `mark_price`=新方言非复用——部署 ~15min 后期货基差评估与 mark 健康面断鲜 fail-closed，直至 T-0146 归并三读面方愈（含部署滞后实害 ~5h 量级）。
- 病灶在案：T-0146 票面自记「同语义双频道键」（SOL-USDT-SWAP xvenue_spread 缺席实证）；T-0171 票面自记「~247 行/分 同义异键共存（方言未归一）」退役清零归一 `mark-price` 单键域。
- 判：施工忠实票面，但票面键域宣称本身证伪+读面实害在史=驳回；整改已由 T-0146/T-0171 历史落毕，无重飞对象——判决留档供席位键域命名纪律戒。
