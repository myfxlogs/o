#!/usr/bin/env bash
# 审计/部署管道 RED 锚（audit/1115-deploy2）：deploy/*.service 的 ExecStart
# 指向仓内脚本径者必须在部署树实存（在册≠实存=死径——o-disk-hygiene
# 钦定 30min worktree 保洁拍 ExecStart=/opt/o/scripts/meta/wt_gc.sh，该件
# 在纪元3清扫 07835f766 整目录删除且无迁址替身=启用即每拍炸+T-0466 孤儿
# 锁保洁面整体蒸发）。正确语义=仓内 ExecStart 目标实存或 unit 件同步退役；
# 修好断言自然绿。外部二进制/绝对构件径（/opt/o-collecto 等）豁免——非仓件。
set -u
cd "$(dirname "$0")/../../.." || exit 2
MISS=0
while IFS= read -r svc; do
	target=$(grep -oE 'ExecStart=[^ ]+' "$svc" | head -1 | cut -d= -f2-)
	case "$target" in
		/opt/o/*) rel="${target#/opt/o/}" ;;
		*) continue ;; # 非仓内径豁免
	esac
	if [ ! -f "$rel" ]; then
		echo "assert RED: $svc ExecStart=$target——仓内目标 $rel 缺件（部署树注册 unit 指死径：启用必炸/功能蒸发静默）"
		MISS=1
	fi
done < <(find deploy -name '*.service')
[ "$MISS" = 0 ] && { echo "assert GREEN: deploy service 仓内 ExecStart 目标全实存"; exit 0; }
exit 1
