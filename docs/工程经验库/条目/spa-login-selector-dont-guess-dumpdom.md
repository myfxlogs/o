---
schema_version: 2
id: spa-login-selector-dont-guess-dumpdom
kind: pitfall
risk: low
status: verified
keywords: [cmcweb, 无头浏览器, 登录表单, 选择器, cookie注入, localStorage, AngularJS]
scope: [cmcweb, browser, login]
summary: 浏览器臂登录面两坑合一——①cookie注入对要localStorage态的SPA不够（注入后仍登录面显形）；②登录选择器凭猜必漏（CMC loginForm=AngularJS 结构:#username type=text 非 type=email/input[submit] ng-click）——修=SG dump-dom 实证结构再写选择器;OneTrust 横幅会盖提交钮先撤
first_seen: 2026-11-16
last_seen: 2026-11-16
evidence_refs:
  - type: log
    ref: "o-cmcweb session 建失败(1/3): 登录面显形（2026-11-16 首部署）"
  - type: server_fact
    source: cex
    ref: "SG /tmp/cmc-dom.html dump-dom:loginForm AngularJS #username/#password/input[submit]"
---

# SPA 登录面：选择器勿猜，dump-dom 取证

## 坑形

cmcweb 浏览器采集中继首部署即两连错：

1. **cookie 注入未生效**：AuthManager 会话 cookie 注入浏览器后导航，
   app 仍显登录面——SPA 登录态不全在 cookie（localStorage/内存 token），
   注入只能算"加速件"不能算"登录件"；真兜底=UI 表单填充。
2. **UI 登录选择器凭猜全漏**：写了 `input[type="email"], input[name*="mail"]`
   通用族——CMC loginForm 是 AngularJS 结构：用户名=`#username type="text"`
   （既非 email 型也不含 mail 字样），提交=`input[type="submit"]` ng-click。
   选择器一个没咬住，UILogin 即败。
3. **隐形拦截面**：OneTrust cookie 同意横幅可能盖在提交钮上——
   chromedp.Click 真鼠标点击会点到横幅而非目标钮。

## 修法（实证后钦定）

- 登录面结构**先 dump-dom 取证再写选择器**：
  `chromium --headless --dump-dom <url>` 30 秒出真 DOM，比猜一百次准。
- 表单三件套实证值（2026-11-16 SG 实机）：
  `#username` / `#password` / `input[type="submit"]`。
- 横幅处置：`#onetrust-accept-btn-handler` 先点（3s 短超时 best-effort），
  不占登录预算。
- 看门狗语义补全：WS 死重载页后仍无流且登录面显形 → 整会话重建
  （重载不能替代重登——只重载会空载死循环）。

## 证据

- SG `/tmp/cmc-dom.html` dump-dom 实录（2026-11-16）；
- 部署首轮日志：`session 建失败(1/3): 登录面显形`。
