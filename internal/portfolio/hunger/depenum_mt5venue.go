// depenum_mt5venue.go —— depConfigReaders CFD 户域名册枚举（工单O-41383）：
// 主册 depenum.go 贴 check-lines 300 行棘轮线（depenum_ops.go 拆件先例
// 同形）——CFD venue 键域不再静态双册，改名册 proto='cfd' 全列枚举：
// fin_venue_roster=注册表唯一真源（业主令 2026-11-25），不限 state——
// dep blind 探测面须见 disabled/registered 户的陈会话行（探测语义=
// 「键域有产者写过即面」，停用户键域仍是事实面非死键）。
package hunger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// cfdRosterDomains 名册 proto='cfd' 全列 venue 集（枚举源=fin_venue_roster
// 唯一真源——O-41383 前 xmglobal/exness 静态 init 双册=名册外硬编码，
// 新 CFD 户入册零码自动入探测面；读败=err 透传 fail-closed 同枚举面他读）。
func cfdRosterDomains(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx,
		`SELECT venue FROM fin_venue_roster WHERE proto='cfd'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// appendConfigPairs ④配置对枚举——app_config 键域谱∩读者册（静态册优先，
// 名册 proto='cfd' 户域补位读者="CFD 户会话/订阅面（名册枚举）"；域在册
// 而册外=产而活零读死键 orphan 形升件，册内域=常对留账供谱面审计）。
func (s *Sensor) appendConfigPairs(ctx context.Context, pairs []depPair) ([]depPair, error) {
	domains, err := s.configDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("配置对键域谱读败: %w", err)
	}
	cfdDomains, err := cfdRosterDomains(ctx, s.pool)
	if err != nil {
		return nil, fmt.Errorf("CFD 户域名册枚举读败: %w", err)
	}
	for _, d := range domains {
		reader, ok := depConfigReaders[d]
		if !ok && cfdDomains[d] {
			reader, ok = "CFD 户会话/订阅面（名册枚举）", true
		}
		if ok {
			pairs = append(pairs, depPair{
				sig: "config:" + d + "→" + reader, family: "config",
				producer: d, consumer: reader})
		} else {
			pairs = append(pairs, depPair{
				sig: "config:" + d + "→∅", family: "config",
				producer: d, consumerAbsent: true})
		}
	}
	return pairs, nil
}
