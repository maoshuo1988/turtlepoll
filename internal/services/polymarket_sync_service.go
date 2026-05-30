package services

import (
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/polymarket"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	polymarketSourceModel = "Polymarket"

	trackingStatusDiscovered = "DISCOVERED"
	trackingStatusTracking   = "TRACKING"
	trackingStatusSettled    = "SETTLED"
	trackingStatusOpsPending = "OPS_PENDING"
	trackingStatusCancelled  = "CANCELLED"

	issueStatusOpen     = "OPEN"
	issueStatusResolved = "RESOLVED"
	issueStatusIgnored  = "IGNORED"

	issueNoWinner           = "NO_WINNER"
	issueNoOutcomeMapping   = "NO_OUTCOME_MAPPING"
	issueAmbiguousOutcome   = "AMBIGUOUS_OUTCOME"
	issueCancelledOrInvalid = "CANCELLED_OR_INVALID"
)

var PolymarketSyncService = newPolymarketSyncService()

type polymarketSyncService struct{}

func newPolymarketSyncService() *polymarketSyncService { return &polymarketSyncService{} }

// SyncMarkets 兼容旧入口：先发现，再按已登记 ID 跟踪一批。
func (s *polymarketSyncService) SyncMarkets(ctx context.Context) error {
	if err := s.DiscoveryPoller(ctx); err != nil {
		return err
	}
	return s.TrackingPoller(ctx)
}

func (s *polymarketSyncService) DiscoveryPoller(ctx context.Context) (retErr error) {
	pm := config.Instance.Polymarket
	if !pm.Enabled {
		slog.Debug("polymarket discovery disabled, skip")
		return nil
	}
	discoveryStart := time.Now()
	defer func() {
		fields := []any{
			slog.Int64("totalCostMs", time.Since(discoveryStart).Milliseconds()),
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			fields = append(fields, slog.String("ctxErr", ctxErr.Error()))
		}
		if retErr != nil {
			fields = append(fields, slog.Any("err", retErr))
			slog.Error("polymarket discovery finished with error", fields...)
			return
		}
		slog.Info("polymarket discovery finished", fields...)
	}()

	client := polymarket.NewGammaClient(pm.BaseURL)
	db := sqls.DB()
	now := dates.NowTimestamp()
	tagBatchSize := polymarketTrackingBatchSize(pm)

	var seenMarkets, upsertOK, upsertFailed int
	tagConfigs, err := s.listEnabledDiscoveryTags(db)
	if err != nil {
		return err
	}
	slog.Info("polymarket discovery start",
		slog.Int("tags", len(tagConfigs)),
		slog.Int("tagBatchSize", tagBatchSize),
		slog.String("baseURL", pm.BaseURL),
	)

	if len(tagConfigs) > 0 {
		resolvedTags, err := s.resolveDiscoveryTags(ctx, client, tagConfigs)
		if err != nil {
			return err
		}
		failedTags := 0
		skippedTags := 0
		for _, tag := range resolvedTags {
			if tag.ExternalTagId <= 0 {
				skippedTags++
				slog.Warn("polymarket discovery skip tag without external id",
					slog.String("slug", tag.Slug),
					slog.String("name", tag.Name),
				)
				continue
			}
			tagSyncStart := time.Now()
			params := discoveryOpenOnlyParams(map[string]string{"tag_id": strconv.FormatInt(tag.ExternalTagId, 10)})
			if err := s.syncMarketsPaged(ctx, db, client, params, []string{"polymarket", tag.Slug}, "tag", now, tagBatchSize, &seenMarkets, &upsertOK, &upsertFailed); err != nil {
				failedTags++
				slog.Error("polymarket discovery tag sync failed",
					slog.String("slug", tag.Slug),
					slog.Int64("tagId", tag.ExternalTagId),
					slog.Int64("costMs", time.Since(tagSyncStart).Milliseconds()),
					slog.Any("err", err),
				)
				continue
			}
			slog.Info("polymarket discovery tag sync done",
				slog.String("slug", tag.Slug),
				slog.Int64("tagId", tag.ExternalTagId),
				slog.Int("limit", tagBatchSize),
				slog.Int64("costMs", time.Since(tagSyncStart).Milliseconds()),
			)
		}
		if failedTags > 0 || skippedTags > 0 {
			slog.Warn("polymarket discovery completed with partial tag failures",
				slog.Int("failedTags", failedTags),
				slog.Int("skippedTags", skippedTags),
			)
		}
	}

	slog.Info("polymarket discovery done",
		slog.Int("seenMarkets", seenMarkets),
		slog.Int("upsertOK", upsertOK),
		slog.Int("upsertFailed", upsertFailed),
		slog.Int64("totalCostMs", time.Since(discoveryStart).Milliseconds()),
	)
	return nil
}

func (s *polymarketSyncService) TrackingPoller(ctx context.Context) error {
	pm := config.Instance.Polymarket
	if !pm.Enabled {
		slog.Debug("polymarket tracking disabled, skip")
		return nil
	}
	db := sqls.DB()
	now := dates.NowTimestamp()
	batchSize := polymarketTrackingBatchSize(pm)
	client := polymarket.NewGammaClient(pm.BaseURL)

	var rows []models.PredictMarketTracking
	if err := db.Where("tracking_status IN ? AND next_retry_at <= ?",
		[]string{trackingStatusDiscovered, trackingStatusTracking}, now).
		Order("last_sync_at asc, id asc").
		Limit(batchSize).
		Find(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		row := rows[i]
		m, err := client.GetMarketByID(ctx, row.ExternalMarketId)
		if err != nil {
			s.markTrackingFailure(db, &row, err, now)
			continue
		}
		if err := s.upsertMarketAndContext(db, m, []string{"polymarket"}, "tracking", now); err != nil {
			s.markTrackingFailure(db, &row, err, now)
			continue
		}
		s.markTrackingSuccess(db, row.ExternalMarketId, now)
	}
	slog.Info("polymarket tracking done", slog.Int("count", len(rows)))
	return nil
}

func (s *polymarketSyncService) syncMarketsPaged(ctx context.Context, db *gorm.DB, client *polymarket.GammaClient, params map[string]string, baseTags []string, source string, now int64, pageSize int, seenMarkets, upsertOK, upsertFailed *int) error {
	listStart := time.Now()
	slog.Info("polymarket list markets start",
		slog.String("source", source),
		slog.Int("pageSize", pageSize),
		slog.Any("params", params),
	)
	list, _, err := client.ListMarketsKeyset(ctx, pageSize, "", params)
	if err != nil {
		slog.Error("polymarket list markets failed", slog.Any("err", err), slog.Any("params", params))
		return err
	}
	slog.Info("polymarket list markets done",
		slog.String("source", source),
		slog.Int("markets", len(list)),
		slog.Int64("apiCostMs", time.Since(listStart).Milliseconds()),
		slog.Any("params", params),
	)
	for i := range list {
		m := list[i]
		if seenMarkets != nil {
			(*seenMarkets)++
		}
		upsertStart := time.Now()
		if err := s.upsertMarketAndContext(db, &m, baseTags, source, now); err != nil {
			if upsertFailed != nil {
				(*upsertFailed)++
			}
			slog.Error("polymarket market upsert failed",
				slog.String("source", source),
				slog.String("marketId", anyToString(m.ID)),
				slog.String("slug", strings.TrimSpace(m.Slug)),
				slog.Int64("upsertCostMs", time.Since(upsertStart).Milliseconds()),
				slog.Any("err", err),
			)
			return err
		}
		if upsertOK != nil {
			(*upsertOK)++
		}
		slog.Info("polymarket market upsert done",
			slog.String("source", source),
			slog.String("marketId", anyToString(m.ID)),
			slog.String("slug", strings.TrimSpace(m.Slug)),
			slog.Int64("upsertCostMs", time.Since(upsertStart).Milliseconds()),
		)
	}
	return nil
}

func discoveryOpenOnlyParams(extra map[string]string) map[string]string {
	params := map[string]string{
		"active": "true",
	}
	for k, v := range extra {
		params[k] = v
	}
	return params
}

func (s *polymarketSyncService) listEnabledDiscoveryTags(db *gorm.DB) ([]models.PolymarketDiscoveryTag, error) {
	var tags []models.PolymarketDiscoveryTag
	err := db.Where("enabled = ?", true).
		Order("rank asc, id asc").
		Find(&tags).Error
	return tags, err
}

func (s *polymarketSyncService) resolveDiscoveryTags(ctx context.Context, client *polymarket.GammaClient, tags []models.PolymarketDiscoveryTag) ([]models.PolymarketDiscoveryTag, error) {
	out := make([]models.PolymarketDiscoveryTag, len(tags))
	copy(out, tags)

	needResolve := false
	for _, t := range out {
		if t.ExternalTagId <= 0 {
			needResolve = true
			break
		}
	}
	if !needResolve {
		return out, nil
	}

	gammaTags, err := client.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	lookup := map[string]int64{}
	for _, g := range gammaTags {
		id := anyToInt64(g.ID)
		if id <= 0 {
			continue
		}
		slug := strings.ToLower(strings.TrimSpace(g.Slug))
		name := strings.ToLower(strings.TrimSpace(g.Name))
		if slug != "" {
			lookup[slug] = id
		}
		if name != "" {
			lookup[name] = id
		}
		lookup[strconv.FormatInt(id, 10)] = id
	}

	for i := range out {
		if out[i].ExternalTagId > 0 {
			continue
		}
		if id, ok := lookup[strings.ToLower(strings.TrimSpace(out[i].Slug))]; ok {
			out[i].ExternalTagId = id
			continue
		}
		if id, ok := lookup[strings.ToLower(strings.TrimSpace(out[i].Name))]; ok {
			out[i].ExternalTagId = id
		}
	}
	return out, nil
}

func (s *polymarketSyncService) upsertMarketAndContext(db *gorm.DB, m *polymarket.Market, baseTags []string, source string, now int64) error {
	if m == nil {
		return nil
	}
	externalMarketID := anyToString(m.ID)
	if externalMarketID == "" {
		return nil
	}
	title := polymarketTitle(m, externalMarketID)
	closeTs := parseGammaTimeToUnix(m.CloseDate)
	if closeTs <= 0 {
		closeTs = parseGammaTimeToUnix(m.EndDate)
	}
	nowSec := now / 1000
	resolved, outID, outName, resolvedAt := extractResolution(m)
	if resolved && resolvedAt <= 0 {
		resolvedAt = nowSec
	}
	status := "OPEN"
	if resolved || (closeTs > 0 && closeTs <= nowSec) || m.Closed {
		status = "CLOSED"
	}

	return db.Transaction(func(tx *gorm.DB) error {
		market, err := s.upsertPredictMarket(tx, m, externalMarketID, title, status, closeTs, resolved, outID, outName, resolvedAt, now)
		if err != nil {
			return err
		}
		if err := s.upsertPredictContext(tx, market.Id, title, m, baseTags, now); err != nil {
			return err
		}
		if err := s.upsertTracking(tx, market.Id, externalMarketID, m.Slug, source, now); err != nil {
			return err
		}
		if err := s.upsertOutcomes(tx, market.Id, m, now); err != nil {
			return err
		}
		if resolved {
			return s.autoSettleResolvedMarket(tx, market, m, outID, outName, now)
		}
		return nil
	})
}

func (s *polymarketSyncService) upsertPredictMarket(tx *gorm.DB, m *polymarket.Market, externalMarketID, title, status string, closeTs int64, resolved bool, outID, outName string, resolvedAt, now int64) (*models.PredictMarket, error) {
	externalMarketIDNum := anyToInt64(m.ID)
	if externalMarketIDNum <= 0 {
		externalMarketIDNum = now
	}
	market := &models.PredictMarket{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("source_model = ? AND source_model_id = ?", polymarketSourceModel, externalMarketIDNum).
		First(market).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		market.SourceModel = polymarketSourceModel
		market.SourceModelId = externalMarketIDNum
		market.ExternalKey = externalMarketID
		market.MarketType = PredictMarketTypeBinary
		market.Title = title
		market.Status = status
		market.CloseTime = closeTs
		market.Resolved = resolved
		market.ResolvedOutcomeId = outID
		market.ResolvedOutcomeName = outName
		market.ResolvedAt = resolvedAt
		market.CreateTime = now
		market.UpdateTime = now
		return market, tx.Create(market).Error
	}
	market.Title = title
	if market.Status != "SETTLED" {
		market.Status = status
	}
	if closeTs > 0 {
		market.CloseTime = closeTs
	}
	if market.ExternalKey == "" {
		market.ExternalKey = externalMarketID
	}
	if resolved {
		market.Resolved = true
		market.ResolvedOutcomeId = outID
		market.ResolvedOutcomeName = outName
		market.ResolvedAt = resolvedAt
	}
	market.UpdateTime = now
	return market, tx.Save(market).Error
}

func (s *polymarketSyncService) upsertPredictContext(tx *gorm.DB, marketId int64, title string, m *polymarket.Market, baseTags []string, now int64) error {
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
	proText, conText := polymarketBinaryOutcomeTexts(m)
	ctxModel := &models.PredictContext{}
	err := tx.Where("market_id = ?", marketId).First(ctxModel).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		ctxModel.MarketId = marketId
		ctxModel.EventName = eventName
		ctxModel.ProText = proText
		ctxModel.ConText = conText
		ctxModel.Tags = strings.Join(tags, ",")
		ctxModel.CreateTime = now
		ctxModel.UpdateTime = now
		return tx.Create(ctxModel).Error
	}
	ctxModel.EventName = eventName
	if proText != "" {
		ctxModel.ProText = proText
	}
	if conText != "" {
		ctxModel.ConText = conText
	}
	ctxModel.Tags = strings.Join(tags, ",")
	ctxModel.UpdateTime = now
	return tx.Save(ctxModel).Error
}

func (s *polymarketSyncService) upsertTracking(tx *gorm.DB, marketId int64, externalMarketID, slug, source string, now int64) error {
	if source == "" {
		source = "tag"
	}
	row := &models.PredictMarketTracking{}
	err := tx.Where("external_market_id = ?", externalMarketID).First(row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row.MarketId = marketId
		row.ExternalMarketId = externalMarketID
		row.ExternalSlug = strings.TrimSpace(slug)
		row.Source = source
		row.TrackingStatus = trackingStatusTracking
		row.NextRetryAt = 0
		row.CreateTime = now
		row.UpdateTime = now
		return tx.Create(row).Error
	}
	row.MarketId = marketId
	if strings.TrimSpace(slug) != "" {
		row.ExternalSlug = strings.TrimSpace(slug)
	}
	if row.TrackingStatus == "" || row.TrackingStatus == trackingStatusDiscovered {
		row.TrackingStatus = trackingStatusTracking
	}
	row.UpdateTime = now
	return tx.Save(row).Error
}

func (s *polymarketSyncService) upsertOutcomes(tx *gorm.DB, marketId int64, m *polymarket.Market, now int64) error {
	if len(m.Outcomes) == 0 {
		return nil
	}
	for i, outcome := range m.Outcomes {
		outcomeID := anyToString(outcome.ID)
		name := strings.TrimSpace(outcome.Name)
		if outcomeID == "" {
			outcomeID = name
		}
		if name == "" {
			name = outcomeID
		}
		if outcomeID == "" {
			continue
		}
		tokenID := ""
		if i < len(m.ClobTokenIds) {
			tokenID = strings.TrimSpace(m.ClobTokenIds[i])
		}
		option := defaultOutcomeOption(len(m.Outcomes), i)
		row := &models.PredictMarketOutcome{}
		err := tx.Where("market_id = ? AND external_outcome_id = ?", marketId, outcomeID).First(row).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if option == "" {
				continue
			}
			row.MarketId = marketId
			row.ExternalOutcomeId = outcomeID
			row.Option = option
			row.ExternalOutcomeName = name
			row.ExternalTokenId = tokenID
			row.DisplayName = name
			row.Sort = i
			row.CreateTime = now
			row.UpdateTime = now
			if err := tx.Create(row).Error; err != nil {
				return err
			}
			continue
		}
		row.ExternalOutcomeName = name
		row.ExternalTokenId = tokenID
		row.DisplayName = name
		row.Sort = i
		row.UpdateTime = now
		if err := tx.Save(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *polymarketSyncService) autoSettleResolvedMarket(tx *gorm.DB, market *models.PredictMarket, m *polymarket.Market, outID, outName string, now int64) error {
	if market.Status == "SETTLED" {
		return s.resolveOpenIssues(tx, market.Id, now)
	}
	if isCancelledOutcome(outID, outName, m.Resolution) {
		if err := s.recordSettleIssue(tx, market.Id, anyToString(m.ID), issueCancelledOrInvalid, m.Resolution, m, now); err != nil {
			return err
		}
		return s.updateTrackingStatus(tx, anyToString(m.ID), trackingStatusOpsPending, now)
	}
	if config.Instance.Polymarket.AutoSettleEnabled != nil && !*config.Instance.Polymarket.AutoSettleEnabled {
		return nil
	}
	winner, reason := detectExternalWinner(m, outID, outName)
	if reason != "" {
		if err := s.recordSettleIssue(tx, market.Id, anyToString(m.ID), reason, m.Resolution, m, now); err != nil {
			return err
		}
		return s.updateTrackingStatus(tx, anyToString(m.ID), trackingStatusOpsPending, now)
	}
	option, reason := s.mapWinnerToOption(tx, market.Id, winner)
	if reason != "" {
		if err := s.recordSettleIssue(tx, market.Id, anyToString(m.ID), reason, m.Resolution, m, now); err != nil {
			return err
		}
		return s.updateTrackingStatus(tx, anyToString(m.ID), trackingStatusOpsPending, now)
	}
	if config.Instance.Polymarket.DryRun {
		return nil
	}
	market.Status = "SETTLED"
	market.Result = option
	market.Resolved = true
	if market.ResolvedOutcomeId == "" {
		market.ResolvedOutcomeId = winner.ID
	}
	if market.ResolvedOutcomeName == "" {
		market.ResolvedOutcomeName = winner.Name
	}
	if market.ResolvedAt == 0 {
		market.ResolvedAt = now
	}
	market.UpdateTime = now
	if err := tx.Save(market).Error; err != nil {
		return err
	}
	if err := s.resolveOpenIssues(tx, market.Id, now); err != nil {
		return err
	}
	return s.updateTrackingStatus(tx, anyToString(m.ID), trackingStatusSettled, now)
}

type externalWinner struct {
	ID      string
	Name    string
	TokenID string
}

func detectExternalWinner(m *polymarket.Market, outID, outName string) (externalWinner, string) {
	if strings.TrimSpace(outID) != "" || strings.TrimSpace(outName) != "" {
		return externalWinner{ID: strings.TrimSpace(outID), Name: strings.TrimSpace(outName)}, ""
	}
	winners := make([]externalWinner, 0, 1)
	for i, o := range m.Outcomes {
		if !o.Winner {
			continue
		}
		tokenID := ""
		if i < len(m.ClobTokenIds) {
			tokenID = strings.TrimSpace(m.ClobTokenIds[i])
		}
		winners = append(winners, externalWinner{ID: anyToString(o.ID), Name: strings.TrimSpace(o.Name), TokenID: tokenID})
	}
	if len(winners) == 1 {
		return winners[0], ""
	}
	if len(winners) > 1 {
		return externalWinner{}, issueAmbiguousOutcome
	}
	return externalWinner{}, issueNoWinner
}

func (s *polymarketSyncService) mapWinnerToOption(tx *gorm.DB, marketId int64, winner externalWinner) (string, string) {
	var rows []models.PredictMarketOutcome
	q := tx.Where("market_id = ?", marketId)
	conds := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if winner.ID != "" {
		conds = append(conds, "external_outcome_id = ?")
		args = append(args, winner.ID)
	}
	if winner.Name != "" {
		conds = append(conds, "lower(external_outcome_name) = lower(?)")
		args = append(args, winner.Name)
	}
	if winner.TokenID != "" {
		conds = append(conds, "external_token_id = ?")
		args = append(args, winner.TokenID)
	}
	if len(conds) == 0 {
		return "", issueNoWinner
	}
	if err := q.Where(strings.Join(conds, " OR "), args...).Find(&rows).Error; err != nil {
		return "", issueNoOutcomeMapping
	}
	if len(rows) == 0 {
		return "", issueNoOutcomeMapping
	}
	option := strings.ToUpper(strings.TrimSpace(rows[0].Option))
	for _, row := range rows {
		if strings.ToUpper(strings.TrimSpace(row.Option)) != option {
			return "", issueAmbiguousOutcome
		}
	}
	if option != PredictOptionA && option != PredictOptionB {
		return "", issueNoOutcomeMapping
	}
	return option, ""
}

func (s *polymarketSyncService) recordSettleIssue(tx *gorm.DB, marketId int64, externalMarketID, reason, rawResolution string, m *polymarket.Market, now int64) error {
	payload := ""
	if b, err := json.Marshal(map[string]any{
		"id":         anyToString(m.ID),
		"slug":       m.Slug,
		"resolved":   m.Resolved,
		"resolution": m.Resolution,
		"outcomes":   m.Outcomes,
	}); err == nil {
		payload = string(b)
		if len(payload) > 4000 {
			payload = payload[:4000]
		}
	}
	row := &models.PredictMarketSettleIssue{}
	err := tx.Where("market_id = ? AND reason = ? AND status = ?", marketId, reason, issueStatusOpen).First(row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row.MarketId = marketId
		row.ExternalMarketId = externalMarketID
		row.Reason = reason
		row.RawResolution = rawResolution
		row.RawPayload = payload
		row.Status = issueStatusOpen
		row.CreateTime = now
		row.UpdateTime = now
		return tx.Create(row).Error
	}
	row.RawResolution = rawResolution
	row.RawPayload = payload
	row.UpdateTime = now
	return tx.Save(row).Error
}

func (s *polymarketSyncService) resolveOpenIssues(tx *gorm.DB, marketId int64, now int64) error {
	return tx.Model(&models.PredictMarketSettleIssue{}).
		Where("market_id = ? AND status = ?", marketId, issueStatusOpen).
		Updates(map[string]any{"status": issueStatusResolved, "update_time": now}).Error
}

func (s *polymarketSyncService) updateTrackingStatus(tx *gorm.DB, externalMarketID, status string, now int64) error {
	return tx.Model(&models.PredictMarketTracking{}).
		Where("external_market_id = ?", externalMarketID).
		Updates(map[string]any{
			"tracking_status": status,
			"last_sync_at":    now,
			"update_time":     now,
		}).Error
}

func (s *polymarketSyncService) markTrackingSuccess(db *gorm.DB, externalMarketID string, now int64) {
	_ = db.Model(&models.PredictMarketTracking{}).
		Where("external_market_id = ? AND tracking_status IN ?", externalMarketID, []string{trackingStatusDiscovered, trackingStatusTracking}).
		Updates(map[string]any{
			"tracking_status": trackingStatusTracking,
			"last_sync_at":    now,
			"next_retry_at":   0,
			"fail_count":      0,
			"last_error":      "",
			"update_time":     now,
		}).Error
}

func (s *polymarketSyncService) markTrackingFailure(db *gorm.DB, row *models.PredictMarketTracking, err error, now int64) {
	maxRetry := polymarketMaxRetry(config.Instance.Polymarket)
	failCount := row.FailCount + 1
	status := trackingStatusTracking
	if failCount >= maxRetry {
		status = trackingStatusOpsPending
	}
	delay := polymarketRetryBaseSeconds(config.Instance.Polymarket) * failCount
	msg := err.Error()
	if len(msg) > 512 {
		msg = msg[:512]
	}
	_ = db.Model(&models.PredictMarketTracking{}).
		Where("id = ?", row.Id).
		Updates(map[string]any{
			"tracking_status": status,
			"fail_count":      failCount,
			"last_error":      msg,
			"next_retry_at":   now + int64(delay),
			"update_time":     now,
		}).Error
}

func polymarketTitle(m *polymarket.Market, externalMarketID string) string {
	title := strings.TrimSpace(m.Question)
	if title == "" {
		title = strings.TrimSpace(m.Title)
	}
	if title == "" {
		title = "Polymarket Market " + externalMarketID
	}
	return title
}

func defaultOutcomeOption(count, idx int) string {
	if count != 2 {
		return ""
	}
	if idx == 0 {
		return PredictOptionA
	}
	if idx == 1 {
		return PredictOptionB
	}
	return ""
}

func polymarketBinaryOutcomeTexts(m *polymarket.Market) (proText, conText string) {
	if m == nil || len(m.Outcomes) != 2 {
		return "", ""
	}
	return polymarketOutcomeDisplayName(m.Outcomes[0]), polymarketOutcomeDisplayName(m.Outcomes[1])
}

func polymarketOutcomeDisplayName(outcome polymarket.Outcome) string {
	name := strings.TrimSpace(outcome.Name)
	if name == "" {
		name = anyToString(outcome.ID)
	}
	if name == "" {
		name = strings.TrimSpace(outcome.Slug)
	}
	return name
}

func isCancelledOutcome(vals ...string) bool {
	for _, v := range vals {
		v = strings.ToUpper(strings.TrimSpace(v))
		if v == "CANCELLED" || v == "CANCELED" || v == "INVALID" || v == "__CANCELLED__" {
			return true
		}
	}
	return false
}

func parseGammaTimeToUnix(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}
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
	res := strings.TrimSpace(m.Resolution)
	if isCancelledOutcome(res) {
		return true, "__CANCELLED__", "CANCELLED", resolvedAt
	}
	if res != "" {
		return resolved || m.Closed, res, res, resolvedAt
	}
	return resolved, "", "", resolvedAt
}

func polymarketTrackingBatchSize(pm config.Polymarket) int {
	if pm.TrackingBatchSize > 0 {
		return pm.TrackingBatchSize
	}
	return 50
}

func polymarketMaxRetry(pm config.Polymarket) int {
	if pm.MaxRetry > 0 {
		return pm.MaxRetry
	}
	return 5
}

func polymarketRetryBaseSeconds(pm config.Polymarket) int {
	if pm.RetryBaseSeconds > 0 {
		return pm.RetryBaseSeconds
	}
	return 60
}

func anyToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
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
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
