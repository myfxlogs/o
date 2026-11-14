// reflex_esc.go —— canon→反射编译器（棒4「经验→闸」器官件）:
// 判词不靠会话记得——裁定记号机械物化 fin_econ_rule 闸,反射回执落账。
// 记号两路:
//
//	①一等裁定行 event='裁定' note='admit_block <match json>'∨
//	  '解禁 admit_block <match json>'——每行一编,1:1 落「反射」回执
//	  (非法形也回执=拒,不吞裁定);
//	②处置行记号 lethal:<tpl>∨放行:<tpl>(runbook 契约——lethal 原喂
//	  种群榜,本件放大成 admission 闸;放行=解禁同义)。
//
// 序:同 match 全源记号按 id 最新胜(裁定∨处置记号同权)——陈记号
// 不许复活已被后件解禁的规约;状态差分即物化,变更才落反射行。
// 自证:裁定行 >1h 无「反射」行引证=编译器停→升件 反射/reflex_stall。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// reflexTokenRe 处置行内 lethal:<tpl>∨放行:<tpl> 记号。
var reflexTokenRe = regexp.MustCompile(`(lethal|放行):([A-Za-z0-9_-]+)`)

// reflexMatchKeys 与 lifecycle econrule 闭合同契——越集键=非法裁定拒收。
var reflexMatchKeys = map[string]bool{"template_ref": true, "inst_id": true, "direction": true}

type reflexVerdict struct {
	id    int64
	op    bool   // true=激活,false=解禁
	canon string // 规范化 match json(map 序列化键序定——天然范式)
	src   string // 反射行引证源
}

func (c coreService) reflexCompileCheck(ctx context.Context, deadSince map[string]time.Time) {
	verdicts := map[string]reflexVerdict{}
	pending := map[int64]struct {
		op    bool
		canon string
		err   error
	}{}
	cited := map[int64]bool{} // 本拍物化已引证的裁定 id(免回执双落)

	// ①裁定行全量扫:parse→按 canon 取 id 最新胜;无反射引证者入 pending
	rows, err := c.pool.Query(ctx, `SELECT e.id, e.note,
		EXISTS(SELECT 1 FROM ops_event r WHERE r.event='反射'
			AND r.note ~ ('裁定#'||e.id||'([^0-9]|$)')) AS cited
		FROM ops_event e WHERE e.event='裁定' ORDER BY e.id`)
	if err != nil {
		return
	}
	for rows.Next() {
		var id int64
		var note string
		var wasCited bool
		if rows.Scan(&id, &note, &wasCited) != nil {
			continue
		}
		op, match, perr := parseReflexNote(note)
		if !wasCited {
			p := pending[id]
			p.op, p.err = op, perr
			if match != nil {
				p.canon = canonMatch(match)
			}
			pending[id] = p
		}
		if perr != nil {
			continue
		}
		canon := canonMatch(match)
		if id > verdicts[canon].id {
			verdicts[canon] = reflexVerdict{id, op, canon, fmt.Sprintf("裁定#%d", id)}
		}
	}
	rows.Close()

	// ②处置记号:同 canon 仅当行 id>现存胜件才替——陈记号不复活已解禁规
	rows, err = c.pool.Query(ctx, `SELECT id,note FROM ops_event WHERE event='处置'
		AND note ~ '(lethal|放行):[A-Za-z0-9_-]+' ORDER BY id`)
	if err == nil {
		for rows.Next() {
			var id int64
			var note string
			if rows.Scan(&id, &note) != nil {
				continue
			}
			for _, m := range reflexTokenRe.FindAllStringSubmatch(note, -1) {
				canon := canonMatch(map[string]string{"template_ref": m[2]})
				op := m[1] == "lethal"
				if id > verdicts[canon].id {
					verdicts[canon] = reflexVerdict{id, op, canon,
						fmt.Sprintf("处置#%d %s:%s", id, m[1], m[2])}
				}
			}
		}
		rows.Close()
	}

	// ③物化:规则态≠欲态→upsert+反射行(未变不书)
	for canon, v := range verdicts {
		var cur *bool
		var active bool
		if err := c.pool.QueryRow(ctx, `SELECT active FROM fin_econ_rule
			WHERE kind='admit_block' AND match=$1::jsonb`, canon).
			Scan(&active); err == nil {
			cur = &active
		}
		if cur != nil && *cur == v.op {
			continue
		}
		var rid int64
		if err := c.pool.QueryRow(ctx, `INSERT INTO fin_econ_rule(kind,match,source,note,active)
			VALUES('admit_block',$1::jsonb,$2,$3,$4)
			ON CONFLICT (kind,match) DO UPDATE SET active=$4,source=$2,note=$3
			RETURNING id`, canon, v.src, v.src, v.op).Scan(&rid); err != nil {
			continue // 写败=下拍重编,stall 探兜「裁定不编译」面
		}
		c.reflexEmit(ctx, fmt.Sprintf("%s→rule#%d admit_block%s active=%v",
			v.src, rid, canon, v.op))
		var srcID int64
		if n, _ := fmt.Sscanf(v.src, "裁定#%d", &srcID); n > 0 {
			cited[srcID] = true
		}
	}

	// ④裁定回执:未被物化引证者 1:1 补(非法形=拒回执非吞)
	for id, p := range pending {
		if cited[id] {
			continue
		}
		if p.err != nil {
			c.reflexEmit(ctx, fmt.Sprintf("裁定#%d→拒:%v", id, p.err))
			continue
		}
		op := "active"
		if !p.op {
			op = "解禁"
		}
		c.reflexEmit(ctx, fmt.Sprintf("裁定#%d→编 admit_block%s %s(后件胜∨态本然)",
			id, p.canon, op))
	}

	// ⑤自证:裁定行>1h 无反射引证=编译器停
	var uncompiled int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='裁定' AND e.ts<now()-interval '1 hour'
		AND NOT EXISTS(SELECT 1 FROM ops_event r WHERE r.event='反射'
			AND r.note ~ ('裁定#'||e.id||'([^0-9]|$)'))`).Scan(&uncompiled); err != nil {
		return
	}
	c.escEdgeUV(ctx, deadSince, "反射", "reflex_stall", "@", uncompiled > 0, func() string {
		return fmt.Sprintf("反射编译器停摆——%d 件裁定行>1h 未编译(规约物化中断:裁定记号不生效,教训无法变闸;修编译器∨清堵行)", uncompiled)
	})
}

func (c coreService) reflexEmit(ctx context.Context, note string) {
	c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','反射','reflex',$1)`, note)
}

// parseReflexNote 裁定行 note 解:'admit_block <match json>'∨
// '解禁 admit_block <match json>';match 键越闭合集∨空值=非法拒收。
func parseReflexNote(note string) (bool, map[string]string, error) {
	note = strings.TrimSpace(note)
	op := true
	if rest, ok := strings.CutPrefix(note, "解禁 "); ok {
		op, note = false, strings.TrimSpace(rest)
	}
	js, ok := strings.CutPrefix(note, "admit_block ")
	if !ok {
		return false, nil, fmt.Errorf("形非法(欲「admit_block <json>」∨「解禁 admit_block <json>」)")
	}
	var match map[string]string
	if err := json.Unmarshal([]byte(js), &match); err != nil {
		return false, nil, fmt.Errorf("match json 不可解")
	}
	if len(match) == 0 {
		return false, nil, fmt.Errorf("match 空(全放行语义不可表)")
	}
	for k, v := range match {
		if !reflexMatchKeys[k] {
			return false, nil, fmt.Errorf("match 键 %q 越闭合集", k)
		}
		if v == "" {
			return false, nil, fmt.Errorf("match 键 %q 值空", k)
		}
	}
	return op, match, nil
}

// canonMatch 规范化 match json——map 序列化键序定,同义异序同键同 canon。
func canonMatch(match map[string]string) string {
	b, err := json.Marshal(match)
	if err != nil {
		return ""
	}
	return string(b)
}
