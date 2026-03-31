package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"
	"sort"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

// PredictTagService 标签统计服务（轻量，不落库）。
var PredictTagService = newPredictTagService()

func newPredictTagService() *predictTagService {
	return &predictTagService{}
}

type predictTagService struct{}

type HotTag struct {
	Tag  string `json:"tag"`
	Heat int64  `json:"heat"`
}

// TopN 将 tag->heat 的 map 输出为按 heat 倒序的 TopN。
// 当 heat 相同，按 tag 字典序升序稳定排序。
func (s *predictTagService) TopN(tagHeat map[string]int64, n int) []HotTag {
	if n <= 0 {
		return []HotTag{}
	}
	list := make([]HotTag, 0, len(tagHeat))
	for tag, heat := range tagHeat {
		list = append(list, HotTag{Tag: tag, Heat: heat})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Heat == list[j].Heat {
			return list[i].Tag < list[j].Tag
		}
		return list[i].Heat > list[j].Heat
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

// RefreshTagsFromContexts 将 t_predict_context.tags（逗号分隔 slug）物化到 t_predict_tag，并生成/更新统计表 t_predict_tag_stat。
// 设计目标：线上查询标签列表不再全表扫描/拆分 tags，从而保证高性能。
//
// 策略：全量刷新（简单可靠，适合标签量不大/刷新频率较低）。
// - 读取 PredictContext，仅取 market_id/tags/update_time
// - 拆分成 tag->(lastSeenAt, marketIdSet)
// - upsert PredictTag（按 slug）
// - upsert PredictTagStat（按 tagId）
//
// 说明：若未来数据量很大，可改为增量刷新（按 PredictContext.UpdateTime watermark）。
func (s *predictTagService) RefreshTagsFromContexts() error {
	return s.refreshTagsFromContexts(sqls.DB())
}

func (s *predictTagService) refreshTagsFromContexts(db *gorm.DB) error {
	if db == nil {
		return errors.New("db is nil")
	}

	// 只取必要字段
	type row struct {
		MarketId    int64
		Tags        string
		UpdateTime  int64
		CreateTime  int64
	}
	var rows []row
	if err := db.Model(&models.PredictContext{}).
		Select("market_id, tags, update_time, create_time").
		Where("tags <> ''").
		Find(&rows).Error; err != nil {
		return err
	}

	// 聚合：tag -> lastSeenAt + marketId 去重
	type agg struct {
		LastSeenAt int64
		Markets    map[int64]struct{}
	}
	aggMap := map[string]*agg{}
	for _, r := range rows {
		tags := strings.TrimSpace(r.Tags)
		if tags == "" {
			continue
		}
		seenAt := r.UpdateTime
		if seenAt <= 0 {
			seenAt = r.CreateTime
		}
		parts := strings.Split(tags, ",")
		for _, p := range parts {
			slug := strings.ToLower(strings.TrimSpace(p))
			if slug == "" {
				continue
			}
			a, ok := aggMap[slug]
			if !ok {
				a = &agg{LastSeenAt: seenAt, Markets: map[int64]struct{}{}}
				aggMap[slug] = a
			}
			if seenAt > a.LastSeenAt {
				a.LastSeenAt = seenAt
			}
			a.Markets[r.MarketId] = struct{}{}
		}
	}

	now := dates.NowTimestamp()
	return db.Transaction(func(tx *gorm.DB) error {
		for slug, a := range aggMap {
			// upsert tag
			t := repositories.PredictTagRepository.Take(tx, "slug = ?", slug)
			if t == nil {
				t = &models.PredictTag{
					Slug:       slug,
					Name:       slug,
					LastSeenAt: a.LastSeenAt,
					CreateTime: now,
					UpdateTime: now,
				}
				if err := repositories.PredictTagRepository.Create(tx, t); err != nil {
					return err
				}
			} else {
				updates := map[string]any{
					"last_seen_at": a.LastSeenAt,
					"update_time":  now,
				}
				// name：若当前 name 为空，补齐为 slug
				if strings.TrimSpace(t.Name) == "" {
					updates["name"] = slug
				}
				if err := repositories.PredictTagRepository.UpsertBySlug(tx, slug, updates); err != nil {
					return err
				}
				// 重新拿一次，确保有 id（以及更新后的 name）
				t = repositories.PredictTagRepository.Take(tx, "slug = ?", slug)
				if t == nil {
					return errors.New("predict tag upsert failed")
				}
			}

			// upsert stat
			mc := int64(len(a.Markets))
			st := repositories.PredictTagStatRepository.TakeByTagId(tx, t.Id)
			if st == nil {
				st = &models.PredictTagStat{
					TagId:       t.Id,
					MarketCount: mc,
					RefreshedAt: now,
					CreateTime:  now,
					UpdateTime:  now,
				}
				if err := repositories.PredictTagStatRepository.Create(tx, st); err != nil {
					return err
				}
			} else {
				st.MarketCount = mc
				st.RefreshedAt = now
				st.UpdateTime = now
				if err := repositories.PredictTagStatRepository.Update(tx, st); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
