---
id: docker-logopts-freshbox-diskbomb
title: docker json-file 日志无上限——daemon.json log-opts 不入供给册=新盒裸生必爆
kind: pit
date: 2026-11-27
source: 工单 O-41390（值守卫生批⑤——OCI 47G/99% 日志爆仓手修后机械化断根）
---

# daemon.json log-opts 供给断言——「手修救当盒，新盒按册复立=裸生同病」（O-41390）

**坑形**：docker 默认 `json-file` 日志驱动**单件无上限**。2026-11-27/28
OCI 盒磁盘 47G/99%——容器日志长爆，手修 `/etc/docker/daemon.json` 落
`{"log-driver":"json-file","log-opts":{"max-size":"50m","max-file":"3"}}`
+restart 只救当盒：`oci_bootstrap.sh stage_sys` 装 docker.io 后零断言，
新盒复立=同病数月后重演；`oci_ci_provision.sh` 建测试容器跑在未配
daemon 上=同裸。SG 盒早前已配（实证），CN 构建盒有 registry-mirrors+
runtimes 但无 log-opts（sudo 手修径另案）。

**根修两件套**（供给册面）：

1. **bootstrap 供给**：`stage_sys` docker.io 装后、任何容器建前跑
   `oci_docker_logopts.sh ensure`——stage_sys 零容器=restart 安全窗；
   幂等已对零变不 restart。
2. **provision 断言**：首个 `docker run` 前 `assert`——check 只读断言
   →缺配 `sudo -n` 自举（OCI ubuntu 免密 sudo）→不可得 die 显式报错
   带手修行，**禁裸建容器**。

**「合并不覆写」律**（本件核心判别式）：daemon.json 可载
registry-mirrors/nvidia runtimes 等异主键——**覆写=断镜像源=昨日
断粮重演**。合并实现=python3 JSON 层只补/改 `log-driver`+`log-opts`
两键，余键原样保；件坏 JSON/顶层非对象=止损不覆写。单源件
`scripts/oci_docker_logopts.sh`（check/ensure/assert 三模）供
bootstrap/provision 双径共吃，禁两套漂移。

**判别式**：凡「盒级 daemon/服务配置」入供给册时三问——①缺省值是否
   含隐性无界资源（日志/临时件/缓存——爆的都是没写上限的）；②配置
   件是否可载异主键（合并不覆写是默认律非可选）；③供给断言是否
   fail-closed（缺配能修则修、不能修则 die 带修法行，禁静默裸建）。

**测试缝**：`DOCKER_DAEMON_JSON`/`DOCKER_LOGOPTS_RESTART` 环境变量
   仅供本地夹具（`oci_docker_logopts_test.sh` 三案+幂等+负案对抗径），
   生产勿设。
