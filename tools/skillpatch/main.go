// skillpatch —— T-0387 canon 自修环裁定席落件机械闸：fin_eng_lesson
// skill_patch 型 pending 的补丁草案→skills/<name>/SKILL.md 落盘。
// 机械五闸（缺一拒写——canon 写域名册钉，矿工/LLM 不直改）：
//
//	① pending 行在册且 status='pending'（裁后行不可再写）；
//	② numbers.target='skill_patch'（型对拍——lesson 件不走本径）；
//	③ numbers.target_skill=--skill 且 skills/<name>/SKILL.md 实存
//	   （名册双钉——pending 钦目标+canon 写域白名单位）；
//	④ evidence_ids 非空（漂移证据锚——无锚补丁拒入，防 LLM 臆改 canon，
//	   票面红线「补丁必须证据锚随行」机械位：落盘节自携证据行非凭撰文）；
//	⑤ lesion_anchor/patch_draft 非空（载荷钦三件——矿工产线齐备）。
//
// 模式：append=新 ## 节追加（补丁增量面）；replace=既有 ## 节正文替换
// （节在案才许换——缺席拒防虚构结构）。verdict 收尾 UPDATE 照旧归
// 裁定席裁流 step4（本件不写 pending 行——落件/判词两拍分权）。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,60}$`)

// payload numbers 机读面（minerPayload skill_patch 三件+锚列同构——
// miner 面私有型不跨包引，对拍靠字段名）。
type payload struct {
	Target       string  `json:"target"`
	TargetSkill  string  `json:"target_skill"`
	LesionAnchor string  `json:"lesion_anchor"`
	PatchDraft   string  `json:"patch_draft"`
	EvidenceIDs  []int64 `json:"evidence_ids"`
}

func main() {
	var (
		skill, mode, section, body, bodyFile, root string
	)
	flag.StringVar(&skill, "skill", "", "目标 skill 名（skills/<name>/）")
	flag.StringVar(&mode, "mode", "append", "append|replace")
	flag.StringVar(&section, "section", "漂移勘正", "目标 ## 节名")
	flag.StringVar(&body, "body", "", "补丁正文（--body-file 二取一）")
	flag.StringVar(&bodyFile, "body-file", "", "补丁正文件路径")
	flag.StringVar(&root, "root", ".", "repo 根（skills/ 上位——测试面）")
	flag.Parse()
	if flag.NArg() != 1 || skill == "" || body == "" && bodyFile == "" {
		fmt.Fprintln(os.Stderr,
			"usage: skillpatch <lesson_id> --skill <name> --mode append|replace [--section <hdr>] (--body <s>|--body-file <f>) [--root <dir>]")
		os.Exit(2)
	}
	if !skillNameRe.MatchString(skill) {
		fmt.Fprintf(os.Stderr, "skill 名非名录形: %q\n", skill)
		os.Exit(2)
	}
	var id int64
	if _, err := fmt.Sscan(flag.Arg(0), &id); err != nil {
		fmt.Fprintf(os.Stderr, "lesson_id 非数: %q\n", flag.Arg(0))
		os.Exit(2)
	}
	text := body
	if bodyFile != "" {
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "body-file 读败: %v\n", err)
			os.Exit(1)
		}
		text = string(b)
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("FIN_PG_DSN"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	out, err := apply(context.Background(), pool, root, id, skill, mode, section, text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skillpatch: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(out)
}

// apply 五闸校验+落盘。返回落件摘要（diff 由 git 面证）。
func apply(ctx context.Context, pool *pgxpool.Pool, root string, id int64,
	skill, mode, section, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("补丁正文空")
	}
	var status, nums string
	if err := pool.QueryRow(ctx,
		`SELECT status,numbers::text FROM fin_eng_lesson WHERE id=$1`, id).
		Scan(&status, &nums); err != nil {
		return "", fmt.Errorf("pending 行读败 id=%d: %w", id, err)
	}
	if status != "pending" {
		return "", fmt.Errorf("闸① 行 %d status=%s 非 pending——裁后行不可再写", id, status)
	}
	var row payload
	if err := json.Unmarshal([]byte(nums), &row); err != nil {
		return "", fmt.Errorf("numbers 非机读: %w", err)
	}
	if row.Target != "skill_patch" {
		return "", fmt.Errorf("闸② 行 %d target=%q 非 skill_patch", id, row.Target)
	}
	if row.TargetSkill != skill {
		return "", fmt.Errorf("闸③ 目标错位——pending 钦 %q 实指 %q",
			row.TargetSkill, skill)
	}
	path := filepath.Join(root, "skills", skill, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("闸③ 名册实存拒——%s: %w", path, err)
	}
	if len(row.EvidenceIDs) == 0 {
		return "", fmt.Errorf("闸④ 行 %d evidence_ids 空集——无证据锚补丁拒入（防臆改 canon）", id)
	}
	if row.LesionAnchor == "" || row.PatchDraft == "" {
		return "", fmt.Errorf("闸⑤ 行 %d lesion_anchor/patch_draft 缺件", id)
	}
	evidence := fmt.Sprintf("> 漂移证据锚=ops_event#%s；病灶锚=%s；源=pending#%d",
		joinIDs(row.EvidenceIDs), row.LesionAnchor, id)
	next, err := patchDoc(string(data), mode, section, evidence+"\n\n"+strings.TrimSpace(text)+"\n")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(next), 0644); err != nil {
		return "", fmt.Errorf("落盘写败: %w", err)
	}
	return fmt.Sprintf("skill_patch 落盘 %s（mode=%s section=%q 证据=ops_event#%s）——"+
		"同拍跑 go run ./tools/kbgen skills 重建索引+git add+commit",
		path, mode, section, joinIDs(row.EvidenceIDs)), nil
}

// patchDoc 节写径：append=文末新 ## 节；replace=既有 ## 节正文整换
// （节缺席拒——防虚构结构；节标题行保留，正文=证据行+补丁文）。
func patchDoc(doc, mode, section, payload string) (string, error) {
	sec := strings.TrimSpace(section)
	if sec == "" || strings.ContainsAny(sec, "\n#") {
		return "", fmt.Errorf("节名非法: %q", section)
	}
	switch mode {
	case "append":
		sep := "\n"
		if !strings.HasSuffix(doc, "\n") {
			sep = "\n\n"
		}
		return doc + sep + "## " + sec + "\n\n" + payload, nil
	case "replace":
		lines := strings.Split(doc, "\n")
		hit := -1
		for i, l := range lines {
			if l == "## "+sec {
				hit = i
				break
			}
		}
		if hit < 0 {
			return "", fmt.Errorf("节 %q 缺席——replace 拒虚构结构（可择 append）", sec)
		}
		end := len(lines)
		for i := hit + 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "## ") {
				end = i
				break
			}
		}
		out := append([]string{}, lines[:hit+1]...)
		out = append(out, "", strings.TrimRight(payload, "\n"), "")
		out = append(out, lines[end:]...)
		return strings.Join(out, "\n"), nil
	default:
		return "", fmt.Errorf("mode 非 append|replace: %q", mode)
	}
}

func joinIDs(ids []int64) string {
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", id)
	}
	return b.String()
}
