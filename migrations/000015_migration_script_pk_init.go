package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func migrate_pk_init_tables() error {
	db := sqls.DB()
	if err := db.AutoMigrate(
		&models.PKTopic{},
		&models.PKSeason{},
		&models.PKRound{},
		&models.PKBet{},
		&models.PKSettlementItem{},
		&models.PKCommentMeta{},
		&models.PKAction{},
	); err != nil {
		slog.Error("migrate pk tables failed", "error", err)
		return err
	}

	now := dates.NowTimestamp()
	seeds := []struct {
		Slug  string
		Title string
		A     string
		B     string
	}{
		{"pk-hero", "足球GOAT之争", "梅西", "C罗"},
		{"pk-1", "女团门面之争", "张元英", "柳智敏"},
		{"pk-2", "未来超级大国", "中国", "美国"},
		{"pk-3", "手机之王", "iPhone", "安卓"},
		{"pk-4", "超级英雄宇宙", "漫威", "DC"},
		{"pk-5", "电竞GOAT", "Faker", "Uzi"},
		{"pk-6", "人类最好的伙伴", "猫", "狗"},
		{"pk-7", "豆腐脑之争", "甜", "咸"},
		{"pk-8", "学历vs能力", "学历", "能力"},
		{"pk-9", "华语乐坛之王", "周杰伦", "林俊杰"},
		{"pk-10", "NBA历史地位", "詹姆斯", "库里"},
		{"pk-11", "世界之都", "纽约", "伦敦"},
		{"pk-12", "新能源汽车之王", "特斯拉", "小米"},
		{"pk-13", "手机操作系统", "安卓", "iOS"},
		{"pk-14", "外卖平台大战", "美团", "饿了么"},
		{"pk-15", "短视频霸主", "抖音", "快手"},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for i, seed := range seeds {
			var count int64
			if err := tx.Model(&models.PKTopic{}).Where("slug = ?", seed.Slug).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}
			topic := &models.PKTopic{
				Slug:       seed.Slug,
				Title:      seed.Title,
				SideAName:  seed.A,
				SideBName:  seed.B,
				Status:     "enabled",
				Sort:       (i + 1) * 10,
				CreateTime: now,
				UpdateTime: now,
			}
			if err := tx.Create(topic).Error; err != nil {
				return err
			}
			season := &models.PKSeason{
				TopicId:    topic.Id,
				SeasonNo:   1,
				StartTime:  now,
				EndTime:    now + 30*24*3600,
				Status:     "active",
				CreateTime: now,
				UpdateTime: now,
			}
			if err := tx.Create(season).Error; err != nil {
				return err
			}
			round := &models.PKRound{
				TopicId:       topic.Id,
				SeasonId:      season.Id,
				RoundNo:       1,
				Phase:         "betting",
				StartTime:     now,
				LockTime:      now + 48*3600,
				EndTime:       now + 72*3600,
				NextRoundTime: now + 72*3600 + 10*60,
				CreateTime:    now,
				UpdateTime:    now,
			}
			if err := tx.Create(round).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.PKTopic{}).Where("id = ?", topic.Id).
				Updates(map[string]interface{}{"current_season_id": season.Id, "current_round_id": round.Id, "update_time": now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
