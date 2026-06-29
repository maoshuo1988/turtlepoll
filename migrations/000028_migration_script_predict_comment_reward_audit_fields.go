package migrations

import (
	"bbs-go/internal/models/models"

	"github.com/mlogclub/simple/sqls"
)

func migrate_predict_comment_reward_audit_fields() error {
	db := sqls.DB()
	return db.AutoMigrate(&models.PredictCommentRewardLog{}, &models.PredictCommentRewardItem{})
}
