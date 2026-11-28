#!/usr/bin/env bash
# oci_docker_logopts.sh —— docker daemon.json log-opts 供给件（O-41390）。
#
# 病灶：docker json-file 日志单件无上限——OCI 盒 47G/99% 爆仓实证
# （2026-11-27/28）。daemon 默认零上限，新盒按供给册复立=裸生同病。
# 本件=「合并不覆写」机械化：保既有键（registry-mirrors/runtimes 实证
# 可载——覆写即断镜像源重演断粮）只补/改 log-driver+log-opts。
#
# 三模（单源——oci_bootstrap.sh stage_sys 直跑 / oci_ci_provision.sh
# 远端 assert 自举，禁两套漂移）：
#   check   只读断言：rc0=log-opts 已对，rc1=缺/不对/件坏（不改一字）
#   ensure  合并写入+按需 restart docker：rc0=已对零变或已写已重启，
#           rc1=败（件坏/写败/restart 败——均显式报因）
#   assert  check→ensure 自举：已对直接 rc0；缺配先试 sudo -n 自举
#           （OCI ubuntu 标准免密 sudo）；sudo 不可得→印修法行 rc1
#
# 测试缝（夹具专用，生产勿设）：
#   DOCKER_DAEMON_JSON  覆盖 daemon.json 路径（缺省 /etc/docker/daemon.json）
#   DOCKER_LOGOPTS_RESTART  覆盖 restart 命令（缺省 systemctl restart docker）
set -uo pipefail

DAEMON_JSON="${DOCKER_DAEMON_JSON:-/etc/docker/daemon.json}"
RESTART_CMD="${DOCKER_LOGOPTS_RESTART:-systemctl restart docker}"
SELF="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

# _merge —— python3 JSON 合并（Ubuntu 24.04 基装）：
#   rc0=已写（变），rc42=已对零变，rc1=败（件坏 JSON/读写败）
_merge() {
	python3 - "$DAEMON_JSON" <<'PYEOF'
import json, os, sys

path = sys.argv[1]
want = {"log-driver": "json-file",
        "log-opts": {"max-size": "50m", "max-file": "3"}}
cfg = {}
if os.path.exists(path):
    try:
        with open(path) as f:
            cfg = json.load(f)
    except Exception as e:
        print(f"daemon.json 解析败（不覆写止损）: {e}", file=sys.stderr)
        sys.exit(1)
    if not isinstance(cfg, dict):
        print("daemon.json 顶层非对象（不覆写止损）", file=sys.stderr)
        sys.exit(1)
changed = False
for k, v in want.items():
    if cfg.get(k) != v:
        cfg[k] = v
        changed = True
if not changed:
    sys.exit(42)
tmp = path + ".oci-logopts.tmp"
try:
    with open(tmp, "w") as f:
        json.dump(cfg, f, indent=2, ensure_ascii=False)
        f.write("\n")
    os.chmod(tmp, 0o644)
    os.replace(tmp, path)
except Exception as e:
    print(f"daemon.json 写败: {e}", file=sys.stderr)
    sys.exit(1)
print(f"daemon.json log-opts 已合并写入（既有键全保）: {path}")
sys.exit(0)
PYEOF
}

# _check —— 只读断言（与 _merge 同判据，禁两套口径漂移）
_check() {
	python3 - "$DAEMON_JSON" <<'PYEOF'
import json, sys
try:
    with open(sys.argv[1]) as f:
        cfg = json.load(f)
except Exception:
    sys.exit(1)
lo = cfg.get("log-opts") if isinstance(cfg, dict) else None
ok = (isinstance(cfg, dict)
      and cfg.get("log-driver") == "json-file"
      and isinstance(lo, dict)
      and lo.get("max-size") == "50m"
      and lo.get("max-file") == "3")
sys.exit(0 if ok else 1)
PYEOF
}

_fix_hint() {
	cat <<'EOF'
  手修径（盒上 root/免密 sudo）：
    python3 - <<'PY'
import json,os
p="/etc/docker/daemon.json"; cfg=json.load(open(p)) if os.path.exists(p) else {}
cfg["log-driver"]="json-file"; cfg["log-opts"]={"max-size":"50m","max-file":"3"}
open(p+".tmp","w").write(json.dumps(cfg,indent=2)+"\n"); os.chmod(p+".tmp",0o644); os.replace(p+".tmp",p)
PY
    systemctl restart docker
  病灶/判别式：docs/工程经验库/条目/docker-logopts-freshbox-diskbomb.md
EOF
}

case "${1:-}" in
check)
	_check
	;;
ensure)
	_merge
	rc=$?
	if [ "$rc" = 42 ]; then
		echo "log-opts 已对——幂等零变不 restart"
		exit 0
	elif [ "$rc" != 0 ]; then
		echo "❌ [docker-logopts] 合并写败 rc=$rc" >&2
		exit 1
	fi
	# rc0=已写——stage_sys 零容器=安全窗；provision 侧容器在册时 restart
	#  bounce 由 --restart unless-stopped 自愈（CI 盒可接受，幂等重跑已对零变）
	$RESTART_CMD || { echo "❌ [docker-logopts] daemon restart 败——配置已写，手跑: $RESTART_CMD" >&2; exit 1; }
	echo "daemon restart 毕——log-opts 生效"
	;;
assert)
	if _check; then
		echo "daemon log-opts 已对——零变"
		exit 0
	fi
	# 缺配/件坏——先试 sudo 自举（非 root 径；root 直跑 ensure）。
	# sudo/ensure 败因（要密码/件坏）stderr 保留——修法行补尾不掩诊断。
	if [ "$(id -u)" = "0" ]; then
		"$SELF" ensure || { _fix_hint; exit 1; }
	elif sudo -n "$SELF" ensure; then
		:
	else
		echo "❌ [docker-logopts] daemon.json 缺 log-opts 且 sudo 修复不可得——禁裸建容器" >&2
		_fix_hint
		exit 1
	fi
	;;
*)
	echo "usage: $0 {check|ensure|assert}" >&2
	exit 2
	;;
esac
