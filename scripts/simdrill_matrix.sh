#!/usr/bin/env bash
# simdrill_matrix.sh —— 演习凭据产线臂（T-0523 能力债探针行使半）：
# SG cron 日拍——o-core:latest 镜像内 /bin/simdrill -matrix 全格批跑，
# 二进制随部署 sha 零漂移；fin_exec_drill 凭据行即心跳（能力债探针
# 「7d 绿凭据」不变量的自持源——臂死=凭据陈=探针如实点火）。
# 用法：crontab `15 4 * * * /opt/o/scripts/simdrill_matrix.sh >> /var/log/o-simdrill.log 2>&1`
set -uo pipefail
IMG="${SIMDRILL_IMG:-o-core:latest}"
DSN="${DB_DSN:-postgres://o:o@127.0.0.1:15433/o?sslmode=disable}"
# --entrypoint 必给——镜像 ENTRYPOINT 是 /bin/core（exec 形），
# 裸 docker run img /bin/simdrill 会被拼成「/bin/core /bin/simdrill」
# 跑成 core 非演习（产线实证 2026-11-10——容器 Up 10min 零凭据行）。
exec docker run --rm --network host --entrypoint /bin/simdrill \
	-e "DB_DSN=$DSN" "$IMG" -matrix
