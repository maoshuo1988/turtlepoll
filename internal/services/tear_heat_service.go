package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"math"

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
	if query.EventId <= 0 || query.RoundId <= 0 {
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
	if req.EventId <= 0 || req.TopicId <= 0 || req.RoundId <= 0 {
		return nil, 0, errors.New("eventId/topicId/roundId are required")
	}
	if req.SnapshotType == "" {
		req.SnapshotType = TearSnapshotCkpt
	}
	if req.FreezeSource == "" {
		req.FreezeSource = "ON_DEMAND"
	}

	if req.EventType != TearEventTypePK {
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

	likeHeatByOption := map[string]float64{PKSideA: 0, PKSideB: 0}
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
		opt := normalizePKSide(item.Option)
		if opt == PKSideA || opt == PKSideB {
			likeHeatByOption[opt] = item.Cnt
		}
	}

	commentHeatByOption := map[string]float64{PKSideA: 0, PKSideB: 0}
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
		if opt == PKSideA || opt == PKSideB {
			commentHeatByOption[opt] += heat
		}
	}

	coinHeatByOption := map[string]float64{PKSideA: 0, PKSideB: 0}
	type coinAgg struct {
		Option string  `gorm:"column:bet_option"`
		Coin   float64 `gorm:"column:coin"`
	}
	var coins []coinAgg
	_ = tx.Table("t_tear_user_event_stat").
		Select("bet_option, COALESCE(SUM(bet_amount) * 0.02, 0) AS coin").
		Where("event_type = ? AND topic_id = ? AND round_id = ?", req.EventType, req.TopicId, req.RoundId).
		Group("bet_option").
		Find(&coins).Error
	for _, item := range coins {
		opt := normalizePKSide(item.Option)
		if opt == PKSideA || opt == PKSideB {
			coinHeatByOption[opt] = item.Coin
		}
	}

	now := dates.NowTimestamp()
	upsertCols := []clause.Column{{Name: "event_type"}, {Name: "event_id"}, {Name: "round_id"}, {Name: "option"}, {Name: "snapshot_type"}}
	for _, opt := range []string{PKSideA, PKSideB} {
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
