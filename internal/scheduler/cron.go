package scheduler

import (
	"fmt"
	"log/slog"
	"time"

	"bbs-go/internal/pkg/config"
	"bbs-go/internal/services"
	"context"

	"github.com/robfig/cron/v3"
)

func Start() {
	c := cron.New()
	slog.Info("scheduler cron start")

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

	// polymarket 只读同步（默认每 30 分钟一次；需 enabled=true 且配置了 tags 或 marketSlugs）
	pm := config.Instance.Polymarket
	if pm.Enabled {
		pmSpec := pm.CronSpec
		if pmSpec == "" {
			pmSpec = "*/30 * * * *"
		}
		addCronFunc(c, pmSpec, func() {
			if err := services.PolymarketSyncService.SyncMarkets(context.Background()); err != nil {
				slog.Error("polymarket sync failed", slog.Any("err", err))
			}
		})
	}

	// battle square 后台轮巡（每 1 分钟一次）
	addCronFunc(c, "*/1 * * * *", func() {
		start := time.Now()
		slog.Info("battle cron tick start", slog.Time("now", start))
		if err := services.BattleService.CronTick(); err != nil {
			slog.Error("battle cron tick failed", slog.Any("err", err))
			return
		}
		slog.Info("battle cron tick done", slog.Duration("cost", time.Since(start)))
	})

	// 预测市场标签物化刷新（默认每 30 分钟一次）
	addCronFunc(c, "*/30 * * * *", func() {
		if err := services.PredictTagService.RefreshTagsFromContexts(); err != nil {
			slog.Error("predict tag refresh failed", slog.Any("err", err))
		}
	})

	c.Start()
	slog.Info("scheduler cron started")
}

func addCronFunc(c *cron.Cron, sepc string, cmd func()) {
	if _, err := c.AddFunc(sepc, cmd); err != nil {
		slog.Error("add cron func error", slog.String("spec", sepc), slog.Any("err", err))
		return
	}
	slog.Info("add cron func ok", slog.String("spec", sepc), slog.String("cmd", fmt.Sprintf("%p", cmd)))
}
