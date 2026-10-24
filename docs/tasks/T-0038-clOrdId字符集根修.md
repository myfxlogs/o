# T-0038 clOrdId 字符集根修——连字符非法（真单实勘 51000）

[角色:决策] 立 2026-10-24 | 优先级：**阻断级（真单实测炸出）**

## 病灶（真单实证 exec_id=1）
- `flow.go` clOrdID=`ph-<exec>-<leg>-<nano>` 含连字符——**OKX clOrdId=大小写英数 only**（禁特殊符）
- 实跑回放：Place 拒（order 未建）→Resolve 同串亦拒 `51000 Parameter clOrdId error`→ambiguous fail-closed（钱零损——fail-closed 兜住实锤）
- NOX 实证口径：纯英数（"abc123" 样例+28 截断）

## 修法
- 格式改 `ph<exec>x<leg>x<nano>`（字母分隔位合法）或 `ph<exec><leg><nano>`——≤32 英数
- 测试锚：clOrdId 正则 `^[A-Za-z0-9]{1,32}$` 断言
- 经验库条目：venue 方言字符集账（clOrdId 禁特殊符——OKX 实拍 51000）

## 验收锚
- 生成串过 `^[A-Za-z0-9]{1,32}$`；flow 全测绿
