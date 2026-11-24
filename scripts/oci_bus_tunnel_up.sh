#!/usr/bin/env bash
# oci_bus_tunnel_up.sh —— o-oci-bus-tunnel ExecStart 壳(2026-11-17
# 管道事故根修件): 每次重起先探直连定 ProxyJump(显式解耦用户
# ssh_config),直连复=下次重起自动归——回退径自然作废。
# 转发面/钥同原 unit 行(localhost:15436→OCI eventbus,o_oci_bus 专钥)。
set -euo pipefail
SELF=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=/dev/null
# 丢参让候选序活——route 件传参=pin 单候选（oci-ts-route 合后本行
# 仍传默认公网 IP=tailnet 候选永不到达）。人工定点改经 env
# O_OCI_HOST（route 件内 ${1:-${O_OCI_HOST:-}} pin 语义同效）。
source "$SELF/oci_ssh_route.sh"
exec /usr/bin/ssh -N -i "$HOME/.ssh/o_oci_bus" $O_OCI_JARG \
	-o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 \
	-L 15436:localhost:15436 "${O_OCI_HOST:-ubuntu@149.118.58.98}"
