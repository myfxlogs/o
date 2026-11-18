// param_intake.go —— 裁定参数包进气道（工单#23296，母机制#21605）：
// DM 写径拍点扫 ops_event 参数请求面（自我令票/唤知裁定行）→DMSet 写
// app_config。病灶实证=#19797：参数值钦定包（行20313 INSERT/UPDATE 谓词组）
// 躺票面数周不可达——裁定参数永不到 DM=参数治理死，堵 calib 与
// funding-hedge 复开全链。修法①钦定：复用既有行格式零新协议——请求行
// 载 `INSERT|UPDATE <全限定键>=<值>` 谓词组即机械可解；写径唯一=DMSet
// （域闸 fail-closed+参数调整三列账行原生，直写 SQL=违例）。
// 进气道边界=机械可解面：全限定键+显式写动词才落；简写形（maker_fee_bps
// =0.58 裸键歧义）/散文提及不解析——歧义归人裁不臆写（手不是脑）。
// 水位=ops_state_row(kind='参数进气',scope='watermark')——机制态非参数
// 不入 app_config 册（check_config_write_domain 覆盖棘轮不受触）；
// 变了才更新=进气道自状态行（自我席每拍可读——透明非私记）。
// 自喂防=event 白名单不含 参数调整+seat 排决策席-DM写径 双闸——DMSet
// 自产审计行永不入扫面（append-only 账不回流成请求）。
package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"o/internal/platform/config"
)

// paramIntakeEvery 进气道拍周期（var=测试面覆写锚——生产值钦定不动）。
// 参数投递非急症（裁定→落键分钟级足够），5min 兼顾响应与扫面成本。
var paramIntakeEvery = 5 * time.Minute

// paramIntakeBatch 单拍扫描上限（参数请求行史上稀——200 足余量；超限下拍续）。
const paramIntakeBatch = 200

// intakeStateKind/Scope 水位行坐标（ops_state_row——机制态非参数册键）。
const (
	intakeStateKind  = "参数进气"
	intakeStateScope = "watermark"
)

// paramSpec 一条参数写规格（解析产物——动词留证据串，INSERT/UPDATE 写径
// 同=upsert 不分语义）。
type paramSpec struct {
	verb string
	key  string
	raw  string
}

// paramSpecRE 写动词谓词组：`INSERT|UPDATE <键>=<值>`——键限
// [A-Za-z0-9_.]（全限定点分域），值贴等号且截 [空白,;+()（）;；:：] 止（=后空格=散文歧义归人不解析）
// （行20313 实证形：0.58(实测…)/0(friction…)/+分隔）。
var paramSpecRE = regexp.MustCompile(
	`(?i)\b(INSERT|UPDATE)\s+([A-Za-z0-9_.]+)\s*=([^\s,;+()（）;；:：]+)`)

// parseParamSpecs note 文本→写规格列（机械解析零判词——散文不臆解）。
func parseParamSpecs(note string) []paramSpec {
	ms := paramSpecRE.FindAllStringSubmatch(note, -1)
	if len(ms) == 0 {
		return nil
	}
	out := make([]paramSpec, 0, len(ms))
	for _, m := range ms {
		out = append(out, paramSpec{verb: m[1], key: m[2], raw: m[3]})
	}
	return out
}

// intakeJSONValue '=' 右值原文→jsonb 标量。JSON 原生形（数字/布尔/串）
// 原样过；裸记号（resolved/abc 类）转 JSON 串——validScalar 契约守在
// DMSet 内再钉一遍（null/结构形此处先拒不送闸）。
func intakeJSONValue(raw string) (json.RawMessage, error) {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		switch v.(type) {
		case string, float64, bool:
			return json.RawMessage(raw), nil
		}
		return nil, fmt.Errorf("非标量形（%T）", v)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("值编码败: %w", err)
	}
	return b, nil
}

// paramIntakeLoop 进气道拍（minerLoop 同构——ctx 界内协程，拉起即首扫：
// 部署即补存量参数包非干等一拍——#19797 积压正是本件要通的径）。
func (r *Runner) paramIntakeLoop(ctx context.Context) {
	t := time.NewTicker(paramIntakeEvery) // sched-exempt-ok 慢拍记账面（事件账进气非事件径——水位幂等重扫无害）
	defer t.Stop()
	r.paramIntakeTick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.paramIntakeTick(ctx)
		}
	}
}

// paramIntakeTick 单拍：水位后参数请求面→逐规格域闸+DMSet。
// 失败纪律=水位停于败行前下拍重试（瞬态败可愈）；域外/不可解规格跳过不阻
// 行（行20313 形含维持散文与他域键——跳过是常态非异常）。
func (r *Runner) paramIntakeTick(ctx context.Context) {
	wm, err := r.intakeWatermark(ctx)
	if err != nil {
		log.Printf("[lifecycle] paramintake 水位读败: %v", err)
		return
	}
	rows, err := r.pool.Query(ctx, `SELECT id,event,seat,ticket,note FROM ops_event
		WHERE id>$1
		  AND event IN ('处置','裁定','条件请求','令','业主令','管理令','优先令','闸值钦定','口径','判词','裁决')
		  AND seat<>'决策席-DM写径'
		  AND note ~* '(insert|update)[[:space:]]+[a-z0-9_.]+[[:space:]]*='
		ORDER BY id LIMIT $2`, wm, paramIntakeBatch)
	if err != nil {
		log.Printf("[lifecycle] paramintake 扫面败: %v", err)
		return
	}
	type cand struct {
		id     int64
		event  string
		seat   string
		ticket string
		note   string
	}
	var cs []cand
	for rows.Next() {
		var x cand
		if err := rows.Scan(&x.id, &x.event, &x.seat, &x.ticket, &x.note); err != nil {
			rows.Close()
			log.Printf("[lifecycle] paramintake 行扫败: %v", err)
			return
		}
		cs = append(cs, x)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("[lifecycle] paramintake 行集败: %v", err)
		return
	}
	rows.Close()
	if len(cs) == paramIntakeBatch {
		log.Printf("[lifecycle] paramintake 批满 %d 行——余量下拍续扫", paramIntakeBatch)
	}
	applied, skipped := 0, 0
	lastID := wm
	for _, x := range cs {
		specs := parseParamSpecs(x.note)
		if len(specs) == 0 {
			lastID = x.id // 预筛命中而解析零规格=非参数请求行——过水位不阻塞
			continue
		}
		for _, sp := range specs {
			if config.ClassifyKey(sp.key) != config.DomainDM {
				skipped++ // 域外键（owner/未登记/凭证形）——闸拒语义前移，行内余规格照走
				continue
			}
			val, err := intakeJSONValue(sp.raw)
			if err != nil {
				skipped++
				continue
			}
			ev := fmt.Sprintf("进气道 ops_event#%d（%s/%s %s %s）",
				x.id, x.seat, x.event, strings.ToUpper(sp.verb), sp.key)
			tk := x.ticket
			if tk == "" {
				tk = "dm-intake"
			}
			if _, err := r.cfg.DMSet(ctx, sp.key, val, ev, tk); err != nil {
				// 写败=本拍让位——水位停于本行前（含已落规格的行也停：
				// 行内序重放靠 DMSet 幂等 upsert+新账行，宁重勿丢）。
				log.Printf("[lifecycle] paramintake #%d %s=%s 落键败（本拍让位下拍重试）: %v",
					x.id, sp.key, sp.raw, err)
				r.intakeSetWatermark(ctx, lastID)
				return
			}
			applied++
		}
		lastID = x.id
	}
	if lastID > wm {
		r.intakeSetWatermark(ctx, lastID)
	}
	if applied > 0 || skipped > 0 {
		log.Printf("[lifecycle] paramintake 水位 %d→%d：落键 %d 拒/跳 %d",
			wm, lastID, applied, skipped)
	}
}

// intakeWatermark 进气道水位读（无行=0——全史首扫：存量参数包正是本件
// 要通的径，#19797 行20313 即首扫拾件；事件白名单+动词预筛使全史扫面
// 仅命中真参数请求行）。
func (r *Runner) intakeWatermark(ctx context.Context) (int64, error) {
	var s string
	err := r.pool.QueryRow(ctx, `SELECT detail FROM ops_state_row
		WHERE kind=$1 AND scope=$2`, intakeStateKind, intakeStateScope).Scan(&s)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("水位行 detail=%q 非整数: %w", s, err)
	}
	return id, nil
}

// intakeSetWatermark 水位行 upsert（变了才更新——detail=裸 id 供读面解析；
// 写败仅记日志不阻拍——水位滞后=下拍重扫幂等无害）。
func (r *Runner) intakeSetWatermark(ctx context.Context, id int64) {
	if _, err := r.pool.Exec(ctx, `INSERT INTO ops_state_row(kind,scope,valence,detail)
		VALUES($1,$2,10,$3)
		ON CONFLICT(kind,scope) DO UPDATE SET detail=EXCLUDED.detail,
			updated_at=CASE WHEN ops_state_row.detail IS DISTINCT FROM EXCLUDED.detail
				THEN now() ELSE ops_state_row.updated_at END`,
		intakeStateKind, intakeStateScope, strconv.FormatInt(id, 10)); err != nil {
		log.Printf("[lifecycle] paramintake 水位写败（下拍重扫幂等）: %v", err)
	}
}
