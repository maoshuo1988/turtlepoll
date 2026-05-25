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

func footballMatchPhase(stage, groupName string) string {
	stage = strings.ToUpper(strings.TrimSpace(stage))
	groupName = strings.TrimSpace(groupName)
	if stage == "GROUP_STAGE" || groupName != "" {
		return MatchPhaseGroup
	}
	switch stage {
	case "LAST_16", "QUARTER_FINALS", "SEMI_FINALS", "THIRD_PLACE", "FINAL":
		return MatchPhaseKnockout
	case "":
		return MatchPhaseUnknown
	default:
		return MatchPhaseKnockout
	}
}

func footballMarketTypeByPhase(phase string) string {
	if phase == MatchPhaseGroup {
		return PredictMarketType1x2
	}
	return PredictMarketTypeBinary
}

func footballTeamsReady(home, away string) bool {
	return strings.TrimSpace(home) != "" && strings.TrimSpace(away) != ""
}

func footballTeamsFrozen(schedule *models.MatchSchedule) bool {
	if schedule == nil {
		return false
	}
	return schedule.HomeTeamId > 0 && schedule.AwayTeamId > 0 &&
		strings.TrimSpace(schedule.HomeTeam) != "" &&
		strings.TrimSpace(schedule.AwayTeam) != ""
}

func footballScore(m footballdata.Match) (homeScore, awayScore int, ok bool) {
	if m.Score.FullTime.Home == nil || m.Score.FullTime.Away == nil {
		return -1, -1, false
	}
	return *m.Score.FullTime.Home, *m.Score.FullTime.Away, true
}

func footballWinnerByScore(homeScore, awayScore int, phase string) string {
	if homeScore < 0 || awayScore < 0 {
		return ""
	}
	if homeScore > awayScore {
		return PredictOptionA
	}
	if homeScore < awayScore {
		return PredictOptionB
	}
	if phase == MatchPhaseGroup {
		return PredictOptionDraw
	}
	return MatchPhaseUnknown
}

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
	for _, m := range resp.Matches {
		schedule := &models.MatchSchedule{}
		err := db.Where("source = ? AND external_id = ?", "football-data", m.ID).First(schedule).Error
		existing := schedule.Id > 0
		teamsFrozen := existing && footballTeamsFrozen(schedule)
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
		schedule.MatchPhase = footballMatchPhase(m.Stage, m.Group)
		schedule.Status = m.Status
		schedule.UtcDate = m.UtcDate.Unix()
		if !teamsFrozen {
			schedule.HomeTeam = translateFootballTeamName(m.HomeTeam.ID, m.HomeTeam.Name)
			schedule.AwayTeam = translateFootballTeamName(m.AwayTeam.ID, m.AwayTeam.Name)
			schedule.HomeTeamId = m.HomeTeam.ID
			schedule.AwayTeamId = m.AwayTeam.ID
		} else {
			if homeTeam, ok := translateKnownFootballTeamName(schedule.HomeTeamId, schedule.HomeTeam); ok {
				schedule.HomeTeam = homeTeam
			}
			if awayTeam, ok := translateKnownFootballTeamName(schedule.AwayTeamId, schedule.AwayTeam); ok {
				schedule.AwayTeam = awayTeam
			}
		}
		if homeScore, awayScore, ok := footballScore(m); ok {
			schedule.HomeScore = homeScore
			schedule.AwayScore = awayScore
			schedule.Winner = footballWinnerByScore(homeScore, awayScore, schedule.MatchPhase)
			if schedule.Winner == MatchPhaseUnknown && strings.EqualFold(schedule.Status, "FINISHED") {
				slog.Error("knockout match ended draw, manual settlement required",
					slog.Int64("externalId", schedule.ExternalId),
					slog.String("stage", schedule.Stage),
					slog.String("groupName", schedule.GroupName),
					slog.Int("homeScore", homeScore),
					slog.Int("awayScore", awayScore),
				)
			}
		}
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
		desiredStatus := "CLOSED"
		if footballTeamsReady(schedule.HomeTeam, schedule.AwayTeam) {
			desiredStatus = "OPEN"
		}
		if schedule.Status == "IN_PLAY" || schedule.Status == "PAUSED" || schedule.Status == "FINISHED" ||
			schedule.Status == "POSTPONED" || schedule.Status == "CANCELLED" || schedule.Status == "SUSPENDED" {
			desiredStatus = "CLOSED"
		}
		marketType := footballMarketTypeByPhase(schedule.MatchPhase)
		if e := db.Where("source_model = ? AND source_model_id = ?", "MatchSchedule", schedule.Id).First(market).Error; e != nil {
			market.SourceModel = "MatchSchedule"
			market.SourceModelId = schedule.Id
			market.MarketType = marketType
			market.Status = desiredStatus
			// 默认在开赛前 10 分钟关闭（先占位规则）
			if schedule.UtcDate > 0 {
				market.CloseTime = schedule.UtcDate - int64((10 * time.Minute).Seconds())
			}
			market.Title = title
			if strings.EqualFold(schedule.Status, "FINISHED") && schedule.Winner != "" && schedule.Winner != MatchPhaseUnknown {
				market.Status = "SETTLED"
				market.Result = schedule.Winner
				market.Resolved = true
				if market.ResolvedAt == 0 {
					market.ResolvedAt = now
				}
			}
			market.CreateTime = now
			market.UpdateTime = now
			if ce := db.Create(market).Error; ce != nil {
				return ce
			}
		} else {
			// 每次同步都更新 title 和 status；closeTime 按赛程时间刷新
			market.Title = title
			market.MarketType = marketType
			if market.Status != "SETTLED" {
				market.Status = desiredStatus
			}
			if schedule.UtcDate > 0 {
				market.CloseTime = schedule.UtcDate - int64((10 * time.Minute).Seconds())
			}
			if market.Status != "SETTLED" && strings.EqualFold(schedule.Status, "FINISHED") && schedule.Winner != "" && schedule.Winner != MatchPhaseUnknown {
				market.Status = "SETTLED"
				market.Result = schedule.Winner
				market.Resolved = true
				if market.ResolvedAt == 0 {
					market.ResolvedAt = now
				}
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
			ctxModel.ListImage = ""
			ctxModel.SideABgImage = ""
			ctxModel.SideBBgImage = ""
			ctxModel.SideABgColor = "#E23D3D"
			ctxModel.SideBBgColor = "#276EF1"
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
