package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_pk_topic_display_fields() error {
	if err := sqls.DB().AutoMigrate(&models.PKTopic{}); err != nil {
		slog.Error("migrate pk topic display fields failed", "error", err)
		return err
	}
	return nil
}
