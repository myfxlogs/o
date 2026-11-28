---
id: docker-base-image-famine-sg-ferry
title: CN盒docker基础镜像断粮——镜像源blob后端死时SG船运径救命
kind: pit
date: 2026-11-28
source: O-41387部署实证（deploy_image构建卡FROM——daocloud R2后端TLS全断）
---

# 基础镜像断粮的救径=SG船运，不是等源活（2026-11-28实证）

**坑形**：`deploy_at_sha` 本机构建 `o-core` 需 `golang/node/alpine`
三底件。CN盒 docker.io 直连不通（预期内），全靠 registry-mirrors；
`docker.m.daocloud.io` 元数据面活着（401正常），但其 blob 后端
`image-mirror.r2.daocloud.vip` TLS握手全断——**manifest能拉、blob全EOF**，
表现为构建卡死在 FROM 层「failed to copy: EOF」。1panel.live/
docker.nju.edu.cn(403)/atomgit 同日全试均不通；7889/1080 代理口
亦不通。**触发诱因易被忽视**：`docker image prune/builder prune`
清底后首次部署=必拉冷底——剪完缓存的第一次部署就是断粮暴露时刻。

**救径（实证几分钟通）**：

```bash
# SG盒(部署管道主径,带宽经过实证)拉件+打包
ssh sg 'docker pull golang:1.27-alpine node:24-alpine alpine:3.20
  >/dev/null 2>&1; docker save golang:1.27-alpine node:24-alpine
  alpine:3.20 | gzip -1 > /tmp/base-imgs.tar.gz'
scp -q sg:/tmp/base-imgs.tar.gz /tmp/
docker load -i /tmp/base-imgs.tar.gz   # FROM本地命中=绕镜像源
```

**判别式**：构建卡 FROM `failed to copy: EOF`∧`curl <mirror>/v2/`
通而 blob 域 TLS 断=此病灶；救径判据=本机/近源盒能否直连
docker.io（Oracle SG/AWS 面可）→ save|scp|load 绕源。
**禁径**：OCI tailnet scp（~60KB/s DERP中继——134MB件实测
小时级，走SG部署主径~400KB/s+几分钟）。
