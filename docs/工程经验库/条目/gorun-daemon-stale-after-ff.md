---
id: gorun-daemon-stale-after-ff
title: go run 常驻件 ff 后续跑旧码——编译时点≠部署时点，哑尸窗内行为与新码背离
kind: pit
date: 2026-11-24
source: 工单O-35463（o-wakebus 11-22 编译旧码实证=lag窗漏读踹发丢死）
---

# go run 常驻件哑尸（ff 落地≠进程重载）

**坑形**：`deploy/*.service` 里 `ExecStart=go run ./<pkg>` 户的二进制
编译时点=进程启动时点——ff 把新码铺进 /opt/o 工作树后，旧进程照旧
跑旧码直到下次重启。哑尸窗内新码修的缺陷原样发作（o-wakebus 实证：
11-22 补扫兜底件落地后未重启，lag 窗内行漏读→工单踹发丢死）。
Restart=always 只保「崩了重拉」，不保「新码重载」——**部署树 ff
与常驻进程生命周期之间无耦合件**。

**判词**：凡 `go run`/`go build` 常驻件，「代码已合≠已生效」——验收
运行时事实必须看进程启动时点 vs 码面 sha 的先后（journal Active
Enter timestamp 对拍 HEAD），只看文件在树≠在跑。

**修法**（deploy_watch.sh ff 点挂钩）：变更路径集→「路径前缀→单元」
映射零表化——扫 deploy/*.service ExecStart 的 `go run ./<pkg>` 现场
生成（service 文件即钦定面，新 go-run 件自动入面，不造第二份表）；
依赖闭包=`go list -deps` 实算（internal/* 共享依赖变更同样令二进
制陈——纯 ExecStart 目录前缀漏依赖层=哑尸残留）；命中→
`systemctl --user restart`，败=deploy_gorun_restart 升件+逐户部署
事实行（unit+触发路径集+base/head sha）。只踢活件
（active/activating/auto-restart/reloading）——oneshot/timer 件每拍
go run 恒新码免疫；重踹幂等无害。
