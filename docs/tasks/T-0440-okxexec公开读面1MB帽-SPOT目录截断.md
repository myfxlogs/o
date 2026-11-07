# T-0440 okxexec 公开读面 1MB 帽——SPOT 目录截断族级让位

- 状态：施工中
- 归口：外派外审径（工程面——行情读径容量，不碰钱径判词）
- 领取：claude-t0440 @ 2026-11-07（业主手动外派，基点 c33b61c1=origin/master+立票件）

## 实证链（2026-11-07 实拍）

- 前端「自选报价条编辑」搜 XRP 无现货候选——候选域=md_ticks 活符∪md_tick_scan 目录符；
- md_tick_scan 实拍：`inst_type` 分布 swap=528 futures=6 **spot=0**；
- OCI collector 日志每拍：`[okxscan] 粗扫族让位: SPOT 族: 宇宙目录读败——fail-closed 不虚构谱域: resp parse: unexpected end of JSON input`；
- OKX 实测：`/public/instruments?instType=SPOT`=**1,306,838B** vs SWAP=582,841B；
- 帽位：`internal/trading/okxexec/client.go:122` `io.ReadAll(io.LimitReader(resp.Body, 1<<20))`——
  公开读面 1MB 帽，SPOT 目录必截断→族级 fail-closed 每拍让位（SWAP/FUTURES 不触发）；
- 传染面：同一 PublicGet 还喂 `scanFam` 的 `/market/tickers?instType=SPOT`（ticker 响应更大）——
  即使目录闸过了 tickers 也会截；一处帽修两径同愈。

## 钦定活

① `okxexec/client.go` 公开读帽升到安全水位——`1<<23`（8MB）起步；帽保留（防失控体）
   不写死无限；注释钉「OKX instruments SPOT 2026-11 已 1.3MB 增长中」；
② 截断语义补强（可选同件）：`io.ReadAll(io.LimitReader(r, N))` 截断静默——
   读满帽仍 parse 败才报错，建议在帽处改 `LimitReader(N+1)`+len>N 显式「响应超帽」错，
   让下次同类病不再以 JSON 语法错伪装；
③ 验收三断言：check_all 绿戳对口 + OCI collecto 日志 `粗扫拍 spot=N>0` 实证 +
   `md_tick_scan` spot>0 且含 `XRP-USDT`；前端自选搜索 XRP 出现现货候选为终验。

## 顺手账（写票面回填）

- `internal/platform/notify/calendar.go:115` 同款 1MB 帽——经济日历响应远小暂不炸，
  不扩面；只在坑条目记「LimitReader 截断静默」模式。

## 结果回填（claude-t0440 @ 2026-11-07）

### 实现摘要

- **①帽升 8MB+③b 共享读件**：新件 `internal/platform/httpread`（PublicCap=1<<23+ErrOverCap+Body(r,cap)——LimitReader(cap+1) 多读一字节探测截断，len>cap 即显式「响应超帽」错）；PublicCap 定义处注释钉实证（SPOT instruments 2026-11-07=1,306,838B≈1.31MB 增长中）；`okxexec/client.go` doOnce 消费共享件（298 行棘轮内），一处帽修两径同愈。
- **③a 族谱水位闸通用件**：新件 `internal/platform/scanwm`（selfprobe/venueFailGate 同族公共模子第三形——按族分域对差判）：期望集=注册面派生全参数注入（闸体零 venue 词）；实落水位=面自持读缝（okx 面=md_tick_scan 逐族 max(recv_ts)）；对差（族零鲜落两形态：零行史首见锚起计持续/陈水水位龄即持续实证）≥10min→ops_event 升件「谱域缺行」（scope=<venue>:<族>，1/族/日 NOT EXISTS 原子去重）；注册面缺席=闸自身 fail-closed 升「谱域不可判」（1/域/日）。
- **okxws 接线**：scanFams 具名 famReg+入 ScanPoller.fams 实例字段（注册面正本不变）；NewScanPoller 自动挂线水位闸（pool 缺=守位 nil）——**collector main.go 零改动**，新族注册即自动入闸域；ScanOnce 拍末 Beat（观测面零侵入：水位读败/落行败 WARN 不拖扫径）。
- **判类册/白名单**：precedent_classes.tsv 追记谱域缺行/谱域不可判两行；check_import_direction.sh 追记边×2（okxexec→httpread、okxws→scanwm）。
- **KB 条目×2+索引重建**：limitreader-truncation-silent-json-masquerade（pitfall/高危——含残留静默位在案：calendar.go:115、research/ctx:113、research/scout:37）+venue-onboard-must-verify-family-data-lands（method——接市逐族验落表）。

### 测试证据

- `internal/platform/httpread/httpread_test.go`：帽内原样/恰帽边界不误报/超帽 ErrOverCap/帽<1 拒装/读败上抛；
- `internal/trading/okxexec/client_cap_test.go`：PublicGet 1.5MB（>旧 1MB 帽）解析过全量+8.5MB 显式 ErrOverCap 非 JSON 错——病灶回归锚+新帽断言；
- `internal/platform/scanwm/scanwm_test.go`：族=0 首见锚起→跨阈升件行落（验收④锚）/陈水即时达阈/鲜落复常清零滞回/不可判 fail-closed/nil 守位；
- `internal/data/okxws/scanwm_test.go`（真实 DB 径）：闸自动挂线+期望集=注册面派生断言+SPOT 族 recv_ts 回拨 25min→升件行落 ops_event（谱域缺行 scope=okx:spot）+复拍 NOT EXISTS 去重实证（testdb_lease 租 5435 跑 PASS）。

### 验收锚对账

- check_all 绿戳对口：**PASS**——`check-green-99a9dde4f8353b0baf8df2f5bf5a58e0c8f68ccc`（mode=scoped；此前 full 档含 DB 测试亦绿）；check-lines 棘轮（client.go 298≤300）绿。
- OCI collecto 日志 `粗扫拍 spot=N>0` 实证：**待部署窗**——生效前提=merge 后发 OCI collecto（外审 merge 时挂部署窗令），施工面无法亲证。
- `md_tick_scan` spot>0 且含 XRP-USDT：**待部署窗**——同上。
- 水位闸桩测（族=0 升件行落）：**PASS**——scanwm 桩测+okxws DB 径升件行/去重实证（见测试证据）。
- 前端自选搜 XRP 现货候选（终验）：**待部署窗**——归外审/业主终验。

### 偏离说明

- 水位闸落位 `internal/platform/scanwm`（票面未钉包位——按 selfprobe 公共模子先例归 platform；md_tick_scan 读缝与注册面派生留 okxws 装配面=模子只管判/去重/升件）；
- 对差实落判据取 recv_ts 水位（零行史+陈水两形态），强于纯 presence（族复停落亦能抓）——票面「对差=0 行」字面实现+水位强化，如实报备；
- 部分让位观测补强未扩（「粗扫族让位」日志行维持原形）——升件径已由水位闸覆盖。

### 涉及面

internal/platform/httpread/**（新）、internal/platform/scanwm/**（新）、internal/trading/okxexec/client.go+client_cap_test.go（新）、internal/data/okxws/scan.go+scanwm_test.go（新）、scripts/check_import_direction.sh、scripts/meta/precedent_classes.tsv、docs/工程经验库/条目/×2+索引、docs/tasks/T-0440（本票）、skills/索引.md（kbgen 机生）。

### 范围外确认

notify/calendar.go 未动（票面钉死）；research/ctx、research/scout 的 8MB LimitReader 未动（KB 条目在案留痕）；collector/core main 装配零改动；未碰部署树 /opt/o。

[施工收口:T-0440] @99a9dde4
