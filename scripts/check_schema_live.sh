#!/usr/bin/env bash
# check_schema_live.sh —— T-0181 漂移哨兵：SG fin/dev md 各查
# max(schema_migrations.version) 对照登记表最大号，落后即红——「已合并≠
# 已执行」灰色态机械报（哨兵只读对照不自动补迁）。用法：
# scripts/check_schema_live.sh [SG_PG_DSN] [MD_PG_DSN]（缺=容器探测）。
set -euo pipefail
# 登记表最大号（单源——migrate_table 登记链是合法迁移清单；按链名取号）
max_of() { grep -oE '\{[0-9]+, "[a-z]+_[0-9]+' internal/store/pgstore/migrate_table.go 2>/dev/null || true; }
SG_DSN="${1:-}"
MD_DSN="${2:-}"
# T-0181 返修：fin 探针改 SSH 径打 SG 真库（docker port 自动探测=本机
# 幻影库读空假红病灶根修——哨兵面=SG 远端权威库非本机探测）。
SG_HOST="${SG_HOST:-root@192.229.85.35}"
if [ -z "$SG_DSN" ]; then
	SG_DSN="ssh"  # SSH 径标记（sg_live() 分派 ssh docker exec 查询）
fi
# T-0211 捎带：md 链正本已迁 OCI 盒（dev o-postgres-md 排水停机）——
# 哨兵 md 探针改 SSH 径打 OCI 权威库（本机容器幻影读空同族坑）。
MD_HOST="${MD_HOST:-ubuntu@149.118.58.98}"
md_q() { # $1=SQL——SSH 径 sudo docker exec（OCI ubuntu 免密 sudo 实证）
	timeout 20 ssh -o BatchMode=yes -o ConnectTimeout=8 "$MD_HOST" \
		"sudo docker exec o-postgres-md psql -U o -d o -tc \"$1\"" 2>/dev/null | tr -d ' '
}
if [ -z "$MD_DSN" ]; then
	MD_DSN="ssh"  # SSH 径标记（md 探针走 md_q）
fi
fail=0
for dom in fin md; do
	reg=$(grep -oE "\{[0-9]+, \"${dom}_[0-9]+" internal/store/pgstore/migrate_table.go \
		| grep -oE '^\{[0-9]+' | tr -d '{' | sort -n | tail -1)
	dsn=$([ "$dom" = fin ] && echo "$SG_DSN" || echo "$MD_DSN")
	if [ "$dom" = fin ] && [ "$SG_DSN" = "ssh" ]; then
		live=$(timeout 20 ssh -o BatchMode=yes -o ConnectTimeout=8 "$SG_HOST" \
			"docker exec o-postgres psql -U o -d o -tc 'SELECT COALESCE(max(version),0) FROM schema_migrations'" 2>/dev/null | tr -d ' ' || true)
	elif [ "$dom" = md ] && [ "$MD_DSN" = "ssh" ]; then
		live=$(md_q 'SELECT COALESCE(max(version),0) FROM schema_migrations')
	else
		live=$(timeout 15 psql "$dsn" -tc "SELECT COALESCE(max(version),0) FROM schema_migrations" 2>/dev/null | tr -d ' ' || true)
	fi
	if [ -z "$live" ]; then
		echo "check_schema_live: $dom 链 schema_migrations 读败——fail-closed 红" >&2
		exit 1
	fi
	if [ "$live" -lt "$reg" ]; then
		echo "FAIL: $dom 链落后——live=$live 登记表=$reg（已合并≠已执行灰色态）" >&2
		fail=1
	else
		echo "OK: $dom 链 live=$live ≥ 登记表=$reg"
	fi
done

# —— 盘漂哨兵（T-0181 追加面并案：SG 58G 盘曾被 nox/phase 死卷悄积 18G
# 无人量——docker system df 卷/镜像/缓存三项超阈即告警入 rejects；
# 输出连容器名一起报防按名误删——alphaforge-postgres 挂卷名
# ant_postgres_data=错名绑定实证教训）。
VOL_WARN_GB="${VOL_WARN_GB:-40}"      # SG 默认卷警线（可 config 化）
BC_WARN_GB="${BC_WARN_GB:-100}"       # dev 默认 buildcache 警线
to_gb() { # docker size 字符串→GB 粗值（kB/MB/GB/TB 四档）
	local v=$1 u=$2
	case $u in
		TB) echo "$v * 1024" | bc | cut -d. -f1 ;;
		GB) echo "$v" | cut -d. -f1 ;;
		MB) echo "$v / 1024" | bc | cut -d. -f1 ;;
		*)  echo 0 ;;
	esac
}
timeout 30 docker system df --format '{{.Type}}|{{.Size}}' 2>/dev/null | while IFS='|' read -r typ sz; do
	[ -n "$typ" ] || continue
	tk=$(echo "$typ" | tr -d ' ' | tr 'A-Z' 'a-z') # 型名归一（Local Volumes→localvolumes）
	v=${sz%% *}; u=${sz##* }
	[ "$u" = "$sz" ] && u="GB" # 无单位=GB 兜底（62GB→v=62 u=GB）
	g=$(to_gb "$v" "$u")
	case $tk in
		*volumes*) th=$VOL_WARN_GB ;;
		*cache*|*images*) th=$BC_WARN_GB ;;
		*) th=999999 ;;
	esac
	if [ "${g:-0}" -gt "$th" ]; then
		echo "CRITICAL: 盘漂——$typ $sz 超阈 ${th}GB（容器面见下防按名误删）" >&2
		timeout 8 docker ps --format '  {{.Names}} {{.Image}}' | head -8 >&2
		fail=1
	else
		echo "OK: $typ $sz ≤ 阈 ${th}GB"
	fi
done

# —— T-0194 普查纠偏（返修：exit 前移位+双域件双侧验）：触发器预期集
# =migrations/*.sql 全谱定义件；侧别=ON <table> →domain.tsv 域查表
# （双域表双侧验——dev 侧再灭失不可见病灶根修）。
while IFS= read -r line; do
	tg=$(echo "$line" | awk '{print $NF}')
	tbl=$(echo "$line" | sed -nE 's/.* ON ([a-z_0-9]+) .*/\1/p')
	dom=$(awk -F'\t' -v t="$tbl" '$1==t{print $2}' migrations/domain.tsv 2>/dev/null | head -1)
	case "$dom" in
		fin)      sides="fin" ;;
		md)       sides="md" ;;
		*)        sides="fin md" ;; # 双域/未知表=双侧验
	esac
	for d in $sides; do
		if [ "$d" = fin ]; then
			hit=$(timeout 20 ssh -o BatchMode=yes -o ConnectTimeout=8 "$SG_HOST" \
				"docker exec o-postgres psql -U o -d o -tc \"SELECT 1 FROM pg_trigger WHERE tgname='$tg'\"" 2>/dev/null | tr -d ' ')
		else
			hit=$(md_q "SELECT 1 FROM pg_trigger WHERE tgname='$tg'")
		fi
		if [ "$hit" != "1" ]; then
			echo "WARN: 触发器缺席 $d/$tg（$tbl——定义在册 pg_trigger 无——DROP+CREATE 吞零件同族坑）" >&2
		fi
	done
done < <(grep -hoE "CREATE (OR REPLACE )?TRIGGER [a-z_]+ ON [a-z_0-9]+ [^;]*" migrations/*.sql | sed -E 's/CREATE (OR REPLACE )?TRIGGER //' | sort -u)



# —— 桥域对账闸（T-0223 三列：授权/触发器存在性/模板漂移——哨兵径）
bash "$(dirname "$0")/check_bridge_registry.sh" || fail=1

exit $fail
