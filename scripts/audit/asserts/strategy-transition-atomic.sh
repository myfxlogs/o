#!/usr/bin/env bash
# 审计锚 2026-11-21: UPDATE fin_strategy SET state 的迁移+fin_strategy_transition
# 留痕必须同事务(禁两拍裸 Exec——中间崩=无痕迁移,verdict-artifact-ordering-seam 判例)。
# 判法: lifecycle.Transition 内联 tx 属合法;机器外裸 UPDATE 须近窗 ±8 行内见 tx.Exec(
# 同 tx 形)且 INSERT transition 与 UPDATE 同 tx 句柄。
set -u
cd "$(git rev-parse --show-toplevel)"
bad=0
for f in $(grep -rln "UPDATE fin_strategy SET state" --include="*.go" internal/ cmd/ | grep -v _test); do
  # 每处 UPDATE 行取上下文:近 8 行窗内须同句柄 tx 写 transition 或同函数内已有 tx.Exec INSERT transition
  awk -v file="$f" '
    /UPDATE fin_strategy SET state/ {
      # 回溯近 12 行窗找 tx.Exec INSERT INTO fin_strategy_transition
      win=""; for (i=NR-12;i<NR;i++) win=win" "buf[i]
      if (win !~ /tx\.Exec.*fin_strategy_transition/ && win !~ /fin_strategy_transition/) {
        # 后向看 12 行
        getline_ok=1; ahead=""
        while ((getline line) > 0 && ahead_n<12) { ahead=ahead" "line; ahead_n++ }
        if (ahead !~ /fin_strategy_transition/) {
          printf "%s:%d 裸 UPDATE 无 tx transition 留痕\n", file, NR; bad=1
        } else if (ahead !~ /tx\.Exec/) {
          printf "%s:%d transition 留痕非 tx 句柄(疑似两拍裸 Exec,非原子)\n", file, NR; bad=1
        }
        ahead_n=0
      }
    }
    { buf[NR]=$0 }
    END { exit bad }
  ' "$f" || bad=1
done
exit $bad
