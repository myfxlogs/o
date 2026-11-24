#!/usr/bin/env bash
# oci_ssh_route.sh —— dev→OCI SSH 路由自愈源件（2026-11-17 管道事故根修）。
# 病灶: dev→OCI 22 口直连死(TCP banner 超时;SG 跳板可达——业主物理面);
#   止血=dev 用户 ~/.ssh/config ProxyJump 块——不入仓,下机/换人/清环境即失,
#   且桥闸/CI/部署/事件总线隧道全经此径=产线依赖藏用户配置里。
# 根修=调用侧自带路由: 探直连(6s/候选),通→显式 -o ProxyJump=none;
#   全败→-o ProxyJump=SG。恒显式=与用户 ssh_config 解耦(止血块在不在
#   同行为,删块零影响);每跑重探=直连恢复即自动回直连,无手工删块步。
# 2026-11-24 tailnet 候选接入(OCI 入 tailnet=ubuntu@100.79.98.60):
#   候选序 tailnet→公网→SG;tailnet 传输层独立于 GFW 跨境段(公网直连
#   已实证会断),tailscaled 死自动落回后两径——零新增故障面。
# 用法(source 件,非执行件):
#   source "$SELF/oci_ssh_route.sh" [<探靶=user@host>]
#   后置 ssh:  ssh $O_OCI_JARG <host> <args...>
#   后置 git:  export GIT_SSH_COMMAND="ssh $O_OCI_JARG"
# 注: 探=连通性判非鉴权判(鉴权败两径同败不扰选径);探耗~1-6s/次,
#   同壳重复 source 一次定不叠耗(O_OCI_JARG 预设=人工覆写受尊重)。
[ -n "${O_OCI_JARG+x}" ] && return 0
_p=${1:-${O_OCI_HOST:-}}
_ts=${O_OCI_TS_HOST:-ubuntu@100.79.98.60}
_d=${O_OCI_HOST:-ubuntu@149.118.58.98}
_j=${O_OCI_JUMP:-root@192.229.85.35}
# 候选序=tailnet→公网直连→SG跳板（tailnet 传输层独立于 GFW 跨境段——
# 11-17 公网直连死实证在案；tailscaled 死则自动落回公网/跳板，零新增故障面）。
# 显式传参/$O_OCI_HOST=人工定点径，只探该径不叠候选。
if [ -n "$_p" ]; then
	_cands="$_p"
else
	_cands="$_ts $_d"
fi
for _c in $_cands; do
	if ssh -o BatchMode=yes -o ConnectTimeout=6 -o ProxyJump=none \
		-o StrictHostKeyChecking=accept-new "$_c" true >/dev/null 2>&1; then
		O_OCI_JARG="-o ProxyJump=none"
		O_OCI_HOST="$_c"
		echo "[oci-route] 直连通——$_c" >&2
		break
	fi
done
if [ -z "${O_OCI_JARG+x}" ]; then
	O_OCI_JARG="-o ProxyJump=$_j"
	echo "[oci-route] 直连全死——转 SG 跳板 $_j" >&2
fi
unset _cands _c
