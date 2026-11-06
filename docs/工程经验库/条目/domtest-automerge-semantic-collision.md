---
schema_version: 2
id: domtest-automerge-semantic-collision
kind: pitfall
risk: medium
status: verified
keywords: [domtest, merge, auto-merge, 语义冲突, const重声明, 断言漂移, 平行票]
scope: [web, domtest, merge, 裁决]
summary: 平行票同改 domtest/组件件 git auto-merge 文本净并仍双炸——const 重声明语法炸+被撤 UI 的断言漂移红；合并树必亲跑测不信文本净并
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: []
evidence_refs:
  - type: commit
    ref: "06ab66a4"
  - type: doc
    ref: web/scripts/quoteboard-domtest.mjs
---

## 事实/症状

T-0411（玻璃浮层升 datalist）与 T-0412（pendingSyms「·晋升中」标）平行
同改 `QuoteBoard.tsx`/`QuoteWatch.tsx`/`quoteboard-domtest.mjs`。
git auto-merge 报三处文本冲突，解完后 domtest 件**零冲突标记**却仍炸
两面：

1. `const phtml` 双侧各自新增同名校验变量→合并树 Identifier 重声明
   SyntaxError（文本净并语法炸——合并器不识 JS 作用域）；
2. T-0412 断言「表单候选 label 携晋升标」按 datalist 世界观写——
   datalist 已被 T-0411 升浮层，断言漂移必红（语义炸——断言锚的 UI
   形态在另一票里已换）。

同型风险面：组件件 props/状态语义平行增补（pending × 浮层态）——
文本不撞语义撞。

## 根因

文本级三方合并不识：①同文件双侧新增同名声明；②断言锚指向的 UI
结构被另一侧撤换。零冲突标记 ≠ 语义无损。

## 对策（T-0418 裁决班实证径）

- 合并票涉同文件平行改：解完文本冲突后**必亲跑该面测谱**（npm test/
  tsc），不信 auto-merge 零冲突收尾；
- 断言漂移修法=断言随合并后 UI 形态改写锚位（⑨表单锚自 datalist
  label 迁浮层候选钮），保原断言语义不降级；
- 负向断言按 `[^<]*` 界内断（防跨界误咬邻钮件），禁 `[\s\S]{0,N}`
  宽窗。
