package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/polymarket"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"

	"gorm.io/gorm"
)

var PolymarketSyncService = newPolymarketSyncService()

type polymarketSyncService struct{}

func newPolymarketSyncService() *polymarketSyncService { return &polymarketSyncService{} }

// SyncMarkets 同步 Polymarket Gamma markets（只同步配置指定范围；不拉价格盘口）。
func (s *polymarketSyncService) SyncMarkets(ctx context.Context) error {
	cfg := config.Instance
	pm := cfg.Polymarket
	if !pm.Enabled {
		slog.Debug("polymarket sync disabled, skip")
		return nil
	}
	if len(pm.Tags) == 0 && len(pm.MarketSlugs) == 0 {
		return errors.New("polymarket enabled but no tags/marketSlugs configured")
	}

	client := polymarket.NewGammaClient(pm.BaseURL)

	// 建立 tag slug -> tagId 映射（用于按 tag 筛选）
	tags, err := client.ListTags(ctx)
	if err != nil {
		return err
	}
	tagSlugToID := map[string]int64{}
	for _, t := range tags {
		slug := strings.ToLower(strings.TrimSpace(t.Slug))
		if slug == "" {
			continue
		}
		tagSlugToID[slug] = t.ID
	}

	pageSize := pm.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	now := dates.NowTimestamp()
	db := sqls.DB()

	// 1) 按 tag 同步
	for _, tagSlug := range pm.Tags {
		tagSlug = strings.ToLower(strings.TrimSpace(tagSlug))
		if tagSlug == "" {
			continue
		}
		tagID, ok := tagSlugToID[tagSlug]
		if !ok {
			slog.Warn("polymarket tag not found in gamma tags", slog.String("tag", tagSlug))
			continue
		}

		params := map[string]string{
			"tag_id": strconv.FormatInt(tagID, 10),
		}
		if err := s.syncMarketsPaged(ctx, db, client, params, []string{"polymarket", tagSlug}, now, pageSize); err != nil {
			return err
		}
	}

	// 2) 白名单 market slug 同步（逐个拉列表：gamma markets 不保证支持 slug 精确查询，这里用列表 + 本地过滤）
	if len(pm.MarketSlugs) > 0 {
		target := map[string]bool{}
		for _, ms := range pm.MarketSlugs {
			s := strings.ToLower(strings.TrimSpace(ms))
			if s != "" {
				target[s] = true
			}
		}
		params := map[string]string{}
		// 扫一遍 limited 列表并过滤 slug；控制最大页数避免全站导出
		maxPages := 50
		for page := 0; page < maxPages; page++ {
			offset := page * pageSize
			list, err := client.ListMarkets(ctx, pageSize, offset, params)
			if err != nil {
				return err
			}
			if len(list) == 0 {
				break
			}
			for _, m := range list {
				slug := strings.ToLower(strings.TrimSpace(m.Slug))
				if slug == "" || !target[slug] {
					continue
				}
				if err := s.upsertMarketAndContext(db, &m, []string{"polymarket"}, now); err != nil {
					return err
				}
			}
			if len(list) < pageSize {
				break
			}
		}
	}

	return nil
}

func (s *polymarketSyncService) syncMarketsPaged(ctx context.Context, db *gorm.DB, client *polymarket.GammaClient, params map[string]string, baseTags []string, now int64, pageSize int) error {
	// 控制最大页数，避免配置错误导致全站导出
	maxPages := 200
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		list, err := client.ListMarkets(ctx, pageSize, offset, params)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			break
		}
		for i := range list {
			m := list[i]
			if err := s.upsertMarketAndContext(db, &m, baseTags, now); err != nil {
				return err
			}
		}
		if len(list) < pageSize {
			break
		}
	}
	return nil
}

func (s *polymarketSyncService) upsertMarketAndContext(db *gorm.DB, m *polymarket.Market, baseTags []string, now int64) error {
	if m == nil {
		return nil
	}
	// source_model/source_model_id 现有表是 int64，这里用 ExternalKey 保存外部 market id（字符串化）
	// 同时用 SourceModel 固定为 Polymarket
	externalMarketID := anyToString(m.ID)
	if externalMarketID == "" {
		return nil
	}

	title := strings.TrimSpace(m.Question)
	if title == "" {
		title = strings.TrimSpace(m.Title)
	}
	if title == "" {
		title = "Polymarket Market " + externalMarketID
	}

	closeTs := parseGammaTimeToUnix(m.CloseDate)
	if closeTs <= 0 {
		closeTs = parseGammaTimeToUnix(m.EndDate)
	}

	resolved, outID, outName, resolvedAt := extractResolution(m)
	if resolved && resolvedAt <= 0 {
		resolvedAt = now
	}

	status := "OPEN"
	if resolved {
		status = "CLOSE"
	} else if closeTs > 0 && closeTs <= time.Now().Unix() {
		status = "CLOSE"
	}

	market := &models.PredictMarket{}
	// 路线1：复用 SourceModel/SourceModelId，SourceModelId 只能 int64，因此用 ExternalKey 承载外部ID。
	// 为保证幂等：用 (SourceModel, ExternalKey) 做查找。
	if err := db.Where("source_model = ? AND external_key = ?", "Polymarket", externalMarketID).First(market).Error; err != nil {
		market.SourceModel = "Polymarket"
		market.SourceModelId = 0
		market.ExternalKey = externalMarketID
		market.MarketType = "AB"
		market.Title = title
		market.Status = status
		if closeTs > 0 {
			market.CloseTime = closeTs
		}
		market.Resolved = resolved
		market.ResolvedOutcomeId = outID
		market.ResolvedOutcomeName = outName
		market.ResolvedAt = resolvedAt
		market.CreateTime = now
		market.UpdateTime = now
		if ce := db.Create(market).Error; ce != nil {
			return ce
		}
	} else {
		market.Title = title
		market.Status = status
		if closeTs > 0 {
			market.CloseTime = closeTs
		}
		// resolved 一旦 true，不回写 false
		if resolved {
			market.Resolved = true
			market.ResolvedOutcomeId = outID
			market.ResolvedOutcomeName = outName
			market.ResolvedAt = resolvedAt
		}
		market.UpdateTime = now
		if ue := db.Save(market).Error; ue != nil {
			return ue
		}
	}

	// tags：baseTags + market.Tags（slug）
	tags := make([]string, 0, 8)
	seen := map[string]bool{}
	addTag := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		tags = append(tags, t)
	}
	for _, bt := range baseTags {
		addTag(bt)
	}
	for _, t := range m.Tags {
		addTag(t.Slug)
	}

	eventName := title
	if m.Event != nil && strings.TrimSpace(m.Event.Title) != "" {
		eventName = strings.TrimSpace(m.Event.Title)
	}

	ctxModel := &models.PredictContext{}
	if e := db.Where("market_id = ?", market.Id).First(ctxModel).Error; e != nil {
		ctxModel.MarketId = market.Id
		ctxModel.EventName = eventName
		ctxModel.Tags = strings.Join(tags, ",")
		ctxModel.CreateTime = now
		ctxModel.UpdateTime = now
		if ce := db.Create(ctxModel).Error; ce != nil {
			return ce
		}
	} else {
		ctxModel.EventName = eventName
		ctxModel.Tags = strings.Join(tags, ",")
		ctxModel.UpdateTime = now
		if ue := db.Save(ctxModel).Error; ue != nil {
			return ue
		}
	}

	return nil
}

func parseGammaTimeToUnix(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// 尝试 RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}
	// 尝试常见格式（Gamma 有时带毫秒）
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Unix()
	}
	return 0
}

func extractResolution(m *polymarket.Market) (resolved bool, outcomeID, outcomeName string, resolvedAt int64) {
	if m == nil {
		return false, "", "", 0
	}
	resolved = m.Resolved
	resolvedAt = parseGammaTimeToUnix(m.ResolvedAt)

	// 取消/无效：若 closed 且 resolution 标记类似 cancelled/invalid
	res := strings.TrimSpace(m.Resolution)
	if strings.EqualFold(res, "CANCELLED") || strings.EqualFold(res, "INVALID") {
		return true, "__CANCELLED__", "CANCELLED", resolvedAt
	}

	if res != "" {
		// 有些市场直接把赢家写在 resolution 里
		outcomeName = res
		outcomeID = res
		return resolved || m.Closed, outcomeID, outcomeName, resolvedAt
	}

	// outcomes 中若能识别 winner（Gamma 字段不完全一致），这里先做保守策略：
	// 如果 resolved==true 但没有 resolution，就留空；后续有需要再增强解析。
	return resolved, "", "", resolvedAt
}

func anyToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		// json number 默认 float64
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
