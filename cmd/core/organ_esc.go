// organ_esc.go —— 器官悬置面探+工位态权威（escLoop 同拍载）：
// 「板上立项而队里无票」元盲区补位——floor 工位册镜像逐件判：态非完工
// ∧ 无开工单载其词 →升件 孤儿账/organ_<名>（「册上有名而队里无票」
// 与孤儿账同族——写了没人干）。实证病灶 2026-11-12：四缺器官立项注记
// 挂屏 2 天无人接——立项显示态≠工作实体，会话工作面只扫升件/工单。
//
// 态权威链（2026-11-12 散文推导第四度翻牌实证根修——账房锚对账信号行/
// D锚裁词括号句,日常词器官散文匹配注定打地鼠）：
//
//	① event='工位' 断言行=态唯一权威（`工位:<名>:<ghost|pin|build|done>`
//	   据#id——会话态变落行,结构化非散文捞）；
//	② 无断言才散文推导（ghostSt 同形分句邻域,信号行 esc_class 非空剔出
//	   ——探针信号非态言）且**自举补断言**——一经断言散文面即退；
//	③ 前端 floorFocus 同改读断言行,散文推导退兜底。
//
// 词册=floorFocus GHOSTS 同表镜像——漂移时探针对自己先照见（词失=
// 全「毛坯」升潮,如实报不抑）。判词归唤知会话——探针只述实,
// 补票/判废/判完工皆会话裁（runbook 孤儿账 organ_* 档）。
package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// organFactWindow 态源窗——超窗无事实=毛坯（90d 无人提=真悬置）。
const organFactWindow = "90 days"

// organTicketWindow 工单覆盖窗（同工单面 30d 口径）。
const organTicketWindow = "30 days"

// organRegistry floor 工位册镜像（名+匹配词——同 GHOSTS.keys 序）。
var organRegistry = []struct {
	name string
	keys []string
}{
	{"B元探", []string{"元探", "自动发探", "缺仪器"}},
	{"C自态", []string{"自态"}},
	{"D目标发起", []string{"目标发起", "abcd", "A-D", "进水口", "intake"}},
	{"旧禁条作废", []string{"禁条", "红线重定义", "redline", "法典"}},
	{"做梦", []string{"做梦"}},
	{"举一反三", []string{"举一反三"}},
	{"连续自我", []string{"连续自我"}},
	{"懂你", []string{"懂你"}},
	{"账房", []string{"账房", "books_daily", "对账"}},
}

// 四态词谱（同 floorFocus DONE_RE/BUILD_RE/PIN_RE——RE2 无 lookahead，
// 「已?销」词界改并吞界符形，判定语义不变）。
var organDoneRe = regexp.MustCompile(
	`销案|部署销|落账毕|法典已部署|已部署|已完成|收口|上线|落地|启用|投产|生效|已?销([：:—→，。、;]|$)|merge|部署至|产线`)
var organBuildRe = regexp.MustCompile(`施工|在施|开工|落码|码修|修码|部署中|处置中|实现|开发中|改造|推进`)
var organPinRe = regexp.MustCompile(`领命|工单|业主令|备记|清单|纳入|认领|立项|排期|消化径`)

// 句界=，。；、：:|（） —— →不算界（工序连接符「认领→修→部署→销」为一句）。
var organClauseL = regexp.MustCompile(`[^，。；;|、：:（）()]*$`)
var organClauseR = regexp.MustCompile(`^[^，。；;|、：:]*`)

// organNear 同分句邻域判（同 floorFocus ghostSt.near——器官名所在分句
// 内找态词，防一注多器官时邻句态词串味）。
func organNear(note string, keys []string, re *regexp.Regexp) bool {
	for _, k := range keys {
		i := strings.Index(note, k)
		if i < 0 {
			continue
		}
		seg := organClauseL.FindString(note[:i])
		after := organClauseR.FindString(note[i+len(k):])
		if re.MatchString(seg + k + after) {
			return true
		}
	}
	return false
}

// organState 推导工位态（同 ghostSt——done>build>pin 优先级,无词兜底
// build=有动过；零事实=ghost 毛坯）。
func organState(note string, hasFact bool, keys []string) string {
	if !hasFact {
		return "ghost"
	}
	if organNear(note, keys, organDoneRe) {
		return "done"
	}
	if organNear(note, keys, organBuildRe) {
		return "build"
	}
	if organNear(note, keys, organPinRe) {
		return "pin"
	}
	return "build"
}

// organAssertRe 工位断言行解析（`工位:<名>:<ghost|pin|build|done>`）。
var organAssertRe = regexp.MustCompile(`工位:([^:：\s]+):(ghost|pin|build|done)`)

// organOrphanEscCheck 器官悬置一探：窗内事实行单程抽出（9 器官共享一
// 扫非逐件一查）→断言权威逐件判态（无断言=散文推导+自举补断言）
// →非完工且无开工单=悬置升件。
// fp=态（态变=新现实重火;态同=同值已阅不暴；票终条件存=销票破
// 纹复验——escEdgeUV 标准语义）。
func (c coreService) organOrphanEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT id,note FROM ops_event
		WHERE event<>'量表' AND COALESCE(esc_class,'')=''
		AND ts>now()-interval '`+organFactWindow+`'
		ORDER BY id DESC`)
	if err != nil {
		return // 读件败本拍让位
	}
	type fact struct {
		id   int64
		note string
	}
	var facts []fact
	for rows.Next() {
		var f fact
		if rows.Scan(&f.id, &f.note) == nil {
			facts = append(facts, f)
		}
	}
	rows.Close()

	// 断言面:工位行=态权威（新行先扫到=最新断言）。
	asserted := map[string]struct {
		st  string
		fid int64
	}{}
	for i := range facts {
		if m := organAssertRe.FindStringSubmatch(facts[i].note); m != nil {
			if _, ok := asserted[m[1]]; !ok {
				asserted[m[1]] = struct {
					st  string
					fid int64
				}{m[2], facts[i].id}
			}
		}
	}

	for _, org := range organRegistry {
		var st string
		var fid int64
		if a, ok := asserted[org.name]; ok {
			st, fid = a.st, a.fid // 断言权威——散文面不再参与
		} else {
			// 无断言=散文推导（同 ghostFact——最新非量表非信号载词行）。
			var hit *fact
			for i := range facts {
				for _, k := range org.keys {
					if strings.Contains(facts[i].note, k) {
						hit = &facts[i]
						break
					}
				}
				if hit != nil {
					break
				}
			}
			st = "ghost"
			if hit != nil {
				fid = hit.id
				st = organState(hit.note, true, org.keys)
			}
			// 自举补断言——推导态落结构化行,此后散文面退场;
			// 写败下拍重试（幂等同值）。
			anch := "无事实行"
			if fid > 0 {
				anch = fmt.Sprintf("据#%d", fid)
			}
			if _, err := c.pool.Exec(ctx,
				`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','工位','',$1)`,
				fmt.Sprintf("工位:%s:%s %s——探针自举(散文推导态;会话态变请落工位断言行)", org.name, st, anch)); err != nil {
				continue // 断言写败本件让位
			}
		}
		if st == "done" {
			continue // 完工件非悬置域
		}
		// 覆盖=OPEN 工单载名或载词（未决口径同工单面——无处置引证）。
		covered := false
		var conds []string
		var args []any
		for i, k := range append([]string{org.name}, org.keys...) {
			args = append(args, "%"+k+"%")
			conds = append(conds, fmt.Sprintf("t.note LIKE $%d", i+1))
		}
		cond := strings.Join(conds, " OR ")
		if err := c.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event t
			WHERE t.event='工单' AND t.ts>now()-interval '`+organTicketWindow+`'
			AND (`+cond+`)
			AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.ts>t.ts AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')))`,
			args...).Scan(&covered); err != nil {
			continue // 读件败本件让位
		}
		org := org
		c.escEdgeUV(ctx, deadSince, "孤儿账", "organ_"+org.name, st, !covered, func() string {
			anch := "无事实行"
			if fid > 0 {
				anch = fmt.Sprintf("据#%d", fid)
			}
			return fmt.Sprintf(
				"孤儿账/organ_%s: 工位态=%s（%s）而 OPEN 工单零载词——板上立项队里无票=悬置器官（补票/判废/判完工归唤知会话裁；词册=floorFocus GHOSTS 镜像）",
				org.name, st, anch)
		})
	}
}
