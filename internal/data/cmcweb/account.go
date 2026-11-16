// account.go —— CMC 登录后中间页「Account Selection」处理：多账户户(本户
// Live 38941616 + Demo 38941465)登录后不落主界面,点行才进平台开 WS。
// SG diag 实证 2026-11-16:url=#/accountOptionsSelection,行=div.row.account-open
// (id 属性=accountID),合成 el.click() 不触发 Angular 委托——须 CDP 真鼠标事件。
package cmcweb

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
)

// OnAccountSelection 账户选择页显形探测（hash 界桩——登录中间页唯一锚）。
func (b *Browser) OnAccountSelection(ctx context.Context) bool {
	if b.bctx == nil {
		return false
	}
	var on bool
	ectx, cancel := context.WithTimeout(b.bctx, 10*time.Second)
	defer cancel()
	if err := chromedp.Run(ectx, chromedp.Evaluate(
		`location.hash.includes('accountOption') || location.hash.includes('accountSelect')`,
		&on)); err != nil {
		return false
	}
	return on
}

// SelectAccount 真鼠标点击账户行（行 id 属性=accountID,SG DOM 实证）。
// 一段:点行本体;二段(调用方判仍在选择页后再调):点行尾 ACTIVE 列件。
func (b *Browser) SelectAccount(ctx context.Context, accountID int64) (string, error) {
	if b.bctx == nil {
		return "", fmt.Errorf("browser 未起")
	}
	sel := fmt.Sprintf(`div.row[id="%d"]`, accountID)
	ectx, cancel := context.WithTimeout(b.bctx, 15*time.Second)
	defer cancel()
	var tail string
	if err := chromedp.Run(ectx,
		chromedp.Click(sel, chromedp.ByQuery),
		chromedp.Evaluate(
			`(document.querySelector(`+strconv.Quote(sel)+`)||{outerHTML:''}).outerHTML.slice(-900)`,
			&tail),
	); err != nil {
		return "", fmt.Errorf("selectAccount click %s: %w", sel, err)
	}
	return "row-click tail=" + tail, nil
}
