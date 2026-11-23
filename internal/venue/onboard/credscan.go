// credscan.go —— venue 凭据族发现（app_secrets 键形票面钉死）——密文行
// 枚举+cryptoutil.Open 正径解密，凭据域只读。
// 方言开集（工单O-35485/#33264 补证复活）：枚举面从 mt5 专制改
// 「<venue>.<dialect>.<field> / <venue>.<field>」通形拆分——
// venueKindSpec 注册表钉各族 schema；非 mt5 族凭据可见（stallTriage
// 「方言已识」判据+族形校验）但不过 MT5 探测径（scan/beat 双闸）。
package onboard

import (
	"context"
	"fmt"
	"o/internal/platform/cryptoutil"
	"strconv"
	"strings"
)

// ── 凭据发现 ─────────────────────────────────────────────────────────

// credFamily 单 venue 凭据族（kind=方言族；fields=明文——内存驻留禁出盘/
// 禁日志回显；missing=凭据不在册〔mapped 闸检测用〕；broken=形坏因
// 〔fail-closed〕）。
type credFamily struct {
	venue   string
	kind    string
	fields  map[string]string
	missing bool
	broken  string
}

// venueKindSpec venue-kind 凭据 schema 注册表（工单O-35485 钦定面）：
// mt5=login/password/server|host；dex-agent=account/agent_key；
// cex=apikey/secret/passphrase（api_key/api_secret 拼写变体同收——
// 产线 okx.* 实形）。fields=必备键名（别名组任一命中即满足）。
type venueKindSpec struct {
	fields [][]string // 每必备件一组——组内任一别名键名命中即满足
}

var venueKindRegistry = map[string]venueKindSpec{
	"mt5":       {fields: [][]string{{"login"}, {"password"}, {"server", "host"}}},
	"dex-agent": {fields: [][]string{{"account"}, {"agent_key"}}},
	"cex":       {fields: [][]string{{"apikey", "api_key"}, {"secret", "api_secret"}, {"passphrase"}}},
}

// kindOfField 2 段键 <venue>.<field> 的 field→kind 反查（注册表字段名
// 全集——先长后短防 'api_key' 被 'key' 类短名误吞；当前无短名冲突）。
func kindOfField(field string) (string, bool) {
	for kind, spec := range venueKindRegistry {
		if kind == "mt5" {
			continue // mt5 族走 .mt5. 方言段不走 2 段 field 反查
		}
		for _, grp := range spec.fields {
			for _, alias := range grp {
				if field == alias {
					return kind, true
				}
			}
		}
	}
	return "", false
}

// discover app_secrets 凭据族枚举（方言开集——mt5 三段/单例形+注册表
// field 名的 2 段 <venue>.<field> 形全收；非注册表键形 credKeySplit
// 拒收不解密）。readcred 域键名枚举、ciphertext 正径解密。
func (r *Runner) discover(ctx context.Context) (map[string]credFamily, error) {
	rows, err := r.fin.Query(ctx, `SELECT key, ciphertext FROM app_secrets
		WHERE key LIKE 'mt5.%' OR strpos(key,'.mt5.')>0
		   OR (length(key)-length(replace(key,'.','')))=1`)
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
		venue, kind, field, ok := credKeySplit(x.k)
		if !ok {
			continue
		}
		fam := fams[venue]
		if fam.fields == nil {
			fam.venue, fam.kind, fam.fields = venue, kind, map[string]string{}
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

// credKeySplit 键→(venue,kind,field)：单例 mt5.<field>→('mt5','mt5',f)；
// <broker>.mt5.<field>→(broker,'mt5',f)；<venue>.<field> 单点键且 field
// 命中注册表→(venue,kindOfField(f),f)。venue token 形闸（小写数线横——
// 冒号/空白禁入台账键域）。
func credKeySplit(k string) (string, string, string, bool) {
	if strings.HasPrefix(k, "mt5.") {
		return "mt5", "mt5", k[4:], true
	}
	if v, f, ok := strings.Cut(k, ".mt5."); ok && v != "" && f != "" {
		if venueTokOK(v) {
			return v, "mt5", f, true
		}
		return "", "", "", false
	}
	if strings.Count(k, ".") == 1 {
		v, f, _ := strings.Cut(k, ".")
		if venueTokOK(v) && f != "" {
			if kind, ok := kindOfField(f); ok {
				return v, kind, f, true
			}
		}
	}
	return "", "", "", false
}

func venueTokOK(v string) bool {
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return v != ""
}

// checkCredShape 凭据形校验（按 kind 走注册表必备件——组内别名任一
// 键名命中即满足；mt5 族沿用 login 数字+password+(server|host) 原判）。
func checkCredShape(f credFamily) credFamily {
	spec, ok := venueKindRegistry[f.kind]
	if !ok {
		f.broken = fmt.Sprintf("%s 未注册 kind=%q", f.venue, f.kind)
		return f
	}
	for _, grp := range spec.fields {
		hit := false
		for _, alias := range grp {
			if f.fields[alias] != "" {
				hit = true
				break
			}
		}
		if !hit {
			f.broken = fmt.Sprintf("%s.%s 凭据缺件（%s 组零命中）",
				f.venue, f.kind, strings.Join(grp, "/"))
			return f
		}
	}
	if f.kind == "mt5" {
		if _, err := strconv.ParseUint(f.fields["login"], 10, 64); err != nil {
			f.broken = fmt.Sprintf("%s.mt5.login 缺/非数字", f.venue)
			return f
		}
	}
	return f
}
