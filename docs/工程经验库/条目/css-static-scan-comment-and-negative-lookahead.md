---
schema_version: 2
id: css-static-scan-comment-and-negative-lookahead
kind: pitfall
risk: low
status: verified
keywords: [CSS, 静态扫描, domtest, 正则, backdrop-filter, 注释, 负向先行, 回溯, 机检]
scope: [web, domtest, css-lint, 机检闸]
summary: CSS 规则级静态扫描两坑——注释文本粘进选择器段误判+`X:\s*(?!none)` 因 \s* 回溯负向先行失效
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: web/scripts/glass-skin-domtest.mjs
  - type: doc
    ref: web/src/react/styles/12-nox-shell.css
---

## 事实/症状

T-0411 玻璃皮肤机检（backdrop-filter 白名单扫描）两处假红：

1. 规则级正则 `(^|})([^@{}]+){...}` 把规则体前的 `/* ... */` 注释整段
   粘进「选择器」捕获——注释里提到 backdrop-filter 字样的规则
   （.card/.card-solid）被判违规；同理规则体内注释含该词也误判。
2. 「显式撤玻璃不算违规」判定写 `backdrop-filter:\s*(?!none)` 想放过
   `backdrop-filter: none`——依然命中：`\s*` 贪婪吃掉空格后 lookahead
   失败会**回溯** `\s*` 到零宽位，lookahead 处看到的是 `" none"` 前的
   空格字符（非 none）→ 负向先行通过 → 误匹配。

## 根因或策略解释

- 正则引擎无注释语义——扫描器必须先机械剥注释（`s.replace(/\/\*[\s\S]*?\*\//g, "")`）
  再做规则级拆解；CSS 文件即文档，注释里出现被扫词是常态非例外。
- JS 正则量词可回溯：负向先行想钉「冒号后（可有空白）必须不是 none」，
  正确形是**把空白吃进 lookahead**：`backdrop-filter:(?!\s*none)`——
  冒号后不再有可回溯量词，`\s*none` 整体在 lookahead 内试匹配。

## 处置

CSS/源码静态扫描器三条纪律：①先剥注释再拆规则；②「值≠X」负向先行
写成 `key:(?!\s*X)`（空白进 lookahead，不留可回溯量词在 lookahead 外）；
③断言红时打印命中段全文（offenders 带选择器+文件）一眼定位假红源。

## 回改点

若后续加第二处负向先行式机检（如禁某属性值），照 `(?!\s*X)` 形写；
新 CSS 扫描 domtest 复制本条三纪律。
