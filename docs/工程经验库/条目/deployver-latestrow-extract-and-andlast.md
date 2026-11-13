---
schema_version: 2
id: deployver-latestrow-extract-and-andlast
kind: pitfall
risk: medium
status: verified
keywords: [部署版读取, DEPLOY 行类, deploy-oci, regexp_match 空, 短路 exit 尾句, systemd FAILURE 噪音, self_state 部署问号, 假滞差]
scope: [cmd/core/selfstate_beat.go, scripts/deploy_watch.sh, ops_event DEPLOY 行面]
summary: "部署版读取面双坑：①「最新行+extract 标记」式查询若行类不先滤，异类行同谓词入列→extract 必空（deploy-oci 臂部署行 seat/event/rc=0 全同却无 o-core: 标记→self_state 部署=?、deploy_watch 假滞差触发多余部署）；②bash 末句 [ cond ] && action 短路——cond 假即以 1 收脚本，成功路径 systemd 标 FAILURE 掩真败。"
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
---

# 部署版读取双坑——最新行 extract 不滤行类 + && 末句假值传染（#13283）

**实案**：deploy_oci projector 臂部署落 `seat='DEPLOY' event='部署'
rc=0` 行但 note 无 `o-core:` 标记——self_state 读「最新 rc=0
部署行再 extract o-core:」= `部署=?` 持续 9min+；deploy_watch
同式读 DEPLOYED=空→判假滞差起部署（恰逢真滞差掩盖）；部署
rc=0 而 unit 标 `Failed with result 'exit-code'`——三症同帧。

**坑① 最新行 extract 面**：`SELECT ... WHERE seat/event/rc=0
ORDER BY id DESC LIMIT 1` 后 `regexp_match(note,'o-core:...')`
——行类（o-core image 部署 vs OCI 臂部署）同 seat/event/rc
共表，取最新行后再 extract 标记=异类行必得空。修=**行类滤先
于 extract**：WHERE 加 `note LIKE '%o-core:%'`（selfstate_beat.go
与 deploy_watch.sh 同式归一）。判词律：凡「最新 N 行取字段」
式查询，谓词面必须把目标行类锁全——靠后 extract 滤=隐性
行类混读。

**坑② && 短路末句**：`[ $rc -ne 0 ] && esc ...` 作脚本末句——
rc=0 时 test 返假即以 1 收脚本（`set -uo pipefail` 不救——脚本
出口码=末句出口码），成功部署 systemd 也标 FAILURE，真假败
不可分。修=显式 `if..exit 1 / exit 0`——条件动作块永远别用
`&&` 单行作函数/脚本末句。

**判词律**：systemd unit「Failed exit-code」先核末句形态——
`&&`/`||` 尾句、while-read 循环末次 read 假值、pipefail 下
grep 空匹配三形同坑（后者已册 while-read-ssh-stdin-eat 条目）。
