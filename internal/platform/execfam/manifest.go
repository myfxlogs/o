// manifest.go —— 家族能力声明册机读面（O-41372 family_manifest.tsv 的
// embed 出缝——册文件唯一真源，embed 读同文件零拷贝零漂移；funnel
// census 闸仍对 tsv 本体断言，本面只供运行时消费方读声明）。
package execfam

import (
	_ "embed"
	"strings"
)

//go:embed family_manifest.tsv
var manifestTSV string

// FamilyDecl 声明册单行（六列——family/terminal/alive_check/drill_shape/
// exec_venue/note；note 含空白不限内嵌 TAB，SplitN(…,6) 取尾整列）。
type FamilyDecl struct {
	Family     string
	Terminal   string
	AliveCheck string
	DrillShape string
	ExecVenue  string
	Note       string
}

// FamilyManifest 声明册全量（# 注释/空行免；每调重解析 ~30 行可忽略——
// 册面 build 期冻结语义，运行期不热更）。
func FamilyManifest() []FamilyDecl {
	var out []FamilyDecl
	for _, ln := range strings.Split(manifestTSV, "\n") {
		ln = strings.TrimRight(ln, "\r")
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		f := strings.SplitN(ln, "\t", 6)
		if len(f) < 5 {
			continue
		}
		decl := FamilyDecl{Family: f[0], Terminal: f[1], AliveCheck: f[2],
			DrillShape: f[3], ExecVenue: f[4]}
		if len(f) == 6 {
			decl.Note = f[5]
		}
		out = append(out, decl)
	}
	return out
}
