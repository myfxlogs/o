#!/usr/bin/env bash
# check_bridge_registry.sh —— T-0223 桥域对账闸（D-0036 §12.4 前置闸）一体三列：
#  ①授权列：domain.tsv 桥域表 × SG 实查授权——bridge-reverse 源表 projector_rw
#    SELECT 必在（T-0214 md_earn_rate 漏配崩环实证）；同表 research_rw SELECT
#    按需钉（研究面隧道读径）；bridge 正向表 projector_rw SELECT 钉（投影镜像
#    源读）——差即红报表名×角色×权。
#  ②存在性列：迁移册派生触发器清单（fin_*→SG 侧/md_*→OCI 侧，仅桥域表）×
#    pg_trigger 实查——缺即红报表名×触发器名（T-0194 静默丢失疤）。
#  ③模板列：deploy/ 模板 DSN 字段（user@host:port）× OCI 实机 env 同键对拍
#    ——漂移即红报件名×键×双值（模板即图，漂移=错图施工）。
# 挂径：check_all 步进（pre-push 绿戳闸）+check_schema_live.sh 尾调（哨兵双径）。
# 桩测友好：声明侧全读本仓——对拍对象=SG/OCI SSH 实查；doctored checkout 即
# 可验三红不触产。
set -euo pipefail
cd "$(dirname "$0")/.."
SG_HOST="${SG_HOST:-root@192.229.85.35}"
MD_HOST="${MD_HOST:-ubuntu@149.118.58.98}"
fail=0
red() { echo "FAIL[bridge-audit] $1" >&2; fail=1; }

sg_q() { timeout 20 ssh -n -o BatchMode=yes -o ConnectTimeout=8 "$SG_HOST" \
	"docker exec o-postgres psql -U o -d o -Atc \"$1\"" 2>/dev/null | tr -d ' '; }
md_q() { timeout 20 ssh -n -o BatchMode=yes -o ConnectTimeout=8 "$MD_HOST" \
	"sudo docker exec o-postgres-md psql -U o -d o -Atc \"$1\"" 2>/dev/null | tr -d ' '; }
oci_f() { timeout 15 ssh -n -o BatchMode=yes -o ConnectTimeout=8 "$MD_HOST" "cat $1" 2>/dev/null; }

# ── ① 授权列 ─────────────────────────────────────────────
# domain.tsv 桥域表集（bridge+bridge-reverse——登记表=名册唯一真源）。
mapfile -t rev_tables < <(grep -P '\tbridge-reverse$' migrations/domain.tsv | cut -f1)
mapfile -t fwd_tables < <(grep -P '\tbridge$' migrations/domain.tsv | cut -f1)
# 期望授权矩阵（钉死钦定面——扩权不判，缺权红）：
#  projector_rw:SELECT×全桥表（反向源读+正向源读）；research_rw:SELECT×reverse 表
want() { echo "$1"; }
for t in "${rev_tables[@]}" "${fwd_tables[@]}"; do
	for g in "projector_rw|SELECT"; do
		role="${g%%|*}"; priv="${g##*|}"
		n=$(sg_q "SELECT count(*) FROM information_schema.role_table_grants WHERE grantee='$role' AND table_name='$t' AND privilege_type='$priv'")
		[ "$n" = "1" ] || red "授权缺: SG $role.$priv on $t（domain.tsv 桥域册有、实机无——T-0214 同型漏配）"
	done
done
for t in "${rev_tables[@]}"; do
	n=$(sg_q "SELECT count(*) FROM information_schema.role_table_grants WHERE grantee='research_rw' AND table_name='$t' AND privilege_type='SELECT'")
	[ "$n" = "1" ] || red "授权缺: SG research_rw.SELECT on $t（reverse 源表研究面按需读）"
done
echo "OK[bridge-audit] 授权列: $((${#rev_tables[@]}+${#fwd_tables[@]})) 桥表×projector_rw.SELECT + ${#rev_tables[@]} reverse×research_rw.SELECT 齐"

# ── ② 触发器存在性列 ─────────────────────────────────────
# 迁移册派生：TRIGGER <name> ... ON <table>，按文件名链分侧（fin_*→SG，md_*→OCI）。
# 清单空=红（抽取败静默过=闸失效同病灶）。
TRG_LIST=$(mktemp)
for m in migrations/fin_*.sql migrations/md_*.sql; do
	[ -f "$m" ] || continue
	side=fin; case "$(basename "$m")" in md_*) side=md;; esac
	grep -oE 'TRIGGER [a-z_]+ (AFTER|BEFORE|INSTEAD OF)[A-Z ]* ON [a-z_]+' "$m" \
		| sed -E "s/TRIGGER ([a-z_]+) .* ON ([a-z_]+)/$side\t\1\t\2/" >> "$TRG_LIST" || true # grep 无匹配+pipefail→set -e 斩（静默死病灶实证）
done
sort -u -o "$TRG_LIST" "$TRG_LIST"
trgn=$(wc -l < "$TRG_LIST")
[ "$trgn" -gt 0 ] || red "触发器清单空——迁移册抽取败（闸失效 fail-closed）"
while IFS=$'\t' read -r side tg tbl; do
	[ -n "$tg" ] || continue
	is_bridge=0
	for t in "${rev_tables[@]}" "${fwd_tables[@]}"; do [ "$t" = "$tbl" ] && is_bridge=1; done
	[ "$is_bridge" = 1 ] || continue
	if [ "$side" = fin ]; then
		n=$(sg_q "SELECT count(*) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid WHERE NOT t.tgisinternal AND c.relname='$tbl' AND t.tgname='$tg'")
	else
		n=$(md_q "SELECT count(*) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid WHERE NOT t.tgisinternal AND c.relname='$tbl' AND t.tgname='$tg'")
	fi
	[ "$n" = "1" ] || red "触发器缺: $side 侧 $tbl.$tg（迁移册声明在、pg_trigger 实查无——T-0194 同型静默丢失）"
done < "$TRG_LIST"
rm -f "$TRG_LIST"
echo "OK[bridge-audit] 存在性列: ${trgn} 声明×桥域实查齐（含 arch-源水印回退径位 T-0230 接入）"

# ── ③ 模板漂移列 ─────────────────────────────────────────
# 模板 DSN 键 × 实机 env 同键对拍（user@host:port——PASSWORD 占位不比对密文）。
dsn_fields() { grep -oE "$2=postgres://[a-z_]+(:[^@\"]*)?@[0-9.]+:[0-9]+" "$1" 2>/dev/null \
	| sed -E "s|$2=postgres://([a-z_]+)(:[^@\"]*)?@([0-9.]+:[0-9]+)|\1@\3|" | head -1; }
env_field() { # $1=remote file $2=key
	oci_f "$1" | grep -oE "$2=postgres://[a-z_]+(:[^@\"]*)?@[0-9.]+:[0-9]+" \
	| sed -E "s|$2=postgres://([a-z_]+)(:[^@\"]*)?@([0-9.]+:[0-9]+)|\1@\3|" | head -1; }
pairs="deploy/o-collecto/compose.yml|SG_PG_DSN|/home/ubuntu/o-collecto.env
deploy/o-projector.env.example|SG_PG_DSN|/home/ubuntu/o-projector.env
deploy/o-projector.env.example|MD_PG_DSN|/home/ubuntu/o-projector.env
deploy/o-research.env.example|SG_PG_DSN|/home/ubuntu/o-research/env
deploy/o-research.env.example|ROSTER_PG_DSN|/home/ubuntu/o-research/env
deploy/o-research.env.example|FIN_PG_DSN|/home/ubuntu/o-research/env"
# compose 内嵌 DSN 形（SG_PG_DSN: postgres://...）：键分隔符是 ': ' 非 '='。
tpl_field() { # $1=file $2=key —— env 形 KEY= 与 compose 形 KEY: 双兼容
	grep -oE "$2[=:] ?postgres://[a-z_]+(:[^@\" ]*)?@[0-9.]+:[0-9]+" "$1" 2>/dev/null \
	| sed -E "s|$2[=:] ?postgres://([a-z_]+)(:[^@\" ]*)?@([0-9.]+:[0-9]+)|\1@\3|" | head -1; }
while IFS='|' read -r tpl key envf; do
	[ -n "$tpl" ] || continue
	tv=$(tpl_field "$tpl" "$key")
	av=$(env_field "$envf" "$key")
	if [ -z "$av" ]; then red "模板对拍实机读缺: $envf $key（漂移面 fail-closed）"; continue; fi
	if [ -z "$tv" ]; then red "模板键缺: $tpl $key（模板即图——键缺=图缺件）"; continue; fi
	[ "$tv" = "$av" ] || red "模板漂移: $tpl $key 册=$tv 实机=$av"
done <<< "$pairs"
echo "OK[bridge-audit] 模板列: deploy 三件套 DSN 键×实机对拍过"

[ "$fail" = 0 ] && echo "OK[bridge-audit] 三列全绿" || exit 1
