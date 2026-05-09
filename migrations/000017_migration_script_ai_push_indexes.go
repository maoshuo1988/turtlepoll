package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_ai_push_indexes() error {
	db := sqls.DB()
	if err := db.AutoMigrate(
		&models.AIMessage{},
		&models.UserAIMemory{},
		&models.DialogueTemplate{},
		&models.TemplateUserView{},
		&models.UserAIPresence{},
	); err != nil {
		slog.Error("migrate ai push tables failed", "error", err)
		return err
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_ai_message_user_request ON t_ai_message(user_id, request_id) WHERE request_id IS NOT NULL AND request_id <> ''").Error; err != nil {
		slog.Error("migrate ai push request index failed", "error", err)
		return err
	}
	return nil
}
