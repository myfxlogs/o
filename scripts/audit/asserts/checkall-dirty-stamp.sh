#!/usr/bin/env bash
# 审计/部署管道 RED 锚（audit/1112-deploy）：check_all 落戳前须拒脏树——
# 戳键=HEAD sha 而 go test 测面=工作树（已提交面∪porcelain 面），脏树盖戳=
# 「未验证的 HEAD commit」被推+被发（mirror_push→deploy_at_sha 链直通）。
# 正确语义=stamp 写点前 git status --porcelain 非空→拒落（或显式环境键放行）。
# 修好断言自然绿：检出「戳写前存在脏树拒闸」即过。
set -u
F=${1:-scripts/check_all.sh}
[ -f "$F" ] || { echo "assert: check_all.sh 缺席 $F"; exit 2; }

# 戳写点锚（原子发布行 mv → $STAMP）；闸判=戳写点前的「脏树→拒」结构。
# 判词=全文件谱「porcelain 非空→die/fail/exit 且含 脏/dirty/uncommitted 语义」——
# 无此闸=戳可盖脏树 HEAD。
if grep -nE 'porcelain' "$F" | grep -vE '工作树面|WLIST|净路径集' >/dev/null \
   && grep -nE '(git status --porcelain|porcelain)' "$F" \
      | xargs -I{} true 2>/dev/null; then :; fi
DIRTY_GUARD=$(awk '
  /git status --porcelain/ {seen=NR}
  /mv -f .*STAMP|> "\$STAMP"/ && seen {print seen":"NR; exit}
' "$F")
# 更直白判：戳写行前是否对 porcelain 结果做非空拒出（fail/die+非空判）。
GUARD=$(grep -nE 'status --porcelain' "$F" | head -1 | cut -d: -f1)
[ -n "$GUARD" ] || { echo "assert FAIL: check_all.sh 无 porcelain 读面（非本断言域）"; exit 1; }
REJECT=$(awk -v g="$GUARD" 'NR>=g && /porcelain|UNFMT|dirty|未提交|工作树/ && /fail|die|exit 1|die "/{print NR; exit}' "$F")
if [ -z "$REJECT" ]; then
  echo "assert RED: check_all.sh 戳键=HEAD sha 但脏树无拒戳闸——porcelain 仅用于定向谱选包，脏树仍盖 check-green-HEAD（被测面≠被戳面，未验证 commit 可推可发）"
  exit 1
fi
echo "assert GREEN: 戳写前脏树拒闸在位（行 $REJECT）"
