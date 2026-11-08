#!/usr/bin/env bash
# oci_ci_provision.sh —— OCI 侧 runner 备件一键装（提案「check_all 甩 OCI」档 A，件①）。
#
# 幂等：go/buf/插件版本对=跳；容器在=start；裸仓/工作 clone/单元在=补齐配置。
# 装面全落 ~/o-ci（不碰 OCI 生产面 ~/o、~/o-research、现有容器/单元）。
#
# 装件：go 1.27.3 + buf 1.74.0 + protoc-gen-{go,connect-go,go-grpc}（ci.yml 同钉）；
#       o-postgres-test 族（5434 / 5436——5435 归 o-postgres-md 生产容器，号位
#       冲突故不建 -b；testdb_lease 按容器名映射端口，族缺 b 合法）；
#       scratch 裸仓 + post-receive + 工作 clone + o-ci-runner@.service；
#       web/dist 快照（go:embed 依赖，仅闸/dev——生产前端唯一径=deploy_image.sh）。
#
# 环境：OCI_CI_SSH（缺省 ubuntu@149.118.58.98）/ OCI_CI_ROOT（缺省 /home/ubuntu/o-ci）。
set -uo pipefail

SSH_TARGET=${OCI_CI_SSH:-ubuntu@149.118.58.98}
CI_ROOT=${OCI_CI_ROOT:-/home/ubuntu/o-ci}
ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "❌ [oci-ci-provision] 非 git 树" >&2; exit 1; }
SRC="$ROOT/scripts/oci_ci"
GO_VER=1.27.3
BUF_VER=1.74.0
PG_IMAGE=timescale/timescaledb:2.30.2-pg18
PG_A=o-postgres-test
PG_C=o-postgres-test-c

say() { echo "▶ [oci-ci-provision] $*" >&2; }
die() { echo "❌ [oci-ci-provision] $*" >&2; exit 1; }
srun() { ssh -o BatchMode=yes -o ConnectTimeout=15 "$SSH_TARGET" "$@"; }

[ -f "$SRC/runner.sh" ] && [ -f "$SRC/post-receive" ] && [ -f "$SRC/o-ci-runner@.service" ] \
  || die "装件缺（$SRC）"

say "探机 $SSH_TARGET"
ARCH=$(srun 'uname -m' 2>/dev/null) || die "SSH 不通（免密键/网络）"
case "$ARCH" in
  aarch64) GOARCH=arm64; BUFARCH=aarch64 ;;
  x86_64)  GOARCH=amd64; BUFARCH=x86_64 ;;
  *) die "未支持架构: $ARCH" ;;
esac
srun 'command -v docker >/dev/null && command -v git >/dev/null && command -v python3 >/dev/null && command -v curl >/dev/null' \
  || die "OCI 缺 docker/git/python3/curl"
AVAIL_G=$(srun "df -Pk '$CI_ROOT' 2>/dev/null | tail -1 | awk '{print \$4}'" 2>/dev/null || true)
[ -z "$AVAIL_G" ] || [ "${AVAIL_G:-0}" -gt 5000000 ] || die "OCI 盘余 <5GB"

# ── Go 工具链 ──
GO_SHA=$(curl -fsSL --max-time 30 https://go.dev/dl/?mode=json 2>/dev/null \
  | python3 -c "import json,sys; print(next((f['sha256'] for r in json.load(sys.stdin) for f in r['files'] if f['filename']=='go${GO_VER}.linux-${GOARCH}.tar.gz'),''))" 2>/dev/null || true)
[ -n "$GO_SHA" ] || die "go $GO_VER checksum 拉取败"

say "go $GO_VER.linux-$GOARCH"
srun "GO_VER='$GO_VER' GOARCH='$GOARCH' GO_SHA='$GO_SHA' CI='$CI_ROOT' bash -s" <<'EOS'
set -uo pipefail
mkdir -p "$CI/bin" "$CI/reports" "$CI/assets" "$CI/gomodcache" || exit 1
if [ -x "$CI/go/bin/go" ] && "$CI/go/bin/go" version 2>/dev/null | grep -q "go${GO_VER}"; then
  echo "  go 已在：$("$CI/go/bin/go" version)"
else
  url="https://go.dev/dl/go${GO_VER}.linux-${GOARCH}.tar.gz"
  curl -fsSL --retry 3 -o /tmp/o-ci-go.tgz "$url" || { echo "  go 下载败" >&2; exit 1; }
  printf '%s  /tmp/o-ci-go.tgz\n' "$GO_SHA" | sha256sum -c - >/dev/null \
    || { echo "  go checksum 不符" >&2; rm -f /tmp/o-ci-go.tgz; exit 1; }
  rm -rf "$CI/go"; mkdir -p "$CI/go"
  tar -C "$CI/go" --strip-components=1 -xzf /tmp/o-ci-go.tgz || exit 1
  rm -f /tmp/o-ci-go.tgz
  echo "  go 装毕：$("$CI/go/bin/go" version)"
fi
EOS
[ $? = 0 ] || die "go 装败"

# ── buf ──
say "buf $BUF_VER.linux-$BUFARCH"
srun "BUF_VER='$BUF_VER' BUFARCH='$BUFARCH' CI='$CI_ROOT' bash -s" <<'EOS'
set -uo pipefail
if [ -x "$CI/bin/buf" ] && "$CI/bin/buf" --version 2>/dev/null | grep -qx "$BUF_VER"; then
  echo "  buf 已在：$("$CI/bin/buf" --version)"
else
  base="https://github.com/bufbuild/buf/releases/download/v${BUF_VER}"
  curl -fsSL --retry 3 -o /tmp/o-ci-buf "$base/buf-Linux-${BUFARCH}" || { echo "  buf 下载败" >&2; exit 1; }
  want=$(curl -fsSL --retry 3 "$base/sha256.txt" | awk -v n="buf-Linux-${BUFARCH}" '$2==n{print $1}') || true
  if [ -n "$want" ]; then
    echo "$want  /tmp/o-ci-buf" | sha256sum -c - >/dev/null || { echo "  buf checksum 不符" >&2; rm -f /tmp/o-ci-buf; exit 1; }
  else
    echo "  ⚠️ sha256.txt 无本条——按下载成功落件（校验面缺，明示非静默）" >&2
  fi
  install -m 0755 /tmp/o-ci-buf "$CI/bin/buf" || exit 1
  rm -f /tmp/o-ci-buf
  echo "  buf 装毕：$("$CI/bin/buf" --version)"
fi
EOS
[ $? = 0 ] || die "buf 装败"

# ── protoc 插件（与 .github/workflows/ci.yml 同钉）──
say "protoc 插件（go v1.36.12 / connect v2.0.1 / go-grpc v1.5.1）"
srun "CI='$CI_ROOT' bash -s" <<'EOS'
set -uo pipefail
export PATH="$CI/go/bin:$PATH"
export GOBIN="$CI/bin"
export GOMODCACHE="${GOMODCACHE:-$CI/gomodcache}"
check() { [ -x "$CI/bin/$1" ] && "$CI/bin/$1" --version 2>/dev/null | grep -q "$2"; }
need=0
check protoc-gen-go v1.36.12 || { go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12 || exit 1; need=1; }
check protoc-gen-connect-go 2.0.1 || { go install connectrpc.com/connect/v2/cmd/protoc-gen-connect-go@v2.0.1 || exit 1; need=1; }
check protoc-gen-go-grpc 1.5.1 || { go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 || exit 1; need=1; }
[ "$need" = 0 ] && echo "  插件齐（版本对）" || echo "  插件装毕"
EOS
[ $? = 0 ] || die "protoc 插件装败"

# ── 测试 PG 族（5434/5436；5435=生产 o-postgres-md 禁占）──
say "测试 PG 族（$PG_A:5434 / $PG_C:5436）"
srun "CI='$CI_ROOT' PG_IMAGE='$PG_IMAGE' PG_A='$PG_A' PG_C='$PG_C' bash -s" <<'EOS'
set -uo pipefail
if docker ps -a --format '{{.Names}}' | grep -qx 'o-postgres-test-b'; then
  echo "  ❌ 发现 o-postgres-test-b——5435 与生产 o-postgres-md 冲突，先下线该容器（testdb_lease 按名映射 5435 会注错库）" >&2
  exit 1
fi
run_pg() { # $1=name $2=port $3=volume
  local name="$1" port="$2" vol="$3"
  if docker inspect "$name" >/dev/null 2>&1; then
    docker start "$name" >/dev/null 2>&1 || true
    echo "  $name 已在（start 触发）"
  else
    docker run -d --name "$name" --restart unless-stopped \
      -m 4g --shm-size=1g \
      -e POSTGRES_DB=o_test -e POSTGRES_USER=o -e POSTGRES_PASSWORD=o \
      -p "127.0.0.1:$port:5432" \
      -v "$vol:/var/lib/postgresql" \
      --health-cmd 'pg_isready -U o -d o_test' --health-interval 10s \
      --health-timeout 5s --health-retries 10 \
      "$PG_IMAGE" postgres -c max_connections=500 \
      -c shared_preload_libraries=timescaledb,pg_stat_statements >/dev/null || return 1
    echo "  $name 建（port $port）"
  fi
}
run_pg "$PG_A" 5434 o-pgdata-test-oci || exit 1
run_pg "$PG_C" 5436 o-pgdata-test-oci-c || exit 1
for i in $(seq 1 30); do
  sa=$(docker inspect -f '{{.State.Health.Status}}' "$PG_A" 2>/dev/null || echo none)
  sc=$(docker inspect -f '{{.State.Health.Status}}' "$PG_C" 2>/dev/null || echo none)
  [ "$sa" = healthy ] && [ "$sc" = healthy ] && { echo "  双库 healthy"; exit 0; }
  sleep 2
done
echo "  ❌ PG 族未 healthy（$PG_A=$sa $PG_C=$sc）" >&2
exit 1
EOS
[ $? = 0 ] || die "测试 PG 族装败"

# ── 裸仓 + hook + 工作 clone ──
say "scratch 裸仓 + post-receive + 工作 clone"
srun "CI='$CI_ROOT' bash -s" <<'EOS'
set -uo pipefail
[ -d "$CI/scratch.git" ] || git init -q --bare "$CI/scratch.git" || exit 1
[ -d "$CI/work/.git" ] || git clone -q "$CI/scratch.git" "$CI/work" || exit 1
git -C "$CI/work" config user.email o-ci@oci.local
git -C "$CI/work" config user.name o-ci
echo "  bare+work 在（$(git -C "$CI/scratch.git" rev-parse --is-bare-repository)）"
EOS
[ $? = 0 ] || die "裸仓/clone 装败"
srun "cat > '$CI_ROOT/scratch.git/hooks/post-receive' && chmod 0755 '$CI_ROOT/scratch.git/hooks/post-receive'" \
  < "$SRC/post-receive" || die "post-receive 装败"

# ── runner + 单元 ──
say "runner 脚本 + o-ci-runner@.service"
srun "mkdir -p '$CI_ROOT/bin' \"\$HOME/.config/systemd/user\""
srun "cat > '$CI_ROOT/bin/o-ci-run-once.sh' && chmod 0755 '$CI_ROOT/bin/o-ci-run-once.sh'" < "$SRC/runner.sh" \
  || die "runner.sh 装败"
srun "cat > '$CI_ROOT/bin/report_lib.sh' && chmod 0644 '$CI_ROOT/bin/report_lib.sh'" < "$SRC/report_lib.sh" \
  || die "report_lib.sh 装败"
srun "cat > \"\$HOME/.config/systemd/user/o-ci-runner@.service\"" < "$SRC/o-ci-runner@.service" \
  || die "unit 装败"
srun "export XDG_RUNTIME_DIR=/run/user/\$(id -u); systemctl --user daemon-reload" \
  || die "systemctl --user daemon-reload 败（linger/XDG）"

# ── web/dist 快照（embed 依赖）──
say "web/dist 快照上传"
[ -f "$ROOT/web/dist/index.html" ] || die "本树无 web/dist（先构建/快照）"
tar -C "$ROOT" -czf - web/dist | srun "cat > '$CI_ROOT/assets/web-dist.tar.gz'" \
  || die "dist 快照上传败"

# ── 本仓 remote 配置 ──
URL="ssh://$SSH_TARGET$CI_ROOT/scratch.git"
if git -C "$ROOT" remote get-url oci-ci >/dev/null 2>&1; then
  CUR=$(git -C "$ROOT" remote get-url oci-ci)
  [ "$CUR" = "$URL" ] || say "⚠️ remote oci-ci 已存在且不同（$CUR）——不擅自改写"
else
  git -C "$ROOT" remote add oci-ci "$URL" || die "remote oci-ci 配置败"
  say "remote oci-ci → $URL"
fi

# ── 冒烟 ──
say "冒烟自检"
srun "CI='$CI_ROOT' bash -s" <<'EOS'
set -uo pipefail
export PATH="$CI/go/bin:$CI/bin:$PATH"
echo "  go:  $("$CI/go/bin/go" version)"
echo "  buf: $("$CI/bin/buf" --version)"
echo "  pg:  $(docker ps --format '{{.Names}}' | grep '^o-postgres-test' | tr '\n' ' ')"
echo "  unit: $(export XDG_RUNTIME_DIR=/run/user/$(id -u); systemctl --user list-unit-files 'o-ci-runner@.service' --no-legend 2>/dev/null | tr -s ' ' | cut -d' ' -f1)"
EOS
say "备件装毕——下一步：git push oci-ci <branch>（scripts/oci_ci_push.sh 包装）"
