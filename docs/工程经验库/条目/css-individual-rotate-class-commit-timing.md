---
schema_version: 2
id: css-individual-rotate-class-commit-timing
kind: pitfall
risk: low
status: verified
keywords: [CSS, rotate, individual-transform, getBoundingClientRect, pointer-drag, React, setPointerCapture, FLIP]
scope: [web, react, pointer-events, 拖拽]
summary: 类挂 rotate 个体变换同帧未提交——拖起瞬间 rect 测量缺旋转 bbox 膨胀致卡在指针下微跳；内联先转再测
first_seen: 2026-11-06
last_verified: 2026-11-06
skills: []
evidence_refs:
  - type: doc
    ref: web/src/react/components/usedeckdrag.ts
---

## 事实/症状

T-0411 模块拖拽：pointermove 过阈时 `setDragId(id)`（React 重渲挂
`.is-dragging` 类，类内 `rotate: -3deg`）同帧内立刻
`getBoundingClientRect()` 测抓取偏移 gx/gy——此时类样式未提交，rect 无
旋转；下一帧 rotate 生效，bbox 膨胀（340×250 卡 3°≈+8px），卡在指针下
跳位数 px。

## 根因或策略解释

React setState 是异步提交；同一事件处理器内 DOM 类挂上前的同步测量拿
不到类样式结果。个体变换属性（rotate/scale/translate）同样参与
getBoundingClientRect 的 bbox 计算——旋转盒 bbox 按外接矩形膨胀，测基点
必须含旋转态。

## 处置

「先变形，后测量」：过阈分支里 `cell.style.rotate = "-3deg"` 内联先落
（与类同值双保险），再测 rect 算 gx/gy；松手 `cell.style.rotate = ""`
内联清（is-settling 过渡承接回弹）。凡「拖起变形+实测锚位」组合一律
此序。

## 回改点

后续 FLIP/拖拽件若用类挂个体变换，抓取测量前先内联同值变形。
