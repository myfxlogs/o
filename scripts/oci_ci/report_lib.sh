#!/usr/bin/env bash
# oci_ci/report_lib.sh —— OCI 戳跑回报格式单源（写侧=runner.sh，读侧=oci_ci_accept.sh）。
#
# 格式：key=value 单行；未知键读侧忽略，缺键=空串。固定字段：
#   sha        被测 commit（40 hex）
#   exit       0=绿；1..74=红；75=未跑（非定向谱/基线缺/环境缺——本盒 fail-open）
#   mode       check_all 档（scoped/merge-scoped=可核签；full/docs-only=不核签）
#   pkgs       实测包集（空格分隔，与 check_all .mode 包清单同源）
#   run/fail/skip  go test -json 计数（含子测试）
#   arch       跑侧 uname -m（aarch64=ARM；分歧探针据此比本盒）
#   baseline   跑侧 origin/master（scope 对拍基线）
#   netgates   闸面完整性：full=全闸在跑侧过；local:bridge-audit=桥域闸延后
#              本盒补跑（accept 必须同 sha 净树补跑绿后才落戳）
#   host/started/finished
#   log_sha256/log_bytes  整跑输出哈希（本盒拉 log 复算防伪造面）
#   note       人读备注（无换行）
#
# 原子写：tmp+mv——读侧永不见半截报告。

oci_report_write() { # $1=file 其余=key=value 对
  local f="$1" tmp; shift
  tmp="$f.tmp.$$"
  : > "$tmp" || return 1
  local kv
  for kv in "$@"; do
    printf '%s\n' "$kv" >> "$tmp" || { rm -f "$tmp"; return 1; }
  done
  mv -f "$tmp" "$f"
}

oci_report_get() { # $1=file $2=key → stdout 值（缺=空串，rc1=文件缺）
  [ -f "$1" ] || return 1
  grep -m1 "^$2=" "$1" 2>/dev/null | cut -d= -f2- || true
}
