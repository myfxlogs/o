#!/usr/bin/env bash
# check-env-reads.sh —— D-564 防偷懒门禁：venue/凭据域键禁 env 直读（自 nox 誊入改 O）。
# 凭据权威在 exchange_configs 密文列/app_secrets/app_config。三层判定：
#   ① 硬拒（无论白名单）：venue 凭据域键（*_API_KEY/*_API_SECRET/*_PASSPHRASE 等）
#   ② app_secret 域键在 Go 消费面 env 直读 → 拒（app_secrets 轨权威）
#   ③ 其余 env 键 → 必须在 scripts/env-allowlist.txt（ratchet 基线，只缩不增）
# 无 Go 面（internal/cmd/tools 缺席）=条件未备，如实报跳过非谎报已检。
set -u
cd "$(dirname "$0")/.."

ALLOWLIST="scripts/env-allowlist.txt"
[ -f "$ALLOWLIST" ] || { echo "FAIL: $ALLOWLIST 缺失"; exit 1; }

# ① 硬拒模式（venue 凭据域——永不豁免）：
HARD='O_GATE_API_KEY|O_GATE_SECRET|O_GATE_BASE|OKX_API_KEY|OKX_API_SECRET|OKX_API_PASSPHRASE|BINANCE_API_KEY|BINANCE_API_SECRET|GATE_API_KEY|GATE_API_SECRET|GATE_BASE|_PRIVATE_KEY|TRX_VOTE_PRIVATE_KEY'
# ② app_secret 域（Go 消费面拒；脚本/工具面 ratchet 域）：
SECRET='O_AI_|O_FEISHU_APP_SECRET|LONGBRIDGE_APP_KEY|LONGBRIDGE_APP_SECRET|LONGBRIDGE_ACCESS_TOKEN|CMC_API_KEY|O_CMC_API_KEY|O_FRED_API_KEY|FRED_API_KEY|TRONGRID_API_KEY|O_TRON_API_KEY|O_ANT_API_KEY|ANT_CLAW|_ACCESS_TOKEN'
fails=0

# Go 消费面：Getenv/LookupEnv 逐键判定
GO_DIRS=""
for d in internal cmd tools; do [ -d "$d" ] && GO_DIRS="$GO_DIRS $d"; done
if [ -z "$GO_DIRS" ]; then
  echo "check-env-reads: 条件未备跳过（无 Go 面——internal/cmd/tools 均未落位）"
else
  while IFS= read -r f; do
    while IFS= read -r ln; do
      lnno=$(echo "$ln" | cut -d: -f1)
      key=$(echo "$ln" | grep -oE '(os\.Getenv|os\.LookupEnv)\("[^"]+"' | sed -E 's/.*"([^"]+)"/\1/' | head -1)
      [ -z "$key" ] && continue
      if echo "$key" | grep -qE "$HARD"; then
        echo "FAIL: $f:$lnno venue 凭据键 $key env 直读（硬拒——凭据走 exchange_configs 密文列）"
        fails=$((fails+1)); continue
      fi
      if echo "$key" | grep -qE "$SECRET"; then
        if grep -qxF "$key" "$ALLOWLIST"; then
          continue # 存量冻结（ratchet 基线，新键禁入）
        fi
        echo "FAIL: $f:$lnno app_secret 域键 $key Go 面 env 直读（走 app_secrets 轨）"
        fails=$((fails+1)); continue
      fi
      if ! grep -qxF "$key" "$ALLOWLIST"; then
        echo "FAIL: $f:$lnno $key env 直读但不在白名单 $ALLOWLIST（新键须过评审）"
        fails=$((fails+1))
      fi
    done < <(grep -nE 'os\.Getenv\("|os\.LookupEnv\("' "$f" 2>/dev/null)
  done < <(find $GO_DIRS -name '*.go' 2>/dev/null | grep -v _test)
fi

# 脚本/工具面（sh/py 键名字面量扫描，仅硬拒域）：
while IFS= read -r f; do
  while IFS= read -r ln; do
    echo "FAIL: $f:$(echo "$ln" | cut -d: -f1) venue 凭据键字面量（硬拒——走 credread/exchange_configs）"
    fails=$((fails+1))
  done < <(grep -inE "$HARD" "$f" 2>/dev/null | grep -vE '^\s*[0-9]+:\s*#')
done < <(find tools scripts \( -name '*.sh' -o -name '*.py' \) 2>/dev/null | grep -v check-env-reads.sh | grep -v env-allowlist)

# 额外扫描路径（临时文件/目录参数——新增键即时判定）。
for extra in "$@"; do
  while IFS= read -r ln; do
    key=$(echo "$ln" | grep -oE '(os\.Getenv|os\.LookupEnv)\("[^"]+"' | sed -E 's/.*"([^"]+)"/\1/' | head -1)
    [ -z "$key" ] && continue
    if echo "$key" | grep -qE "$HARD"; then
      echo "FAIL: $extra venue 凭据键 $key env 直读（硬拒）"; fails=$((fails+1)); continue
    fi
    if ! grep -qxF "$key" "$ALLOWLIST"; then
      echo "FAIL: $extra $key env 直读但不在白名单（新键须过评审）"; fails=$((fails+1))
    fi
  done < <(grep -nE 'os\.Getenv\("|os\.LookupEnv\("' "$extra" 2>/dev/null)
done

if [ $fails -gt 0 ]; then
  echo "check-env-reads: $fails 处违规（凭据只入 pg——practices「凭据归 pg」规则）"
  exit 1
fi
echo "check-env-reads: OK（硬拒域/app_secret 域零直读；白名单 $(grep -cE '^[A-Z]' "$ALLOWLIST") 键）"
