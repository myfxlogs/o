# T-0359 IBKR 凭据读面 adapter——paper 先行

- 状态：已批（COS 例行代理 @2026-11-05——venue 引入方向业主已裁，
  paper 凭据已封 app_secrets `ibkr.paper_*` 三键；读面-only 无红线件）
- 领取：（待派）
- 级别：P2 进水口扩面——`fin_venue_candidate ibkr state='candidate'`
  卡凭据闸；凭据已落地，接读面即升 `credentialed`，CME 基差族
  （docs/venue/ibkr-carry-族枚举.md 枚举在案）才有评估前提

## 前置事实（实查在案，勿再诊）

- 凭据：paper 户 `wrtjke924`/`DUT171280`，三键已 Seal 入
  `app_secrets`（`ibkr.paper_username/paper_account/paper_password`），
  解密走 `cryptoutil.Open` 正径（readcred 同款），**明文凭据禁入
  argv/env/log/文件**；
- API 径钦定：Client Portal Gateway（docs/venue/ibkr-api-对照.md
  结论行——无头轻量、REST+WS 与 okx 读面形态最近、paper 完整）；
- 账户事实面：okxpriv 先例在案（`internal/data/okxpriv/collector.go`
  →`md_account_events` 事实行+心跳活性行），本票对同款形；
- venue 读败升件闸在案（T-0352）+条件请求盒在案（T-0353 件④
  fin_condition_gap）——复用，禁新造；
- **Futures-US 权限业主请求已提交但冷静期 ~7 天**——本票不依赖
  期货权限（balance/position/margin 读面不需要），CME 行情面
  留待权限生效后另票；
- **CME L1 订阅本票不订**（业主裁：未 armed 不付数据费——延迟
  行情评估期够用）。

## 任务（修法供裁）

1. **gateway 部署面**：OCI 部署机（ubuntu@149.118.58.98）装 JRE+
   CP Gateway，systemd user unit 常驻；首登 `localhost:5000` 走
   SSH 隧道由业主网页登录+IB Key 2FA（业主配合项落 runbook
   明文步序）；
2. **读面 adapter**：定时拍拉 REST（balance/position/margin 三面
   纸户），事实行落 `md_account_events`（src 键域按 okxpriv
   归一规），心跳活性行同款；
3. **会话死活面**：CP 会话每日到期是钦定约束——到期/读败即
   fail-loud 升件（T-0352 闸）+条件请求盒落「待业主重登」
   （owner-domain，禁自动绕过）；**重登税评估**：CP daily 2FA
   vs IBGW auto-restart（GUI 依赖代价）vs 其他——选项+代价
   落 runbook 供裁，默认 CP；
4. **候选升态**：读面实证后 `fin_venue_candidate ibkr`
   `candidate→credentialed`（迁移随件落）。

## 验收锚

1. OCI 上 gateway+adapter 常驻，paper balance/position/margin
   三面事实行实出 `md_account_events`（查询锚）；
2. 会话到期/读败实证升件+条件盒落行（合成或实发）；
3. `fin_venue_candidate ibkr state='credentialed'`（迁移锚）；
4. 凭据零明文面实证（日志/env/文件 grep 无值）；
5. 重登 runbook 落 `docs/runbooks/`（业主步序一页）；
6. check_all 绿戳对口提交 SHA。

## 红线

- **读面-only**：venue 零 mutating——禁接任何 order/trade 端点，
  禁预留下单径代码（哪怕 dead code）；
- paper 户 only——live 凭据切换另票另闸（业主令前置）；
- 凭据只自 `app_secrets` 密文解——禁 env 明文/禁配置文件明写/
  禁日志回显；
- 不动既有 venue 读面/阈/白名单；
- CME L1 订阅闸不动（订阅=业主动作，armed 提案前置才议）。
