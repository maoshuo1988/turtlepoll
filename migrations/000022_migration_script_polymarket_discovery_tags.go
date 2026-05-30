package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm/clause"
)

func migrate_polymarket_discovery_tags() error {
	db := sqls.DB()
	if err := db.AutoMigrate(&models.PolymarketDiscoveryTag{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(&models.PredictTag{}); err != nil {
		return err
	}

	now := dates.NowTimestamp() / 1000
	rows := []models.PolymarketDiscoveryTag{
		{Rank: 1, Slug: "politics", Name: "Politics", ExternalTagId: 2, Description: "政治类事件", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 2, Slug: "crypto", Name: "Crypto", ExternalTagId: 21, Description: "加密货币相关", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 3, Slug: "sports-games", Name: "Sports / Games", ExternalTagId: 100639, Description: "体育赛事（单场比赛，非赛季期货）", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 4, Slug: "technology-tech", Name: "Technology / Tech", ExternalTagId: 1401, Description: "科技类事件", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 5, Slug: "culture-entertainment", Name: "Culture / Entertainment", ExternalTagId: 596, Description: "娱乐/文化类预测", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 6, Slug: "finance", Name: "Finance", ExternalTagId: 120, Description: "金融类事件", Enabled: true, CreateTime: now, UpdateTime: now},
		{Rank: 7, Slug: "geopolitics", Name: "Geopolitics", ExternalTagId: 100265, Description: "地缘政治类", Enabled: true, CreateTime: now, UpdateTime: now},
	}

	keepSlugs := make([]string, 0, len(rows))
	for _, row := range rows {
		keepSlugs = append(keepSlugs, row.Slug)
	}
	if err := db.Model(&models.PolymarketDiscoveryTag{}).
		Where("slug NOT IN ?", keepSlugs).
		Updates(map[string]any{"enabled": false, "update_time": now}).Error; err != nil {
		return err
	}

	for _, row := range rows {
		payload := row
		payload.UpdateTime = now
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"rank", "name", "external_tag_id", "description", "enabled", "update_time"}),
		}).Create(&payload).Error; err != nil {
			return err
		}
	}

	for _, row := range rows {
		tagPayload := models.PredictTag{
			Slug:       row.Slug,
			Name:       row.Name,
			CnName:     row.Description,
			LastSeenAt: 0,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "cn_name", "update_time"}),
		}).Create(&tagPayload).Error; err != nil {
			return err
		}
	}

	extraTags := []models.PredictTag{
		{Slug: "football", Name: "football", CnName: "足球", LastSeenAt: 0, CreateTime: now, UpdateTime: now},
		{Slug: "wc", Name: "wc", CnName: "世界杯", LastSeenAt: 0, CreateTime: now, UpdateTime: now},
	}
	for _, tagPayload := range extraTags {
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "cn_name", "update_time"}),
		}).Create(&tagPayload).Error; err != nil {
			return err
		}
	}
	return nil
}
