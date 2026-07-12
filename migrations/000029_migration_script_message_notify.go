package migrations

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm/clause"
)

func migrate_message_notify_tables() error {
	db := sqls.DB()
	now := dates.NowTimestamp()
	if err := db.AutoMigrate(&models.MessageNotifyTemplate{}, &models.UserMessageNotifyRecord{}); err != nil {
		return err
	}
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uk_user_message_notify_idem_not_empty ON t_user_message_notify_record(user_id, idempotency_key) WHERE idempotency_key <> ''`).Error; err != nil {
		return err
	}

	templates := []models.MessageNotifyTemplate{
		{
			BusinessCode:      services.MessageNotifyBusinessReward,
			TemplateCode:      "daily_login_reward",
			SubjectTemplate:   "今日登录奖励已到账",
			BodyTemplate:      "领取 {amount} 龟币，连续登录 {loginStreak} 天。",
			DetailUrlTemplate: "/rewards/daily",
			TemplateParams:    `["amount","loginStreak"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "每日登录奖励",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessDarkMarket,
			TemplateCode:      "predict_settle_win",
			SubjectTemplate:   "你参与的 {marketTitle} 已结算",
			BodyTemplate:      "预测命中，奖励 {payout} 龟币已进入余额。",
			DetailUrlTemplate: "/predict/markets/{marketId}",
			TemplateParams:    `["marketTitle","payout","marketId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "预测结算命中",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessDarkMarket,
			TemplateCode:      "predict_settle_lose",
			SubjectTemplate:   "你参与的 {marketTitle} 已结算",
			BodyTemplate:      "本次未命中，消耗 {amount} 龟币。",
			DetailUrlTemplate: "/predict/markets/{marketId}",
			TemplateParams:    `["marketTitle","amount","marketId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "预测结算未命中",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessDarkMarket,
			TemplateCode:      "predict_settle_refund",
			SubjectTemplate:   "你参与的 {marketTitle} 已退款",
			BodyTemplate:      "本次盘口已退款，{amount} 龟币已返还余额。",
			DetailUrlTemplate: "/predict/markets/{marketId}",
			TemplateParams:    `["marketTitle","amount","marketId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "预测结算退款",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessDarkMarket,
			TemplateCode:      "predict_new_market",
			SubjectTemplate:   "暗盘有新盘口",
			BodyTemplate:      "你关注的 {marketTitle} 已开盘，当前热度正在上升。",
			DetailUrlTemplate: "/predict/markets/{marketId}",
			TemplateParams:    `["marketTitle","marketId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "新盘口提醒",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessTearSquare,
			TemplateCode:      "comment_reply",
			SubjectTemplate:   "有人回复了你的观点",
			BodyTemplate:      "{nickname} 在开撕台回复了你的评论。",
			DetailUrlTemplate: "{detailUrl}",
			TemplateParams:    `["nickname","detailUrl"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "开撕台回复",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessTearSquare,
			TemplateCode:      "comment_quote",
			SubjectTemplate:   "你的观点被引用",
			BodyTemplate:      "你的观点被 {quoteCount} 位玩家引用，进入战局查看。",
			DetailUrlTemplate: "{detailUrl}",
			TemplateParams:    `["quoteCount","detailUrl"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "开撕台引用",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessSystem,
			TemplateCode:      "platform_rule_update",
			SubjectTemplate:   "平台规则更新",
			BodyTemplate:      "{title}，请在开局前查看。",
			DetailUrlTemplate: "/notice/{noticeId}",
			TemplateParams:    `["title","noticeId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "平台规则更新",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessSystem,
			TemplateCode:      "maintenance_notice",
			SubjectTemplate:   "系统维护公告",
			BodyTemplate:      "今晚 {maintenanceTime} 将进行短暂维护，期间部分功能不可用。",
			DetailUrlTemplate: "/notice/{noticeId}",
			TemplateParams:    `["maintenanceTime","noticeId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "维护公告",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessIntel,
			TemplateCode:      "intel_follow_update",
			SubjectTemplate:   "你关注的线报有更新",
			BodyTemplate:      "热门线报「{intelTitle}」新增了 {discussionCount} 条讨论和 {sourceCount} 条来源补充。",
			DetailUrlTemplate: "/intel/{intelId}",
			TemplateParams:    `["intelTitle","discussionCount","sourceCount","intelId"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "关注线报更新",
		},
		{
			BusinessCode:      services.MessageNotifyBusinessUndergroundBank,
			TemplateCode:      "pet_order_completed",
			SubjectTemplate:   "乌龟购买成功",
			BodyTemplate:      "{petName} 已放入你的宠物仓库，消耗 {cost} 龟币。",
			DetailUrlTemplate: "/pet/warehouse",
			TemplateParams:    `["petName","cost"]`,
			Status:            services.MessageNotifyTemplateEnabled,
			Remark:            "黑市宠物订单完成",
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
