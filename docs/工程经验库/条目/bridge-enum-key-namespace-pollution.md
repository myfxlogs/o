---
id: bridge-enum-key-namespace-pollution
title: 枚举器键域与异域配置键撞名——bridge.*.symbols 误吃 okxws depth5 真键
kind: pit
date: 2026-11-27
source: 工单O-41376（o-collecto 日志 venue='depth5' 每~60s 死等 session_token 噪声；吸收在飞工单#40291）
---

# LIKE 枚举键域的命名冲突（XM 上桌二期实证）

**坑形**：`cmd/collector/main.go bridgeLegs` 用
`SELECT key FROM app_config WHERE key LIKE 'bridge.%.symbols'` 把键名
后缀当 venue 身份枚举桥腿。`bridge.depth5.symbols` 本是 okxws
books5 深度面的标的集配置（`internal/data/okxws/depth5.go` 读件），
命名落进了 bridge.* 前缀域 → 枚举器把它当 venue='depth5' 起腿 →
`sharedAttach` 等 `depth5.session_token` 永不可得（depth5 非 MT5 户）
→ 每拍一行「共享会话未立」日志噪声。无害但语义=枚举污染，且任何
未来 `bridge.<非venue>.*` 键都会同型中招。

**判别**：日志里出现「枚举出来的身份」在名册/凭据面查无此户——
先怀疑键名被 LIKE 模式跨域吃掉，不是真 venue 缺凭据。

**修法**：根修=迁名出枚举域（`bridge.depth5.symbols`→
`okx.depth5.symbols`，消费码+config-write-domain.tsv+dmwrite
ownerKeyGlobs+裸Get白名单+测试同 commit 原子换——迁移落行与码部署
同列车零空窗，无需双读兜）。**禁**在枚举器加 venue 白名单硬编码——
venue-agnostic 是设计钉（新 venue=落键即起腿零码）。

**推广**：凡 `LIKE '前缀.%'` 从配置表枚举身份的面，前缀域=保留命名空间
——异域配置键严禁落入；新键命名先问「会不会被某个 LIKE 枚举器吃成
身份」。域册（config-write-domain.tsv）是撞名的第一道可见面。
