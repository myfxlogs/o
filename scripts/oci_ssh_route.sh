#!/usr/bin/env bash
# oci_ssh_route.sh —— dev→OCI SSH 路由自愈源件（2026-11-17 管道事故根修）。
# 病灶: dev→OCI 22 口直连死(TCP banner 超时;SG 跳板可达——业主物理面);
#   止血=dev 用户 ~/.ssh/config ProxyJump 块——不入仓,下机/换人/清环境即失,
#   且桥闸/CI/部署/事件总线隧道全经此径=产线依赖藏用户配置里。
# 根修=调用侧自带路由: 探直连(6s),通→显式 -o ProxyJump=none;
#   败→-o ProxyJump=SG。恒显式=与用户 ssh_config 解耦(止血块在不在
#   同行为,删块零影响);每跑重探=直连恢复即自动回直连,无手工删块步。
# 用法(source 件,非执行件):
#   source "$SELF/oci_ssh_route.sh" [<探靶=user@host>]
#   后置 ssh:  ssh $O_OCI_JARG <host> <args...>
#   后置 git:  export GIT_SSH_COMMAND="ssh $O_OCI_JARG"
# 注: 探=连通性判非鉴权判(鉴权败两径同败不扰选径);探耗~1-6s/次,
#   同壳重复 source 一次定不叠耗(O_OCI_JARG 预设=人工覆写受尊重)。
[ -n "${O_OCI_JARG+x}" ] && return 0
_p=${1:-${O_OCI_HOST:-ubuntu@149.118.58.98}}
_j=${O_OCI_JUMP:-root@192.229.85.35}
if ssh -o BatchMode=yes -o ConnectTimeout=6 -o ProxyJump=none \
	-o StrictHostKeyChecking=accept-new "$_p" true >/dev/null 2>&1; then
	O_OCI_JARG="-o ProxyJump=none"
	echo "[oci-route] 直连通——$_p" >&2
else
	O_OCI_JARG="-o ProxyJump=$_j"
	echo "[oci-route] 直连死——$_p 转 SG 跳板 $_j" >&2
fi
