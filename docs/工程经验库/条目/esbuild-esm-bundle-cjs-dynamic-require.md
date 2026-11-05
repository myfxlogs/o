---
schema_version: 2
id: esbuild-esm-bundle-cjs-dynamic-require
kind: pitfall
risk: low
status: verified
keywords: [esbuild, bundle, ESM, react-dom/server, Dynamic require, createRequire, banner, node 内建件]
scope: [web, 前端构建, 测试断言, esbuild]
summary: esbuild format=esm 全捆扎含 CJS 依赖（react-dom/server node 变体）——运行时 Dynamic require of "util" 断；须挂 createRequire banner 桥或落件于可解析 node_modules 的目录
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: []
evidence_refs:
  - type: commit
    ref: t/0349-survival-card-domain
---

## esbuild ESM 捆扎 CJS 依赖：动态 require 内建件断

- **实证**：T-0349 存活卡渲染面断言（`web/scripts/metaseats-domtest.mjs`）用 esbuild API
  `format:"esm", platform:"node"` 全捆扎 metaseats.tsx+react-dom/server 落 /tmp 再 import——
  react-dom-server.node.development.js 内部 `require("util")`（CJS 动态 require）在 ESM
  产物里成 esbuild 注的 `throw Error('Dynamic require of "util" is not supported')` 桩，运行即断。
- **害**：断言面全瘫；且坑在「产物落哪都能跑」的错觉——首次跑才显形。
- **教训**：①ESM 产物含 CJS 依赖且该依赖动态 require node 内建件时，挂
  `banner: { js: 'import { createRequire } from "node:module"; const require = createRequire(import.meta.url);' }`
  桥（内建件跨路径必解析，三方包走 bundle 内化）；②外置依赖（`packages:"external"`）时
  产物落点必须在 node_modules 解析可达处（/tmp 落件解析不到 web/node_modules——
  `ERR_MODULE_NOT_FOUND: react-dom` 同拍实踩姊妹面）；③renderToStaticMarkup 无 DOM 容器
  也可跑=repo 无 jsdom 时最小 DOM 断言可行径。
- **收口实证**：banner 桥后 `node scripts/metaseats-domtest.mjs` 12/12 全绿 exit 0（T-0349 收口拍）。
