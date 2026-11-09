---
schema_version: 2
id: heredoc-sql-unquoted-silent-swallow
kind: pitfall
risk: high
status: verified
keywords: [bash, heredoc, sql, 拼值无引号, 静默吞, gw_ledger, 账本哑写]
scope: [shell, psql, ops_event, gateway, 账本写径]
summary: heredoc 拼 SQL 值不包引号→裸标识符语法错，被 >/dev/null 2>&1 静默吞——账本写径全哑且零可见
first_seen: 2026-11-09
last_verified: 2026-11-09
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0517-升件归领口分类路由-机械可证类自销+LLM类呈CEO.md
  - type: doc
    ref: scripts/meta/gateway/common.sh:42
---

## heredoc 拼 SQL 裸值+静默吞=整条写径全哑不可见

- **实证**：`gw_ledger`（T-0469）写
  `VALUES('gateway',$(gw_qc "$1"),...)`——`gw_qc` 只做
  `sed s/'/''/g` 转义**不包引号**，拼出
  `VALUES('gateway',升件路由,T-9999,...)` 裸标识符→psql 语法错，
  且调用点 `>/dev/null 2>&1` 全吞→**o-gateway 四域全部账本行
  （升件路由/收敛 PLAN/FIRE/席权/路由）自启用起零落，只有
  gatewayd 直连 gw_q 的 quoted-literal 心跳落过一条**。
  桩测断言 `,收敛,` 恰好只匹配裸值错误格式——断言编码了 bug
  本身，八组桩测全绿掩盖了死写。
- **波及**：升件路由梯（escalate）整域哑火——未认领升件只攒不派，
  「自动递 CEO」语义在案零执行；冻结断言下 189+95 笔账面升件
  无人销账压死收敛面。
- **教训**：①heredoc/字符串拼 SQL，值必须 `'$(qc "$v")'` 包引+
  内容转义双件齐——转义函数名（qc=quote-content）不负责包引；
  ②写径 `2>&1` 全吞前至少留一行 stderr 样本（或写败计数账），
  「写败静默退」不能静默到语法级；③桩测断言 SQL 文本时按
  **正确格式**断言（`,'词',` 非 `,词,`）——断言跟着实现漂=bug
  合法化。
