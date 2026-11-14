// draft_blacklist_test.go —— 工单#15188 判据编码：提案起草面黑名单滤
// 不变量。fin_venue_untradeable=venue 拒过标的册（T-0142），起草径
// INSERT fin_proposal 前必核——BTC-USD-270326 黑名单在册 2h 仍被
// basis 径武装开仓致 #158/#159 venue_untradeable 死案。
// 断言形：internal/research 全树凡含 "INSERT INTO fin_proposal" 的
// 源文件，必须在该语句前出现 fin_venue_untradeable 滤——新起草径
// 自动入网（加径不补滤=本测试红）。
package grammar

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDraftSiteBlacklistFilter(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	researchDir := filepath.Dir(filepath.Dir(file)) // internal/research/
	var sites []string
	err := filepath.Walk(researchDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), "INSERT INTO fin_proposal") {
			sites = append(sites, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) == 0 {
		t.Fatal("起草径扫描空——fin_proposal INSERT 面失位须同步本锚")
	}
	for _, p := range sites {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		s := string(src)
		ins := strings.Index(s, "INSERT INTO fin_proposal")
		bl := strings.Index(s, "fin_venue_untradeable")
		if bl < 0 || bl > ins {
			t.Errorf("起草径黑名单滤失位: %s——fin_proposal INSERT 前无 fin_venue_untradeable 核（#15188 死案径回退）",
				filepath.Base(p))
		}
	}
}
