# IBKR 凭据路书（T-0339 第一产出——业主照单可点）

核证日 2026-11-04；每步附出处。本件只做「怎么开」清单——
**是否开通/何时开通归业主裁**；执行面零动作。

## 0. 前置事实

- 业主已持 IBKR 户 → **唯一跳过注册/KYC 瓶颈的候选 venue**。
- CP Gateway 官方明文要求「**opened and funded** 账户」才可连
  （出处：https://www.interactivebrokers.com/campus/trading-lessons/launching-and-authenticating-the-gateway/）
  ——户内有资即过，无额外资金门槛（无 API 开通费/无 $0 月最低佣金坑）。
- **API 无申请审批流**（CP/TWS 两径）——凭据=现有账密+2FA，
  开通动作=下载 gateway+登录，非向 IBKR 递件等批。

## 1. 推荐径：Client Portal Gateway（读面首选）逐步清单

| # | 动作 | 位置/命令 | 出处 |
|---|------|----------|------|
| 1 | 拿 paper 户凭据（开发验证零风险径） | Client Portal → 右上角人头 → Settings → Account Configuration → **Paper Trading Account** → 取 Paper Username/重置 Paper 密码 | https://www.interactivebrokers.com/docs/web-api/authentication/paper |
| 2 | 装 Java JRE（gateway 依赖） | `java -version` 验；缺则按官方下载链装 | 同 gateway 教程页 |
| 3 | 下载 Client Portal Gateway zip | cpwebapi 文档站 Quickstart → API Gateway 下载面板 | https://interactivebrokers.github.io/cpwebapi/ |
| 4 | 拉起 gateway | 解压后 `bin/run.sh root/conf.yaml`（窗口须常开） | 同上 |
| 5 | 浏览器登录认证 | 开 `https://localhost:5000` → 账密登录 → **IBKR Mobile（IB Key）2FA** 确认 | 同上 |
| 6 | 确认会话活性 | gateway 页显 authenticated=true 即可调 REST/WS | 同上 |
| 7 | **每日重登**：CP 会话每日到期，需重新网页认证一次（官方限，无绕过——机构 OAuth 除外） | 同 #5 | tws-api/initial_setup 及 Lumibot 文档佐证 |

## 2. 补充径：TWS API（行情深面时才需要）

| # | 动作 | 位置 |
|---|------|------|
| 1 | 装 TWS 或 IB Gateway（IBGW 省资源 ~40%，推荐） | 官网下载「offline」版 |
| 2 | GUI 登录账密（headless 官方不支持——需图形环境） | 登录窗 |
| 3 | 开 API socket | TWS → Global Configuration → API → Settings → Enable ActiveX and Socket Clients，默认端口 live 7496 / paper 7497 |
| 4 | 开 auto-restart（974+） | 同 Settings——可跨日免重登跑 Sun→Sun |
| | 出处 | https://interactivebrokers.github.io/tws-api/initial_setup.html |

**运维坑**：TWS/IBGW 设计为每日重启；同 username 登 Client Portal 会
挤掉 API 会话——正式跑需**加开副 username**（Settings → Users 管理），
行情订阅按 username 计费。

## 3. 权限与订阅（业主在 Client Portal 内勾选）

| 项 | 位置 | 费用 | 出处 |
|---|------|------|------|
| 交易权限：Futures（United States） | Settings → Trading Permissions | $0（勾选即得，部分辖区要求经验问卷） | pricing/margin-requirements 页 |
| 行情：CME Real-Time L1（非专业） | Settings → Market Data Subscriptions | **$1.55/月/交易所**；四所 bundle $4.65/月；L2 深度 $12.10/月 | stockmarketstack 汇总 + IBKR market-data-pricing 页（CME 官方名录价同） |
| 备用档：US Securities Snapshot & Futures Value Bundle | 同上 | $10 基础费**佣金达标即免**+ $0.01/snapshot | IBKR market-data-pricing 页 |
| 2FA | IBKR Mobile IB Key | $0 | gateway 认证流程 |

**paper 户零费完成开发验证**：paper username 共享 live 行情订阅档
（订阅开在 live 户上，paper 会话可见），不另计费。

## 4. 不推荐径（留档防再走）

- **OAuth 直连 Web API**：机构/第三方 onboarding 径，零售个人户不适用。
- **FIX CTCI**：$10,000 equity 门槛 + $1,500/月最低佣金（首节）——
  读面阶段零价值（出处：https://www.interactivebrokers.com/docs/fix/requirements.md）。

## 5. 业主待办汇总（若批准开通）

1. （可选）Settings → Paper Trading Account 取 paper 凭据；
2. Settings → Trading Permissions 勾 Futures-US；
3. Settings → Market Data Subscriptions 订 CME L1（$1.55/月）；
4. 部署机上装 JRE + 下载 gateway + `bin/run.sh` + localhost:5000 登录
   + IB Key 确认 → 凭据事实落地（env 件由业主/凭据径处理，不入本档）。
