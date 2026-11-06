// credscan.go —— MT5 族凭据族发现（app_secrets <broker>.mt5.*/mt5.*
// 键形票面钉死）——密文行枚举+cryptoutil.Open 正径解密，凭据域只读。
package onboard

import (
	"context"
	"fmt"
	"o/internal/platform/cryptoutil"
	"strconv"
	"strings"
)

// ── 凭据发现 ─────────────────────────────────────────────────────────

// credFamily 单 venue 凭据族（fields=明文——内存驻留禁出盘/禁日志回显；
// missing=凭据不在册〔mapped 闸检测用〕；broken=形坏因〔fail-closed〕）。
type credFamily struct {
	venue   string
	fields  map[string]string
	missing bool
	broken  string
}

// discover app_secrets MT5 族凭据枚举（readcred 域 `<broker>.mt5.*` 键形——
// strpos '.mt5.' 族锚+单例 'mt5.' 前缀；IBKR ibkr.paper_* 同款 dotted 命名判例）。
func (r *Runner) discover(ctx context.Context) (map[string]credFamily, error) {
	rows, err := r.fin.Query(ctx, `SELECT key, ciphertext FROM app_secrets
		WHERE key LIKE 'mt5.%' OR strpos(key,'.mt5.')>0`)
	if err != nil {
		return nil, fmt.Errorf("凭据族枚举: %w", err)
	}
	defer rows.Close()
	fams := map[string]credFamily{}
	var cts []struct {
		k  string
		ct []byte
	}
	for rows.Next() {
		var x struct {
			k  string
			ct []byte
		}
		if err := rows.Scan(&x.k, &x.ct); err != nil {
			return nil, err
		}
		cts = append(cts, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var mk []byte
	for _, x := range cts {
		venue, field, ok := credKeySplit(x.k)
		if !ok {
			continue
		}
		fam := fams[venue]
		if fam.fields == nil {
			fam.venue, fam.fields = venue, map[string]string{}
		}
		if mk == nil {
			if mk, err = r.mk(); err != nil {
				return nil, fmt.Errorf("master key: %w", err)
			}
		}
		pt, err := cryptoutil.Open(mk, x.ct)
		if err != nil {
			fam.broken = fmt.Sprintf("%s 解密败: %v", x.k, err)
		} else {
			fam.fields[field] = pt
		}
		fams[venue] = fam
	}
	for v, f := range fams {
		if f.broken == "" {
			fams[v] = checkCredShape(f)
		}
	}
	return fams, nil
}

// discoverVenue 单 venue 凭据族（Advance 干跑面——全枚举后提件）。
func (r *Runner) discoverVenue(ctx context.Context, venue string) (credFamily, error) {
	fams, err := r.discover(ctx)
	if err != nil {
		return credFamily{}, err
	}
	f, ok := fams[venue]
	if !ok {
		return credFamily{}, fmt.Errorf("venue %q 凭据族不在册", venue)
	}
	return f, nil
}

// credKeySplit 键→(venue,field)：单例 mt5.<field>→('mt5',field)；
// <broker>.mt5.<field>→(broker,field)。venue token 形闸（小写数线横——
// 冒号/空白禁入台账键域）。
func credKeySplit(k string) (string, string, bool) {
	if strings.HasPrefix(k, "mt5.") {
		return "mt5", k[4:], true
	}
	v, f, ok := strings.Cut(k, ".mt5.")
	if !ok || v == "" || f == "" {
		return "", "", false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return "", "", false
		}
	}
	return v, f, true
}

// checkCredShape 凭据形校验：login 数字+password+(server|host) 必备
// （server 径优先 ConnectEx；host 逗号多候选径）。
func checkCredShape(f credFamily) credFamily {
	if _, err := strconv.ParseUint(f.fields["login"], 10, 64); err != nil {
		f.broken = fmt.Sprintf("%s.mt5.login 缺/非数字", f.venue)
		return f
	}
	if f.fields["password"] == "" {
		f.broken = fmt.Sprintf("%s.mt5.password 缺/空", f.venue)
		return f
	}
	if f.fields["server"] == "" && f.fields["host"] == "" {
		f.broken = fmt.Sprintf("%s.mt5.server|host 双缺——双轨寻址零件", f.venue)
	}
	return f
}
