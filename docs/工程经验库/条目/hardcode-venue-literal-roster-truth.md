---
id: hardcode-venue-literal-roster-truth
title: venue/缺省/谓词三族硬编码——名册真源与单源常量缺位=漂移实证
kind: pit
date: 2026-11-27
source: O-41383（exness disabled 实案——13+ 处 venue 字面量钉死户记账活）
---

# 硬编码三族病灶（O-41383 普查根修）

**坑形**：同一语义在多处各自手写字面量——改一处忘其余=静默漂移：

1. **venue 字面量**：`fin_venue_roster` 是注册表唯一真源（业主令
   2026-11-25），但 `'exness'`/`'"exness"'` 字面量散在名册外 ~13+ 文件
   （xvenuepair/simdrill/sizing/depenum/xshadow/ledger/xvenueswap）。
   exness 退役 disabled 后，纸腿/演习格仍钉死户=**死户记账活病灶**；
   depConfigReaders 静态双册=新 CFD 户入册零码也不入探测面。
   根修=`mt5.ActiveCfdVenue(ctx,pool)`（proto='cfd'∧state='active'
   ORDER BY venue LIMIT 1，零活户 err fail-closed）+ depenum 改名册
   proto='cfd' 全列枚举（不限 active——探测面须见 disabled 户陈会话行）。

2. **缺省双源**：`lifecycle.dir.*` 三键+`exec.tick_max_age_s` 缺省值
   在 dirpaper/dirtrig/basis/dcd_paper/notional 五读点平行手写
   （0.15/0.30/240.0/120.0）——注释自承「缺省同 dirpaper 内置值」=
   第二套自供状。根修=legset `DirStopDefFrac/DirTakeDefFrac/
   DirTTLDefH/TickMaxAgeDefS` 常量单源（SparseTickBoundDefS 先例位）。

3. **谓词平行件**：`strings.HasSuffix(inst,"-SWAP")` 13+ 处各自实现
   （含 `instID[len-5:]=="-SWAP"` 手写切片同义件）——键形判据漂移面。
   根修=`legset.IsSwapInst`/`IsUSDTSwapInst`（后者窄判不收币本位
   -USD-SWAP——语义勿混）。

**判词**：`venue=="okx"` 模板契约判/`session_token` 键拼接/`proto='cfd'`
名册读面本体=**不动产**（架构语义非 venue 枚举）——动了=超面。
**闸**：`scripts/check_hardcode.sh` 四面断言常驻化（check_all
immune 后挂位）——劝诫会失效、闸才守得住；白名单内嵌=显形决策点
（grammar.go 模板族声明面 inert 残余留册注明另票）。
