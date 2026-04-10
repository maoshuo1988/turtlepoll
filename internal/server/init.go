package server

import (
	"bbs-go/internal/install"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/services"

	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Init() {
	install.InitConfig()
	install.InitLogger()
	install.InitLocales()
	if config.Instance.Installed {
		if err := install.InitDB(); err != nil {
			panic(err)
		}
		if err := install.InitMigrations(); err != nil {
			panic(err)
		}
		// Ensure FeatureCatalog has default items so pet abilities can be validated/executed.
		services.FeatureCatalogService.EnsureDefaultSeeds()
		install.InitOthers()
	}
}
