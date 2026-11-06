# T-0398 O-Dev 施工提示词——部署窗触发律

领单唯一权威范围：
/opt/o/docs/tasks/T-0398-部署窗触发律-攒批变职责.md
（AGENTS/collab/施工席SKILL 先读；票面范围/红线即全部授权）

要点轴：判据两条件机读化——①origin/master↔生产部署 SHA
间 diff 含镜像面路径（镜像面≈非 docs/skills/md 白名单之补集,
先看 deploy_image.sh/Dockerfile 实际入镜像面再钦定清单）
②无在飞码件(systemd o-worker-* 实查)+无待裁码件(ops_event
推导——dm_queue.sh 同源口径)。三态 exit。SKILL 部署窗节照
票面语义落。

红线守：deploy_image.sh/SHA tag/推戳闸一行不碰；判据不读
票面状态行；docs-only 永不触发。

收口：新 wt→bootstrap→check_all 绿戳→收口文+推支不合并不部署。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | 镜像面路径清单未钦定死——给了推导法 | 防自猜 | 工人先读 deploy_image.sh/Dockerfile 实证定清单落票面回填 |
