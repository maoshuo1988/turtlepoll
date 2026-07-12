package migrations

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm/clause"
)

func migrate_message_notify_reward_scene_templates() error {
	db := sqls.DB()
	now := dates.NowTimestamp()
	templates := []models.MessageNotifyTemplate{
		{
			BusinessCode:      services.MessageNotifyBusinessReward,
			TemplateCode:      "signup_bonus",
			SubjectTemplate:   "新手礼包已到账",
			BodyTemplate:      "{petName} 已为你送上 {amount} 龟币，欢迎来到龟投。",
			DetailUrlTemplate: "/pet",
			TemplateParams:    `["petName","amount"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "新用户注册赠送龟币",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessReward,
			TemplateCode:      "first_bet_bonus_predict",
			SubjectTemplate:   "财神龟首投奖励已到账",
			BodyTemplate:      "你在{sceneName}「{targetTitle}」完成今日首次下注，额外获得 {amount} 龟币。",
			DetailUrlTemplate: "{detailUrl}",
			TemplateParams:    `["sceneName","targetTitle","amount","detailUrl"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "财神龟暗盘每日首次下注奖励",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessReward,
			TemplateCode:      "first_bet_bonus_pk",
			SubjectTemplate:   "财神龟首投奖励已到账",
			BodyTemplate:      "你在{sceneName}「{targetTitle}」完成今日首次下注，额外获得 {amount} 龟币。",
			DetailUrlTemplate: "{detailUrl}",
			TemplateParams:    `["sceneName","targetTitle","amount","detailUrl"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "财神龟开撕台每日首次下注奖励",
		},
	}
	for _, item := range templates {
		tpl := item
		tpl.CreateTime = now
		tpl.UpdateTime = now
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "business_code"}, {Name: "template_code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"subject_template",
				"body_template",
				"detail_url_template",
				"template_params",
				"status",
				"remark",
				"update_time",
			}),
		}).Create(&tpl).Error; err != nil {
			return err
		}
	}
	return nil
}
