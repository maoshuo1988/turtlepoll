package migrations

import (
	"bbs-go/internal/models/models"
	"fmt"

	"github.com/mlogclub/simple/sqls"
)

func migrate_pet_definition_add_display_pricing_fields() error {
	db := sqls.DB()
	m := db.Migrator()
	if !m.HasColumn(&models.PetDefinition{}, "display_json") {
		if err := m.AddColumn(&models.PetDefinition{}, "display_json"); err != nil {
			return fmt.Errorf("add column display_json: %w", err)
		}
	}
	if !m.HasColumn(&models.PetDefinition{}, "pricing_json") {
		if err := m.AddColumn(&models.PetDefinition{}, "pricing_json"); err != nil {
			return fmt.Errorf("add column pricing_json: %w", err)
		}
	}
	// backfill with empty json
	_ = db.Exec("UPDATE t_pet_definition SET display_json = COALESCE(display_json, '{}')").Error
	_ = db.Exec("UPDATE t_pet_definition SET pricing_json = COALESCE(pricing_json, '{}')").Error
	return nil
}
