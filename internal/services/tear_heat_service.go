package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var TearHeatService = newTearHeatService()

type tearHeatService struct{}

type TearHeatSnapshotQuery struct {
	EventType    string
	EventId      int64
	RoundId      int64
	SnapshotType string
}

type TearHeatSnapshotComputeRequest struct {
	EventType    string
	EventId      int64
	TopicId      int64
	RoundId      int64
	SnapshotType string
	FreezeSource string
}

type TearHeatUserItem struct {
	UserId      int64
	Option      string
	CommentHeat float64
	LikeHeat    float64
	CoinHeat    float64
	TotalHeat   float64
}

type PredictUserHeatStats struct {
	ActionCount       int64
	CommentCount      int64
	ReplyCount        int64
	LikeCount         int64
	ReceivedLikeCount int64
	BetAmount         int64
}

func newTearHeatService() *tearHeatService {
	return &tearHeatService{}
}

func (s *tearHeatService) GetHeatSnapshot(db *gorm.DB, query TearHeatSnapshotQuery) ([]map[string]any, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	if query.EventType == "" {
		return nil, errors.New("eventType is required")
	}
	if query.EventId <= 0 || query.RoundId < 0 {
		return nil, errors.New("eventId and roundId are required")
	}

	if query.SnapshotType == "" {
		query.SnapshotType = TearSnapshotCkpt
	}

	var rows []models.TearHeatSnapshot
	if err := db.Where("event_type = ? AND event_id = ? AND round_id = ? AND snapshot_type = ?", query.EventType, query.EventId, query.RoundId, query.SnapshotType).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	ret := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		ret = append(ret, map[string]any{
			"option":       row.Option,
			"hLike":        row.HLike,
			"hComment":     row.HComment,
			"hCoin":        row.HCoin,
			"hTotal":       row.HTotal,
			"snapshotType": row.SnapshotType,
			"snapshotTime": row.SnapshotTime,
		})
	}
	return ret, nil
}

func (s *tearHeatService) ComputeHeatSnapshot(tx *gorm.DB, req TearHeatSnapshotComputeRequest) ([]map[string]any, int64, error) {
	if tx == nil {
		return nil, 0, errors.New("tx is required")
	}
	if req.EventType == "" {
		return nil, 0, errors.New("eventType is required")
	}
	if req.EventId <= 0 || req.TopicId <= 0 || req.RoundId < 0 {
		return nil, 0, errors.New("eventId/topicId/roundId are required")
	}
	if req.SnapshotType == "" {
		req.SnapshotType = TearSnapshotCkpt
	}
	if req.FreezeSource == "" {
		req.FreezeSource = "ON_DEMAND"
	}

	if req.EventType != TearEventTypePK && req.EventType != constants.EntityPredictMarket {
		return nil, 0, errors.New("unsupported event type")
	}

	if req.SnapshotType == TearSnapshotSettle {
		exist, err := s.GetHeatSnapshot(tx, TearHeatSnapshotQuery{
			EventType:    req.EventType,
			EventId:      req.EventId,
			RoundId:      req.RoundId,
			SnapshotType: TearSnapshotSettle,
		})
		if err == nil && len(exist) > 0 {
			last := int64(0)
			for _, item := range exist {
				if v, ok := item["snapshotTime"].(int64); ok && v > last {
					last = v
				}
			}
			return exist, last, nil
		}
	}

	options := []string{PKSideA, PKSideB}
	if req.EventType == constants.EntityPredictMarket {
		// 预测市场兼容 DRAW（对于 binary 市场不会产生有效数据）。
		options = []string{PredictOptionA, PredictOptionB, PredictOptionDraw}
	}

	likeHeatByOption := map[string]float64{}
	for _, opt := range options {
		likeHeatByOption[opt] = 0
	}
	type optionAgg struct {
		Option string  `gorm:"column:option_at_action"`
		Cnt    float64 `gorm:"column:cnt"`
	}
	var likeAgg []optionAgg
	_ = tx.Table("t_tear_interact_log").
		Select("option_at_action, COUNT(1) AS cnt").
		Where("event_type = ? AND topic_id = ? AND round_id = ? AND action_type = ?", req.EventType, req.TopicId, req.RoundId, "like").
		Group("option_at_action").
		Find(&likeAgg).Error
	for _, item := range likeAgg {
		opt := strings.ToUpper(strings.TrimSpace(item.Option))
		if req.EventType == TearEventTypePK {
			opt = normalizePKSide(opt)
		}
		if _, ok := likeHeatByOption[opt]; ok {
			likeHeatByOption[opt] = item.Cnt
		}
	}

	commentHeatByOption := map[string]float64{}
	for _, opt := range options {
		commentHeatByOption[opt] = 0
	}
	if req.EventType == TearEventTypePK {
		var metas []models.PKCommentMeta
		if err := tx.Where("topic_id = ? AND round_id = ?", req.TopicId, req.RoundId).Find(&metas).Error; err != nil {
			return nil, 0, err
		}
		for _, meta := range metas {
			comment := repositories.CommentRepository.Get(tx, meta.CommentId)
			if comment == nil || comment.Status != constants.StatusOk {
				continue
			}
			heat := math.Min(3*(1+math.Log(1+float64(comment.LikeCount))), 20)
			opt := normalizePKSide(meta.Side)
			if _, ok := commentHeatByOption[opt]; ok {
				commentHeatByOption[opt] += heat
			}
		}
	} else {
		likeCntByComment := map[int64]float64{}
		type commentLikeAgg struct {
			CommentId int64   `gorm:"column:entity_id"`
			Cnt       float64 `gorm:"column:cnt"`
		}
		var likeRows []commentLikeAgg
		_ = tx.Table("t_tear_interact_log").
			Select("entity_id, COUNT(1) AS cnt").
			Where("event_type = ? AND topic_id = ? AND round_id = ? AND action_type = ? AND entity_type = ?", constants.EntityPredictMarket, req.TopicId, req.RoundId, "like", constants.EntityComment).
			Group("entity_id").
			Find(&likeRows).Error
		for _, row := range likeRows {
			likeCntByComment[row.CommentId] = row.Cnt
		}

		var metas []models.PredictCommentMeta
		if err := tx.Where("market_id = ?", req.TopicId).Find(&metas).Error; err != nil {
			return nil, 0, err
		}
		for _, meta := range metas {
			heat := math.Min(3*(1+math.Log(1+likeCntByComment[meta.CommentId])), 20)
			opt := strings.ToUpper(strings.TrimSpace(meta.Option))
			if _, ok := commentHeatByOption[opt]; ok {
				commentHeatByOption[opt] += heat
			}
		}
	}

	coinHeatByOption := map[string]float64{}
	for _, opt := range options {
		coinHeatByOption[opt] = 0
	}
	type coinAgg struct {
		Option string  `gorm:"column:bet_option"`
		Coin   float64 `gorm:"column:coin"`
	}
	var coins []coinAgg
	if req.EventType == constants.EntityPredictMarket {
		type predictCoinAgg struct {
			Option string  `gorm:"column:option"`
			Coin   float64 `gorm:"column:coin"`
		}
		var predictCoins []predictCoinAgg
		_ = tx.Table("t_predict_bet").
			Select("option, COALESCE(SUM(amount) * 0.02, 0) AS coin").
			Where("market_id = ?", req.TopicId).
			Group("option").
			Find(&predictCoins).Error
		for _, item := range predictCoins {
			coins = append(coins, coinAgg{Option: item.Option, Coin: item.Coin})
		}
	} else {
		_ = tx.Table("t_tear_user_event_stat").
			Select("bet_option, COALESCE(SUM(bet_amount) * 0.02, 0) AS coin").
			Where("event_type = ? AND topic_id = ? AND round_id = ?", req.EventType, req.TopicId, req.RoundId).
			Group("bet_option").
			Find(&coins).Error
	}
	for _, item := range coins {
		opt := strings.ToUpper(strings.TrimSpace(item.Option))
		if req.EventType == TearEventTypePK {
			opt = normalizePKSide(opt)
		}
		if _, ok := coinHeatByOption[opt]; ok {
			coinHeatByOption[opt] = item.Coin
		}
	}

	now := dates.NowTimestamp()
	upsertCols := []clause.Column{{Name: "event_type"}, {Name: "event_id"}, {Name: "round_id"}, {Name: "option"}, {Name: "snapshot_type"}}
	for _, opt := range options {
		item := &models.TearHeatSnapshot{
			EventType:    req.EventType,
			EventId:      req.EventId,
			TopicId:      req.TopicId,
			RoundId:      req.RoundId,
			Option:       opt,
			HLike:        likeHeatByOption[opt],
			HComment:     commentHeatByOption[opt],
			HCoin:        coinHeatByOption[opt],
			HTotal:       likeHeatByOption[opt] + commentHeatByOption[opt] + coinHeatByOption[opt],
			SnapshotType: req.SnapshotType,
			FreezeSource: req.FreezeSource,
			SnapshotTime: now,
			CreateTime:   now,
			UpdateTime:   now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: upsertCols,
			DoUpdates: clause.Assignments(map[string]any{
				"h_like":        item.HLike,
				"h_comment":     item.HComment,
				"h_coin":        item.HCoin,
				"h_total":       item.HTotal,
				"freeze_source": item.FreezeSource,
				"snapshot_time": item.SnapshotTime,
				"update_time":   now,
			}),
		}).Create(item).Error; err != nil {
			return nil, 0, err
		}
	}

	snapshots, err := s.GetHeatSnapshot(tx, TearHeatSnapshotQuery{
		EventType:    req.EventType,
		EventId:      req.EventId,
		RoundId:      req.RoundId,
		SnapshotType: req.SnapshotType,
	})
	if err != nil {
		return nil, 0, err
	}
	return snapshots, now, nil
}

func (s *tearHeatService) GetUserHeatItems(tx *gorm.DB, eventType string, topicId, roundId int64) ([]TearHeatUserItem, error) {
	if tx == nil {
		return nil, errors.New("tx is required")
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return nil, errors.New("eventType is required")
	}
	if topicId <= 0 || roundId < 0 {
		return nil, errors.New("topicId/roundId are required")
	}
	if eventType != TearEventTypePK && eventType != constants.EntityPredictMarket {
		return nil, errors.New("unsupported event type")
	}
	if eventType == constants.EntityPredictMarket {
		return s.getPredictUserHeatItems(tx, topicId, roundId)
	}

	var rows []models.TearUserEventStat
	if err := tx.Where("event_type = ? AND topic_id = ? AND round_id = ?", eventType, topicId, roundId).
		Order("user_id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []TearHeatUserItem{}, nil
	}

	items := make([]TearHeatUserItem, 0, len(rows))
	for _, row := range rows {
		option := strings.ToUpper(strings.TrimSpace(row.BetOption))
		if eventType == TearEventTypePK {
			option = normalizePKSide(option)
		}
		if option == "" {
			continue
		}
		coinHeat := float64(row.BetAmount) * 0.02
		items = append(items, TearHeatUserItem{
			UserId:      row.UserId,
			Option:      option,
			CommentHeat: row.HeatContribution,
			LikeHeat:    float64(row.LikeCount),
			CoinHeat:    coinHeat,
			TotalHeat:   row.HeatContribution + float64(row.LikeCount) + coinHeat,
		})
	}
	return items, nil
}

func (s *tearHeatService) getPredictUserHeatItems(tx *gorm.DB, marketId, roundId int64) ([]TearHeatUserItem, error) {
	if roundId != 0 {
		return []TearHeatUserItem{}, nil
	}

	itemsByUser := make(map[int64]*TearHeatUserItem)
	upsert := func(userId int64, option string) *TearHeatUserItem {
		option = strings.ToUpper(strings.TrimSpace(option))
		if userId <= 0 || option == "" {
			return nil
		}
		item, ok := itemsByUser[userId]
		if !ok {
			item = &TearHeatUserItem{UserId: userId, Option: option}
			itemsByUser[userId] = item
		}
		if item.Option == "" {
			item.Option = option
		}
		return item
	}

	type likeRow struct {
		UserId int64   `gorm:"column:user_id"`
		Option string  `gorm:"column:option_at_action"`
		Cnt    float64 `gorm:"column:cnt"`
	}
	var likeRows []likeRow
	_ = tx.Table("t_tear_interact_log").
		Select("user_id, option_at_action, COUNT(1) AS cnt").
		Where("event_type = ? AND topic_id = ? AND round_id = ? AND action_type = ?", constants.EntityPredictMarket, marketId, 0, "like").
		Group("user_id, option_at_action").
		Find(&likeRows).Error
	for _, row := range likeRows {
		item := upsert(row.UserId, row.Option)
		if item != nil {
			item.LikeHeat += row.Cnt
		}
	}

	type coinRow struct {
		UserId int64   `gorm:"column:user_id"`
		Option string  `gorm:"column:option"`
		Coin   float64 `gorm:"column:coin"`
	}
	var coinRows []coinRow
	_ = tx.Table("t_predict_bet").
		Select("user_id, option, COALESCE(SUM(amount) * 0.02, 0) AS coin").
		Where("market_id = ?", marketId).
		Group("user_id, option").
		Find(&coinRows).Error
	for _, row := range coinRows {
		item := upsert(row.UserId, row.Option)
		if item != nil {
			item.CoinHeat += row.Coin
		}
	}

	type commentLikeRow struct {
		CommentId int64   `gorm:"column:entity_id"`
		Cnt       float64 `gorm:"column:cnt"`
	}
	likeCntByComment := map[int64]float64{}
	var commentLikeRows []commentLikeRow
	_ = tx.Table("t_tear_interact_log").
		Select("entity_id, COUNT(1) AS cnt").
		Where("event_type = ? AND topic_id = ? AND round_id = ? AND action_type = ? AND entity_type = ?", constants.EntityPredictMarket, marketId, 0, "like", constants.EntityComment).
		Group("entity_id").
		Find(&commentLikeRows).Error
	for _, row := range commentLikeRows {
		likeCntByComment[row.CommentId] = row.Cnt
	}

	var metas []models.PredictCommentMeta
	if err := tx.Where("market_id = ?", marketId).Find(&metas).Error; err != nil {
		return nil, err
	}
	for _, meta := range metas {
		item := upsert(meta.UserId, meta.Option)
		if item == nil {
			continue
		}
		heat := math.Min(3*(1+math.Log(1+likeCntByComment[meta.CommentId])), 20)
		item.CommentHeat += heat
	}

	ret := make([]TearHeatUserItem, 0, len(itemsByUser))
	for _, item := range itemsByUser {
		item.TotalHeat = item.CommentHeat + item.LikeHeat + item.CoinHeat
		ret = append(ret, *item)
	}
	return ret, nil
}

func (s *tearHeatService) GetPredictUserHeatStats(tx *gorm.DB, marketId, userId int64) (PredictUserHeatStats, error) {
	stats := PredictUserHeatStats{}
	if tx == nil {
		return stats, errors.New("tx is required")
	}
	if marketId <= 0 || userId <= 0 {
		return stats, errors.New("marketId/userId are required")
	}

	type actionRow struct {
		ActionType string `gorm:"column:action_type"`
		Cnt        int64  `gorm:"column:cnt"`
	}
	var actionRows []actionRow
	_ = tx.Table("t_tear_interact_log").
		Select("action_type, COUNT(1) AS cnt").
		Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", constants.EntityPredictMarket, marketId, 0, userId).
		Group("action_type").
		Find(&actionRows).Error
	for _, row := range actionRows {
		stats.ActionCount += row.Cnt
		switch strings.ToLower(strings.TrimSpace(row.ActionType)) {
		case "comment":
			stats.CommentCount += row.Cnt
		case "reply":
			stats.ReplyCount += row.Cnt
		case "like":
			stats.LikeCount += row.Cnt
		}
	}

	var commentIds []int64
	_ = tx.Model(&models.PredictCommentMeta{}).
		Where("market_id = ? AND user_id = ?", marketId, userId).
		Pluck("comment_id", &commentIds).Error
	if len(commentIds) > 0 {
		_ = tx.Table("t_tear_interact_log").
			Where("event_type = ? AND topic_id = ? AND round_id = ? AND action_type = ? AND entity_type = ? AND entity_id IN ?", constants.EntityPredictMarket, marketId, 0, "like", constants.EntityComment, commentIds).
			Count(&stats.ReceivedLikeCount).Error
	}

	_ = tx.Table("t_predict_bet").
		Select("COALESCE(SUM(amount), 0)").
		Where("market_id = ? AND user_id = ?", marketId, userId).
		Scan(&stats.BetAmount).Error

	return stats, nil
}
