#!/usr/bin/env bash
# t0370_gate_hook_test.sh —— T-0370 双件注入实证：
# ① post-commit 收口词面收紧——桩 commit 各形 subject 干跑钩件，
#    名词连缀（收口支SHA/收口待审/收口钩/收口戳）零误中，动作词位
#    （收口——/收口+/收口结尾）照中；
# ② check_master_direct.sh first-parent 闸——施工直写 master 拒、
#    merge 第二亲合支放行、非施工直写放行、对账区间不可解 fail-closed。
set -euo pipefail
ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf -- "$TMP"' EXIT

# ══ ① post-commit 词面矩阵 ══
REPO="$TMP/hookrepo"; mkdir -p "$REPO" "$TMP/home/bin"
git -C "$REPO" init -q
git -C "$REPO" config user.email t@t && git -C "$REPO" config user.name t
cat > "$TMP/home/bin/report.sh" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$TMP/report-log"
EOF
chmod +x "$TMP/home/bin/report.sh"
: > "$TMP/report-log"

fire() { git -C "$REPO" commit -q --allow-empty -m "$1"; (cd "$REPO" && HOME="$TMP/home" bash "$ROOT/scripts/hooks/post-commit"); }
nofire() { fire "$1"; }

# 动作词位形——应触发
fire '[角色:施工][T-0370] 收口——测试件A'
fire '[角色:施工][T-0370] 收口+T-0307 返修②——测试件B'
fire '[角色:施工][T-0370] 三件套收口'
# 名词连缀/他角色/无票——应不触发
nofire '[角色:施工][T-0370] 票面回填——领取行+收口支SHA'
nofire '[角色:施工][T-0370] 返修①票面回填——状态翻返修①收口待审+结果回填节'
nofire '[角色:施工][T-0370] 收口钩活干跑实证（空commit）'
nofire '[角色:施工][T-0370] 结果回填+翻待验——重飞三件对账+收口戳'
nofire '[角色:施工][T-0370] 返修①——票面回填+收口自报钩机制级解'
nofire '[角色:决策][T-0370] 裁决通过——合master(收口支tip abc1234)'
nofire '[角色:施工] 无票号 收口——不应有TID'

FIRE_N=$(grep -c '收口 T-' "$TMP/report-log" || true) # tolerate: grep 空集=零命中合法径
[[ "$FIRE_N" -eq 3 ]] && echo "① post-commit 词面：3 触发/6 不触 PASS" \
	|| { echo "FAIL fire_n=$FIRE_N"; cat "$TMP/report-log"; exit 1; }
grep -q '收口 T-0370' "$TMP/report-log"

# ══ ② master_direct_gate first-parent 判法 ══
G="$TMP/gate"; mkdir -p "$G"; git -C "$G" init -q -b master
git -C "$G" config user.email t@t && git -C "$G" config user.name t
GATE="$ROOT/scripts/check_master_direct.sh"

# 场景1：施工 commit 直写 master first-parent → 拒
B0=$(git -C "$G" commit-tree $(git -C "$G" mktree </dev/null) -m 'base')
git -C "$G" update-ref refs/heads/master "$B0"
git -C "$G" checkout -q master
C1=$(git -C "$G" commit-tree $(git -C "$G" mktree </dev/null) -p "$B0" -m '[角色:施工][T-0370] 直写部署树件')
git -C "$G" update-ref refs/heads/master "$C1"
if (cd "$G" && "$GATE" "$C1" "$B0") 2>/dev/null; then echo "FAIL 场景1 未拒"; exit 1; fi
echo "②a 施工直写 master → 拒 PASS"

# 场景2：施工 commit 经 --no-ff merge 入 master（第二亲）→ 放行
# （master 先回卷 B0——隔离场景1 的直写件不污染本场景 first-parent 链）
git -C "$G" update-ref refs/heads/master "$B0"
git -C "$G" checkout -q -b work "$B0"
C2=$(git -C "$G" commit-tree $(git -C "$G" mktree </dev/null) -p "$B0" -m '[角色:施工][T-0370] 票支施工件')
git -C "$G" update-ref refs/heads/work "$C2"
git -C "$G" checkout -q master
MC=$(git -C "$G" commit-tree $(git -C "$G" mktree </dev/null) -p "$B0" -p "$C2" -m '[角色:决策][T-0370] 裁决合master')
git -C "$G" update-ref refs/heads/master "$MC"
(cd "$G" && "$GATE" "$MC" "$B0") && echo "②b 合支第二亲施工件 → 放行 PASS" \
	|| { echo "FAIL 场景2 误拒"; exit 1; }

# 场景3：非施工前缀直写 → 放行
C3=$(git -C "$G" commit-tree $(git -C "$G" mktree </dev/null) -p "$MC" -m '[角色:决策][T-0370] 裁决直写件')
git -C "$G" update-ref refs/heads/master "$C3"
(cd "$G" && "$GATE" "$C3" "$MC") && echo "②c 非施工直写 → 放行 PASS" \
	|| { echo "FAIL 场景3 误拒"; exit 1; }

# 场景4：remote_sha 本地缺件=对账不可解 → fail-closed 拒
if (cd "$G" && "$GATE" "$C3" "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef") 2>/dev/null; then
	echo "FAIL 场景4 不可解放行"; exit 1
fi
echo "②d 对账区间不可解 → fail-closed 拒 PASS"

# 场景5：全零 sha（删 ref/新 ref）→ 跳过放行
(cd "$G" && "$GATE" "0000000000000000000000000000000000000000" "$B0") \
	&& (cd "$G" && "$GATE" "$C3" "0000000000000000000000000000000000000000") \
	&& echo "②e 全零 sha 跳过 PASS" || { echo "FAIL 场景5"; exit 1; }

echo "t0370 master-direct gate + post-commit hook precision PASS"
