package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/footballdata"
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
)

var FootballSyncService = newFootballSyncService()

type footballSyncService struct{}

func newFootballSyncService() *footballSyncService { return &footballSyncService{} }

// SyncWorldCupSchedules 拉取 football-data 世界杯赛程并落库，同时为每个赛程创建/更新一个预测市场。
// 这里是“第 0 阶段”：只做数据同步 + 市场占位，不实现下注/结算。
func (s *footballSyncService) SyncWorldCupSchedules(ctx context.Context) error {
	cfg := config.Instance
	fd := cfg.FootballData
	if fd.APIKey == "" {
		// 没配置就不跑，避免 prod 启动后一直报错
		slog.Warn("football-data api key not configured, skip sync")
		return nil
	}
	client := footballdata.NewClient(fd.APIKey)
	if fd.BaseURL != "" {
		client.BaseURL = fd.BaseURL
	}
	competition := fd.CompetitionCode
	if competition == "" {
		competition = "WC"
	}

	resp, err := client.GetCompetitionMatches(ctx, competition, fd.Season)
	if err != nil {
		return err
	}
	slog.Info("football-data api response received", slog.Int("matches", len(resp.Matches)))
	now := dates.NowTimestamp()
	db := sqls.DB()
	// title 统一生成：避免空队名导致的 " vs " 或误导性标题
	buildMarketTitle := func(home, away string) string {
		if home != "" && away != "" {
			return home + " vs " + away
		}
		if home != "" {
			return home + " vs TBD"
		}
		if away != "" {
			return "TBD vs " + away
		}
		return "TBD vs TBD"
	}
	// 只有主客队都有值才允许 OPEN，其余都关闭
	isTeamsReady := func(home, away string) bool {
		return home != "" && away != ""
	}
	for _, m := range resp.Matches {
		schedule := &models.MatchSchedule{}
		err := db.Where("source = ? AND external_id = ?", "football-data", m.ID).First(schedule).Error
		// upsert-ish
		if err != nil {
			// create
			schedule.Source = "football-data"
			schedule.ExternalId = m.ID
			schedule.CreateTime = now
		}
		schedule.Competition = m.Competition.Code
		if schedule.Competition == "" {
			schedule.Competition = competition
		}
		schedule.Season = fd.Season
		if schedule.Season == 0 {
			schedule.Season = m.Season.Year
		}
		schedule.Matchday = m.Matchday
		schedule.Stage = m.Stage
		schedule.GroupName = m.Group
		schedule.Status = m.Status
		schedule.UtcDate = m.UtcDate.Unix()
		schedule.HomeTeam = m.HomeTeam.Name
		schedule.AwayTeam = m.AwayTeam.Name
		schedule.HomeTeamId = m.HomeTeam.ID
		schedule.AwayTeamId = m.AwayTeam.ID
		schedule.LastSyncedAt = now
		schedule.UpdateTime = now

		if schedule.Id == 0 {
			if e := db.Create(schedule).Error; e != nil {
				return e
			}
		} else {
			if e := db.Save(schedule).Error; e != nil {
				return e
			}
		}

		// 每个赛程一个预测市场
		market := &models.PredictMarket{}
		title := buildMarketTitle(schedule.HomeTeam, schedule.AwayTeam)
		desiredStatus := "CLOSE"
		if isTeamsReady(schedule.HomeTeam, schedule.AwayTeam) {
			desiredStatus = "OPEN"
		}
		if e := db.Where("source_model = ? AND source_model_id = ?", "MatchSchedule", schedule.Id).First(market).Error; e != nil {
			market.SourceModel = "MatchSchedule"
			market.SourceModelId = schedule.Id
			market.MarketType = "1x2"
			market.Status = desiredStatus
			// 默认在开赛前 10 分钟关闭（先占位规则）
			if schedule.UtcDate > 0 {
				market.CloseTime = schedule.UtcDate - int64((10 * time.Minute).Seconds())
			}
			market.Title = title
			market.CreateTime = now
			market.UpdateTime = now
			if ce := db.Create(market).Error; ce != nil {
				return ce
			}
		} else {
			// 每次同步都更新 title 和 status；closeTime 按赛程时间刷新
			market.Title = title
			market.Status = desiredStatus
			if schedule.UtcDate > 0 {
				market.CloseTime = schedule.UtcDate - int64((10 * time.Minute).Seconds())
			}
			market.UpdateTime = now
			if ue := db.Save(market).Error; ue != nil {
				return ue
			}
		}

		// 市场上下文（展示用，一对一）
		ctxModel := &models.PredictContext{}
		competitionTag := strings.ToLower(schedule.Competition)
		if e := db.Where("market_id = ?", market.Id).First(ctxModel).Error; e != nil {
			ctxModel.MarketId = market.Id
			ctxModel.EventName = market.Title
			ctxModel.ImageUrl = ""
			ctxModel.ParticipantCount = 0
			ctxModel.ProText = schedule.HomeTeam + " 胜"
			ctxModel.ConText = schedule.AwayTeam + " 胜"
			ctxModel.Detail = ""
			ctxModel.Tags = "football," + competitionTag
			ctxModel.CreateTime = now
			ctxModel.UpdateTime = now
			if ce := db.Create(ctxModel).Error; ce != nil {
				return ce
			}
		} else {
			// 仅更新动态字段，避免覆盖人工编辑的详情/图片等
			// 注意：event_name 如果不刷新，会导致前端仍看到 "TBD vs TBD"
			ctxModel.EventName = market.Title
			ctxModel.ProText = schedule.HomeTeam + " 胜"
			ctxModel.ConText = schedule.AwayTeam + " 胜"
			// tags 作为轻量元数据也跟随刷新，避免 competition 变化导致查询不到
			ctxModel.Tags = "football," + competitionTag
			ctxModel.UpdateTime = now
			if ue := db.Save(ctxModel).Error; ue != nil {
				return ue
			}
		}
	}

	slog.Info("football schedules synced", slog.Int("matches", len(resp.Matches)))
	return nil
}
