# IBKR API 三径对照（T-0339 产出①——能力面枚举）

公开文档核证 2026-11-04。IBKR 官方 API 总览：https://www.interactivebrokers.com/en/trading/ib-api.php

## 对照表

| 维度 | TWS API（socket） | Client Portal / Web API（REST+WS） | FIX CTCI |
|---|---|---|---|
| 形态 | TCP socket 客户端连本机 TWS / IB Gateway 进程 | HTTPS REST + WebSocket | 行业标准 FIX 会话（仅订单路由） |
| 前置进程 | **必须**跑 TWS 或 IB Gateway，GUI 登录（headless 官方不支持） | 零售=本机 CP Gateway（Java，`bin/run.sh root/conf.yaml`，浏览器登 https://localhost:5000）；机构=OAuth 2.0 直连 | 无本地进程；VPN/extranet/leased line/cross-connect 接入 |
| 鉴权 | TWS/IBGW 登录用户名+密码（同户多 username 可并行） | CP Gateway=网页账密（每日会话重登）；OAuth=机构申请 | FIX session 由 IB FIX Engineering 开通，QA 认证测试后投产 |
| 能力面 | TWS 全功能集（下单/账户/行情/历史/扫描器），行情/账户数据最全 | 交易/监控/账户管理/开户入金/报表；覆盖面≈Client Portal 网页 | **仅下单路由**——行情/账户数据须另开 TWS API username |
| 费用门槛 | $0 月最低佣金（internet） | $0 月最低佣金（internet）；extranet 等专径 $500/月最低佣金 | **$1,500/月最低佣金首节，$1,000/节后续；申请前 $10,000 equity 门槛** |
| paper 支持 | ✅（paper username，共享 live 行情订阅） | ✅（官方建议先在 paper 开发验证） | ❌（paper 不兼容 FIX；测试走 IB QA 系统+认证测试） |
| 速率面 | 入站消息默认 ~50 msg/s；行情行数按订阅档位 | 各端点 pacing 限制（快照/历史有节流），无单一公布总限 | 机构级直通 |
| 客户端 | GPL 开源 SDK：C++/C#/Java/Python（v9.72+ 配 TWS ≥952） | 无 SDK 依赖，纯 HTTP/JSON | 自实现 FIX 引擎或 vendor |
| 出处 | tws-api initial_setup：https://interactivebrokers.github.io/tws-api/initial_setup.html | cpwebapi：https://interactivebrokers.github.io/cpwebapi/；web-api auth：https://www.interactivebrokers.com/docs/web-api/authentication/cpgw/installation-authentication.md | requirements：https://www.interactivebrokers.com/docs/fix/requirements.md；required minimums：https://www.interactivebrokers.com/en/accounts/required-minimums.php |

## O 读面优先结论

- **首选 Client Portal / Web API（CP Gateway 模式）**：无头轻量、REST+WS 与现有 okx/mtapi 读面形态最接近、零月最低佣金、paper 完整。
- **TWS API 为补充**：行情/历史深面（扫描器、深度、组合）超过 CP 面时用，代价=常驻 TWS/IBGW 进程+GUI 登录运维面。
- **FIX 不在本阶段考虑**：$1,500/月最低佣金+仅下单面——读面研究阶段零价值，除非日后执行通量到机构级。
