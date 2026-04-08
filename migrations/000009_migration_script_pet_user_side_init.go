package migrations

import (
	"bbs-go/internal/models/models"
	"log/slog"

	"github.com/mlogclub/simple/sqls"
)

func migrate_pet_user_side_init_tables() error {
	db := sqls.DB()
	if err := db.AutoMigrate(&models.UserPetState{}, &models.UserPet{}, &models.PetDailySettleLog{}); err != nil {
		slog.Error("migrate pet user-side tables failed", "error", err)
		return err
	}
	return nil
}
