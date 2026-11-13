---
schema_version: 2
id: projector-wake-cap1-starvation
kind: pitfall
risk: high
status: verified
keywords: [projector, ev_signal, wake cap1, 塌陷, 道饿死, md_signal 陈, 假断流升件, mirrorStaleSince 掩蔽, depth5]
scope: [internal/projector, internal/platform/events, ops_face 断流探针, SG 镜像面]
summary: events wake chan cap=1 塌陷期丢弃新帧——ev_ticks 洪泛（1s×千符）占槽使 ev_signal/ev_dcd 名永不达 fn，md_signal 桥陈>2min 触发假断流升件；sentinela mirrorStaleSince 用 max()全局被高频道掩蔽盲报平安。修=mdWake 唤醒名只作叫醒源，每醒全 ts 道过闸（水印+min-pass 兜幂等）+哨按表分段。判「feed 断」前必先对拍 MD 正本——SG 镜像陈≠上游断。
first_seen: 2026-11-13
last_verified: 2026-11-13
skills: []
---

# projector wake cap1 道饿死——低频道塌陷丢弃成假断流（#13226/#13227）

**实案**：SG `md_signal` 全 kind 陈 7-10min（depth5/basis_ann/
funding_view 同冻）触发 depth5「面断流」升件两连——但 OCI MD
正本鲜 1s、sigdist 每分钟打在案、ticks/markidx 镜像 0.3s 鲜。
手动 UPDATE MD 行不镜像=通道死实证；projector restart 后仅
短期回鲜（cap1 竞争仍在）。

**根因**：`events.listenSession.wake chan string cap=1`——
通知到而槽占=`default` 直接丢弃新帧。`ev_ticks`（千符订阅
~1s/拍）恒占槽位，`ev_signal`（60s 拍）/`ev_dcd`（低频）落地
即被弃——唤醒名永不达 mdWake。水面以下沉默饿死：无 error、
sentinela 不响（`mirrorStaleSince` max()全局——ticks 鲜即
「非陈」）、depth5 探针读 SG 镜像误判面断流。

**修法（T-13230）**：
①`mdWake` 唤醒名只作叫醒源——任何 ts 帧到=全五道过一遍
`projectTsTable`（谓词切片+水印+min-pass 闸使空拍零代价；
signal 道无谓词全量 916 行扫 min-pass 抬 1s→5s 防自伤洪泛，
仍远<2min dead 阈）。
②`mirrorStaleSince` 补按表判：快道鲜∧signal 陈>5min=道饿死
签名（dcd/funding 低频不入判——其陈窗=正常拍）。

**判词律**：凡「某 feed 断流」升件，先对拍**正本**再定——
SG 镜像陈≠上游断，三步分诊：①MD 正本鲜度 ②镜像桥活否
（ticks 同桥鲜？）③触发器/wake 径在否。镜像陈+正本鲜=桥件
病灶非采集病灶，勿去追 venue/订阅面。

**反面**：低频道「陈但镜像对账在走」可能是正常拍窗——dcd/
funding 类按写拍判，勿按 dead 阈一刀切。
