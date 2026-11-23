---
id: bootstrap-gen-clobber
title: 降级径 rsync 无 --ignore-existing——陈快照按名盖 buf 鲜产（三烧列车）
kind: pit
date: 2026-11-23
source: 工单O-35491③/源票#35526（Symbols 覆没实证，烧过三列车 #35333/#35357/#35377）
---

# bootstrap 降级径盖鲜产

**坑形**：`scripts/bootstrap_worktree.sh` 降级径
`rsync -a "$SRC/internal/gen/" "$WT/internal/gen/"` 无
`--ignore-existing`——任一 gen 目录缺件触发降级时，部署树**陈快照**
按文件名把 worktree 里 buf 正径刚产出的**鲜产**整面覆写。症状=
合并点编译绿但行为陈旧（Symbols 面覆没烧过三列车才抓到——
gen 件 gitignored 不入库 diff 盲区，绿戳验的是编译非新鲜度）。

**连带坑**：unit-PATH 无 `~/go/bin`——buf 插件（protoc-gen-*）与
buf 本体（go install 产物）隐依赖，systemd/cron 环境跑 bootstrap
正径静默缺插件→降级径接手→上坑连发。修法=件首
`export PATH="$(go env GOPATH)/bin:$PATH"`。

**判别**：worktree `internal/gen` 件 mtime 新于 proto commit 时间
但内容=部署树旧版——`diff -q` 抽查 gen 件 vs `/opt/o` 同件可证
（注意：`-a` 保时间戳，mtime 不可靠，看内容不看时间）。

**修法**：降级径 `rsync -a --ignore-existing`——只补缺不盖鲜；
web gen 整面 `--delete` 快照径不动（无本产可保）。

**验收仿真**：`GOPATH=/tmp/fake PATH=/tmp/nobuf:...` 造 buf 必败
环境（注意件首 PATH 前置会把 `~/go/bin/buf` 真身顶回——仿真须
连 GOPATH 一起换，否则假 buf 被自家修复件架空）；断言=wt 内
自植 marker 文件存活+缺面补齐。

**推广**：「快照降级补缺」语义必须显式声明覆盖方向——`-a` 默认
按名盖写，补缺与保鲜是两件事，rsync 不会替你猜。
