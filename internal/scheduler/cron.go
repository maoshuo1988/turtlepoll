package scheduler

import (
	"log/slog"

	"bbs-go/internal/pkg/config"
	"bbs-go/internal/services"
	"context"

	"github.com/robfig/cron/v3"
)

func Start() {
	c := cron.New()

	// football-data 世界杯赛程同步（默认每 30 分钟一次）
	spec := config.Instance.FootballData.CronSpec
	if spec == "" {
		// robfig/cron 默认支持 5 或 6 字段，若用秒字段需要 WithSeconds；这里沿用 5 字段：分 时 日 月 周
		spec = "*/30 * * * *"
	}
	addCronFunc(c, spec, func() {
		if err := services.FootballSyncService.SyncWorldCupSchedules(context.Background()); err != nil {
			slog.Error("football schedule sync failed", slog.Any("err", err))
		}
	})

	c.Start()
}

func addCronFunc(c *cron.Cron, sepc string, cmd func()) {
	if _, err := c.AddFunc(sepc, cmd); err != nil {
		slog.Error("add cron func error", slog.Any("err", err))
	}
}
