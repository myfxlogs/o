# T-0427 粗扫扩全宇宙——instType 多族并入现货/期货目录

- 状态：施工中（外派领单 2026-11-07）
- 领取：外派施工（O-Dev 径——扫面扩族钦定全，不占 DM 钱径额度；支 t/0427-scan-universe @ /opt/o-wt/t0427）
  ——施工 session `zcode-t0427` @ 2026-11-07 领单开工
- 优先级：P1——发现域结构性缺（业主令根基定位）
- 收口移交归：决策席
- 关联：T-0401（两级制本件）、T-0416（UniverseCatalog 供给件
  当前 SWAP-only）、T-0412（选符域吃 md_tick_scan——本票落地
  即现货/期货可选可提名，前端零改动）、方向单数据面（另案）

## 病灶（实机三证在案）

- `md_tick_scan` 528 行清一色 SWAP（`NOT LIKE '%-SWAP'`=0）；
- `scan.go` REST 钦定 `instType=SWAP` 单请求；
- `UniverseCatalog` 拉 `/public/instruments?instType=SWAP`——
  目录供给件同盲；
- 后果链：自选报价/价格提醒搜不到未订阅现货符
  （BTC-USDT 靠持仓腿侥幸在采，ETH-USDT 类全盲）；
  期货/现货宇宙零发现——「全宇宙」实为「全 SWAP」。

## 范围

1. **UniverseCatalog 多族**：`instType` 参数化——
   `SWAP`/`SPOT`/`FUTURES` 三族目录（同供给件同 TTL 同
   fail-closed；OPTION/MARGIN 票面不收——无消费面）；目录
   行须携 inst_type 归位（消费面后续按族取子集）；
2. **okxscan 多族扫**：每拍按族各一发 REST（60s 节律不变，
   扫面 ×3 实机约 +800 行最新行——体积纪律不变）；谱域闸
   按族各判（该族目录读败=该族本拍让位不虚构，他族照常——
   fail-closed 粒度到族）；
3. **`md_tick_scan` 族标**：inst_id 形状可辨族（-USDT/-SWAP/
   -USD-YYMMDD），但钦定显式 > 键内隐式——加 `inst_type`
   列（迁移序号接续；存量行回填 'swap'）——防族判散落
   成 inst_id 正则复制件（单位错配同族病预防）；
4. **消费面对拍**：选符域 `candSets`/symDomain 谓词零改动
   应自动见新族符（断言行在验收）；晋升径 fin_sub_request
   对现货/期货符同事务提名→collector SubRequestArm 拾取
   →okxws 订阅该族符 tick——订阅臂族兼容实证（现货符
   WS 通道若与 SWAP 不同源，本票实接不假设）；
5. **MT5 谱系不沾**：MT5 目录=broker Symbols 另径
   （venue 注册面族），本票不并。

## 验收

- 红桩：三族 tickers fixture→md_tick_scan 携 inst_type 落行；
  某族目录读败=该族零行他族不受影响（fail-closed 到族断言）；
- 实机锚（部署后验）：SG `md_tick_scan` 三族行在（SPOT/
  FUTURES 非零）+选符域 candSets 见现货符（如 ETH-USDT
  非订阅态）+现货符建提醒同事务提名晋升→md_ticks 实采
  到 tick（全径实证非断言）；
- 绿戳对口+全链。

## 结果回填

- 状态：待派
