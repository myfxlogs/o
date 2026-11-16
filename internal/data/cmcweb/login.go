// login.go —— UI 登录件：dom.PerformSearch 穿透 shadow DOM 拿可见输入件
// (旧 AngularJS 表单 name=username/#password 与新 cmc-login web-component
// closed-shadow 表单通用,SG performSearch 实证 2026-11-16),真鼠标坐标点击
// 聚焦+input.InsertText 填充+按钮文本解析点 Login(Enter 兜底)。
package cmcweb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// fieldBox 页内可点件坐标。
type fieldBox struct{ X, Y float64 }

// UILogin 填表登录——shadow 穿透盘点→坐标点击→InsertText→Login 钮/Enter。
func (b *Browser) UILogin(ctx context.Context, email, pass string) error {
	if b.bctx == nil {
		return fmt.Errorf("browser 未起")
	}
	lctx, cancel := context.WithTimeout(b.bctx, 60*time.Second)
	defer cancel()
	// OneTrust 同意横幅先撤（best-effort——不在场=3s 过）
	bctx, bcancel := context.WithTimeout(lctx, 3*time.Second)
	_ = chromedp.Run(bctx, chromedp.Click(`#onetrust-accept-btn-handler`, chromedp.ByQuery))
	bcancel()

	user, pwd, err := b.loginFields(lctx)
	if err != nil {
		return err
	}
	if err := b.clickType(lctx, user, email); err != nil {
		return fmt.Errorf("username 填: %w", err)
	}
	if err := b.clickType(lctx, pwd, pass); err != nil {
		return fmt.Errorf("password 填: %w", err)
	}
	if btn, ok := b.loginButton(lctx); ok {
		return chromedp.Run(lctx, chromedp.MouseClickXY(btn.X, btn.Y))
	}
	return chromedp.Run(lctx, chromedp.ActionFunc(func(ctx context.Context) error {
		for _, t := range []input.KeyType{input.KeyDown, input.KeyUp} {
			p := input.DispatchKeyEvent(t).
				WithKey("Enter").WithCode("Enter").
				WithWindowsVirtualKeyCode(13)
			if t == input.KeyDown {
				p = p.WithText("\r")
			}
			if err := p.Do(ctx); err != nil {
				return err
			}
		}
		return nil
	}))
}

// loginFields 可见 username/password 输入件盘点（name 锚——新旧表单同享;
// 同名隐藏复件盒=0 滤掉）。
func (b *Browser) loginFields(ctx context.Context) (user, pwd fieldBox, err error) {
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		nodes, err := searchNodes(ctx, "input")
		if err != nil {
			return err
		}
		for _, n := range nodes {
			name, typ := attrOf(n, "name"), strings.ToLower(attrOf(n, "type"))
			cx, cy, ok := boxCenter(ctx, n)
			if !ok {
				continue
			}
			switch {
			case name == "username" && user.X == 0:
				user = fieldBox{cx, cy}
			case typ == "password" && pwd.X == 0:
				pwd = fieldBox{cx, cy}
			}
		}
		return nil
	}))
	if err != nil {
		return
	}
	if user.X == 0 || pwd.X == 0 {
		err = fmt.Errorf("登录输入件未盘点到 (user=%v pwd=%v)", user.X > 0, pwd.X > 0)
	}
	return
}

// loginButton 可见钮中按文本匹 login（innerText/value 解析;影子内件走
// ResolveNode→callFunction 取文本）。
func (b *Browser) loginButton(ctx context.Context) (fieldBox, bool) {
	var hit fieldBox
	_ = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		nodes, err := searchNodes(ctx, "button")
		if err != nil {
			return err
		}
		for _, n := range nodes {
			cx, cy, ok := boxCenter(ctx, n)
			if !ok {
				continue
			}
			if strings.Contains(strings.ToLower(nodeText(ctx, n)), "log") {
				hit = fieldBox{cx, cy}
				return nil
			}
		}
		return nil
	}))
	return hit, hit.X > 0
}

// clickType 坐标真点击聚焦+InsertText（shadow/iframe 免疫——CDP 输入域走
// 合成事件管线,目标=聚焦件不问归属）。
func (b *Browser) clickType(ctx context.Context, f fieldBox, text string) error {
	return chromedp.Run(ctx,
		chromedp.MouseClickXY(f.X, f.Y),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return input.InsertText(text).Do(ctx)
		}),
	)
}

// searchNodes PerformSearch 全量命中（穿 shadow——盘点唯一可靠径）。
func searchNodes(ctx context.Context, q string) ([]*cdp.Node, error) {
	searchID, n, err := dom.PerformSearch(q).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("performSearch %s: %w", q, err)
	}
	ids, err := dom.GetSearchResults(searchID, 0, n).Do(ctx)
	_ = dom.DiscardSearchResults(searchID).Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*cdp.Node, 0, len(ids))
	for _, id := range ids {
		nd, err := dom.DescribeNode().WithNodeID(id).Do(ctx)
		if err == nil {
			out = append(out, nd)
		}
	}
	return out, nil
}

func attrOf(n *cdp.Node, k string) string {
	for i := 0; i+1 < len(n.Attributes); i += 2 {
		if n.Attributes[i] == k {
			return n.Attributes[i+1]
		}
	}
	return ""
}

// boxCenter 盒模型中心（0 尺寸=不可见）。
func boxCenter(ctx context.Context, n *cdp.Node) (float64, float64, bool) {
	box, err := dom.GetBoxModel().WithNodeID(n.NodeID).Do(ctx)
	if err != nil || box == nil || len(box.Content) < 6 {
		return 0, 0, false
	}
	w := box.Content[4] - box.Content[0]
	h := box.Content[5] - box.Content[1]
	if w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return box.Content[0] + w/2, box.Content[1] + h/2, true
}

// nodeText 节点文本（shadow 内件靠 backend node 解析——JS 够不着的也能读）。
func nodeText(ctx context.Context, n *cdp.Node) string {
	remote, err := dom.ResolveNode().WithNodeID(n.NodeID).Do(ctx)
	if err != nil {
		return ""
	}
	defer func() {
		if remote.ObjectID != "" {
			_ = runtime.ReleaseObject(remote.ObjectID).Do(ctx)
		}
	}()
	res, _, err := runtime.CallFunctionOn(`function(){return this.innerText||this.value||this.textContent||''}`).
		WithObjectID(remote.ObjectID).WithReturnByValue(true).Do(ctx)
	if err != nil || res == nil {
		return ""
	}
	return strings.Trim(string(res.Value), `"`)
}
