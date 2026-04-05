package services

import (
	"bbs-go/internal/models/models"
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
		err := errors.New("polymarket enabled but no tags/marketSlugs configured")
		slog.Error("polymarket sync config invalid", slog.Any("err", err))
		return err
	}

	slog.Info("polymarket sync start",
		slog.Int("tags", len(pm.Tags)),
		slog.Int("marketSlugs", len(pm.MarketSlugs)),
		slog.Int("pageSize", pm.PageSize),
		slog.String("baseURL", pm.BaseURL),
	)

	client := polymarket.NewGammaClient(pm.BaseURL)

	// 建立 tag slug -> tagId 映射（用于按 tag 筛选）
	tags, err := client.ListTags(ctx)
	if err != nil {
		slog.Error("polymarket list tags failed", slog.Any("err", err))
		return err
	}
	slog.Info("polymarket list tags done", slog.Int("count", len(tags)))
	tagSlugToID := map[string]int64{}
	for _, t := range tags {
		slug := strings.ToLower(strings.TrimSpace(t.Slug))
		if slug == "" {
			continue
		}
		id := anyToInt64(t.ID)
		if id <= 0 {
			// 不阻断整体同步：tag 没有可用 id 就跳过
			continue
		}
		tagSlugToID[slug] = id
	}
	slog.Info("polymarket tag mapping built", slog.Int("mapped", len(tagSlugToID)))

	pageSize := pm.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	now := dates.NowTimestamp()
	db := sqls.DB()

	var (
		seenMarkets  int
		upsertOK     int
		upsertFailed int
	)

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
		slog.Info("polymarket sync by tag start", slog.String("tag", tagSlug), slog.Int64("tagID", tagID))

		params := map[string]string{
			"tag_id": strconv.FormatInt(tagID, 10),
		}
		if err := s.syncMarketsPaged(ctx, db, client, params, []string{"polymarket", tagSlug}, now, pageSize, &seenMarkets, &upsertOK, &upsertFailed); err != nil {
			return err
		}
		slog.Info("polymarket sync by tag done", slog.String("tag", tagSlug))
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
				slog.Error("polymarket list markets failed", slog.Any("err", err), slog.Int("page", page), slog.Int("offset", offset))
				return err
			}
			if len(list) == 0 {
				break
			}
			slog.Info("polymarket scan markets page", slog.Int("page", page), slog.Int("offset", offset), slog.Int("count", len(list)))
			for _, m := range list {
				slug := strings.ToLower(strings.TrimSpace(m.Slug))
				if slug == "" || !target[slug] {
					continue
				}
				seenMarkets++
				if err := s.upsertMarketAndContext(db, &m, []string{"polymarket"}, now); err != nil {
					upsertFailed++
					slog.Error("polymarket upsert market failed", slog.Any("err", err), slog.String("slug", slug), slog.String("marketID", anyToString(m.ID)))
					return err
				}
				upsertOK++
			}
			if len(list) < pageSize {
				break
			}
		}
	}

	slog.Info("polymarket sync done",
		slog.Int("seenMarkets", seenMarkets),
		slog.Int("upsertOK", upsertOK),
		slog.Int("upsertFailed", upsertFailed),
	)
	return nil
}

func (s *polymarketSyncService) syncMarketsPaged(ctx context.Context, db *gorm.DB, client *polymarket.GammaClient, params map[string]string, baseTags []string, now int64, pageSize int, seenMarkets, upsertOK, upsertFailed *int) error {
	// 控制最大页数，避免配置错误导致全站导出
	maxPages := 200
	for page := 0; page < maxPages; page++ {
		offset := page * pageSize
		list, err := client.ListMarkets(ctx, pageSize, offset, params)
		if err != nil {
			slog.Error("polymarket list markets failed", slog.Any("err", err), slog.Int("page", page), slog.Int("offset", offset), slog.Any("params", params))
			return err
		}
		if len(list) == 0 {
			break
		}
		slog.Info("polymarket list markets page", slog.Int("page", page), slog.Int("offset", offset), slog.Int("count", len(list)), slog.Any("params", params))
		for i := range list {
			m := list[i]
			if seenMarkets != nil {
				*seenMarkets = *seenMarkets + 1
			}
			if err := s.upsertMarketAndContext(db, &m, baseTags, now); err != nil {
				if upsertFailed != nil {
					*upsertFailed = *upsertFailed + 1
				}
				slog.Error("polymarket upsert market failed", slog.Any("err", err), slog.String("slug", strings.ToLower(strings.TrimSpace(m.Slug))), slog.String("marketID", anyToString(m.ID)))
				return err
			}
			if upsertOK != nil {
				*upsertOK = *upsertOK + 1
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
	// source_model/source_model_id 现有表是 int64，因此用 SourceModel=Polymarket + SourceModelId(外部 market id) 来保证幂等。
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
	// 注意：表的唯一索引是 (source_model, source_model_id)。
	// 为避免所有 Polymarket 市场都落在 source_model_id=0 导致冲突，这里把外部 market id 尽量解析成 int64 写入 source_model_id。
	// 不再写 external_key。
	externalMarketIDNum := anyToInt64(m.ID)
	if externalMarketIDNum <= 0 {
		// 极少数情况下外部 id 不是纯数字：为了保证数据可落库且不冲突，用时间戳做一个弱兜底。
		// 说明：该兜底不是强幂等；如遇到非数字 id，建议后续升级表结构（例如增加 (source_model, external_key) uniqueIndex）。
		externalMarketIDNum = now
	}

	// 幂等查找：优先走唯一索引 (source_model, source_model_id)
	err := db.Where("source_model = ? AND source_model_id = ?", "Polymarket", externalMarketIDNum).First(market).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		market.SourceModel = "Polymarket"
		market.SourceModelId = externalMarketIDNum
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
	e := db.Where("market_id = ?", market.Id).First(ctxModel).Error
	if e != nil {
		if !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
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

func anyToInt64(v any) int64 {
	s := anyToString(v)
	if s == "" {
		return 0
	}
	// Gamma 可能返回 "123" 或 123（被 json 解成 float64）；这里统一走 ParseInt。
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
