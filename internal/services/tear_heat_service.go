package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
	"errors"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

const (
	tearHeatCommentBase = 3.0
	tearHeatCommentCap  = 20.0
	tearHeatCoinFactor  = 0.02
	tearHeatLive        = "LIVE"

	TearBusinessTypeDark  = 1
	TearBusinessTypeArena = 2
)

var TearHeatService = newTearHeatService()

func newTearHeatService() *tearHeatService {
	return &tearHeatService{}
}

type tearHeatService struct{}

type TearHeatOptionSummary struct {
	Option   string  `json:"option"`
	Side     string  `json:"side"`
	HLike    float64 `json:"hLike"`
	HComment float64 `json:"hComment"`
	HCoin    float64 `json:"hCoin"`
	HTotal   float64 `json:"hTotal"`
}

type TearHeatSummary struct {
	MarketId int64                   `json:"marketId"`
	Options  []TearHeatOptionSummary `json:"options"`
	My       []models.TearEventHeat  `json:"my,omitempty"`
}

func (s *tearHeatService) RefreshBetHeat(tx *gorm.DB, marketId, userId int64, option string) error {
	return s.RefreshUserOption(tx, marketId, userId, option)
}

func (s *tearHeatService) RefreshCommentHeat(tx *gorm.DB, marketId, userId int64, option string) error {
	return s.RefreshUserOption(tx, marketId, userId, option)
}

// AddPKHeat writes one action snapshot and updates cumulative event heat.
func (s *tearHeatService) AddPKHeat(tx *gorm.DB, roundId, userId int64, side string, businessType int, isLike, isComment, isBet bool, amount float64) error {
	if tx == nil {
		tx = sqls.DB()
	}
	if roundId <= 0 || userId <= 0 {
		return nil
	}
	side = strings.ToUpper(strings.TrimSpace(side))
	if side == "" {
		return nil
	}
	if businessType <= 0 {
		businessType = TearBusinessTypeArena
	}

	hLike, hComment, hCoin := calcTearActionHeat(isLike, isComment, isBet, amount)
	if hLike == 0 && hComment == 0 && hCoin == 0 {
		return nil
	}

	now := dates.NowTimestamp()
	snapshot := &models.TearHeatSnapshot{
		EventId:      roundId,
		UserId:       userId,
		Option:       side,
		BusinessType: businessType,
		HLike:        hLike,
		HComment:     hComment,
		HCoin:        hCoin,
		SnapshotTime: now,
		SnapshotType: tearHeatLive,
		CreateTime:   now,
		UpdateTime:   now,
	}
	if err := repositories.TearHeatRepository.CreateSnapshot(tx, snapshot); err != nil {
		return err
	}
	return TearEventHeatService.AddHeat(tx, roundId, userId, side, businessType, hLike, hComment, hCoin)
}

func (s *tearHeatService) RefreshByLikedComment(tx *gorm.DB, commentId int64, userId int64) error {
	if tx == nil {
		tx = sqls.DB()
	}
	pkMeta := repositories.PKRepository.TakeCommentMeta(tx, "comment_id = ?", commentId)
	if pkMeta != nil {
		return s.AddPKHeat(tx, pkMeta.RoundId, userId, pkMeta.Side, TearBusinessTypeArena, true, false, false, 0)
	}

	meta, err := s.findCommentMeta(tx, commentId)
	if err != nil || meta == nil {
		return err
	}
	return s.RefreshUserOption(tx, meta.MarketId, meta.UserId, meta.Option)
}

func (s *tearHeatService) RefreshUserOption(tx *gorm.DB, marketId, userId int64, option string) error {
	if tx == nil {
		tx = sqls.DB()
	}
	if marketId <= 0 || userId <= 0 {
		return nil
	}
	option = strings.ToUpper(strings.TrimSpace(option))
	if option == "" {
		return nil
	}

	hLike, err := s.sumLikeHeat(tx, marketId, userId, option)
	if err != nil {
		return err
	}
	hComment, err := s.sumCommentHeat(tx, marketId, userId, option)
	if err != nil {
		return err
	}
	hCoin, err := s.sumCoinHeat(tx, marketId, userId, option)
	if err != nil {
		return err
	}

	return TearEventHeatService.SetHeat(tx, marketId, userId, option, TearBusinessTypeDark, hLike, hComment, hCoin)
}

func (s *tearHeatService) GetSummary(marketId, currentUserId int64) (*TearHeatSummary, error) {
	if marketId <= 0 {
		return nil, errors.New("marketId is required")
	}
	list, err := repositories.TearEventHeatRepository.FindByEvent(sqls.DB(), marketId, 0)
	if err != nil {
		return nil, err
	}

	optionMap := make(map[string]*TearHeatOptionSummary)
	for _, item := range list {
		key := item.Option
		if key == "" {
			continue
		}
		summary := optionMap[key]
		if summary == nil {
			summary = &TearHeatOptionSummary{Option: key, Side: key}
			optionMap[key] = summary
		}
		summary.HLike += item.HLike
		summary.HComment += item.HComment
		summary.HCoin += item.HCoin
		summary.HTotal += item.HTotal
	}

	options := make([]TearHeatOptionSummary, 0, len(optionMap))
	for _, key := range []string{PredictOptionA, PredictOptionB, PredictOptionDraw} {
		if summary := optionMap[key]; summary != nil {
			options = append(options, *summary)
			delete(optionMap, key)
		}
	}
	for _, summary := range optionMap {
		options = append(options, *summary)
	}

	ret := &TearHeatSummary{MarketId: marketId, Options: options}
	if currentUserId > 0 {
		my, err := repositories.TearEventHeatRepository.FindByUser(sqls.DB(), marketId, currentUserId, 0)
		if err != nil {
			return nil, err
		}
		ret.My = my
	}
	return ret, nil
}

func (s *tearHeatService) sumLikeHeat(tx *gorm.DB, marketId, userId int64, option string) (float64, error) {
	var count int64
	err := tx.Table("t_user_like AS ul").
		Joins("JOIN t_predict_comment_meta AS pcm ON pcm.comment_id = ul.entity_id").
		Where("ul.entity_type = ?", constants.EntityComment).
		Where("pcm.market_id = ? AND pcm.user_id = ? AND pcm.option = ?", marketId, userId, option).
		Count(&count).Error
	return float64(count), err
}

func (s *tearHeatService) sumCommentHeat(tx *gorm.DB, marketId, userId int64, option string) (float64, error) {
	var comments []models.Comment
	if err := tx.Table("t_comment AS c").
		Select("c.id, c.like_count").
		Joins("JOIN t_predict_comment_meta AS pcm ON pcm.comment_id = c.id").
		Where("pcm.market_id = ? AND pcm.user_id = ? AND pcm.option = ?", marketId, userId, option).
		Where("c.status = ?", constants.StatusOk).
		Scan(&comments).Error; err != nil {
		return 0, err
	}

	total := 0.0
	for _, comment := range comments {
		total += calcTearCommentHeat(comment.LikeCount)
	}
	return total, nil
}

func (s *tearHeatService) sumCoinHeat(tx *gorm.DB, marketId, userId int64, option string) (float64, error) {
	var total int64
	err := tx.Model(&models.PredictBet{}).
		Where("market_id = ? AND user_id = ? AND option = ?", marketId, userId, option).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	return float64(total) * tearHeatCoinFactor, err
}

func (s *tearHeatService) findCommentMeta(tx *gorm.DB, commentId int64) (*models.PredictCommentMeta, error) {
	meta := &models.PredictCommentMeta{}
	if err := tx.Take(meta, "comment_id = ?", commentId).Error; err == nil {
		return meta, nil
	}

	comment := repositories.CommentRepository.Get(tx, commentId)
	if comment == nil || comment.EntityType != constants.EntityComment || comment.EntityId <= 0 {
		return nil, nil
	}
	if err := tx.Take(meta, "comment_id = ?", comment.EntityId).Error; err != nil {
		return nil, nil
	}
	return meta, nil
}

func calcTearActionHeat(isLike, isComment, isBet bool, amount float64) (float64, float64, float64) {
	hLike := 0.0
	if isLike {
		hLike = 1
	}
	hComment := 0.0
	if isComment {
		hComment = 2
	}
	hCoin := 0.0
	if isBet && amount > 0 {
		hCoin = amount * tearHeatCoinFactor
	}
	return hLike, hComment, hCoin
}

func calcTearCommentHeat(likeCount int64) float64 {
	if likeCount < 0 {
		likeCount = 0
	}
	heat := tearHeatCommentBase * (1 + math.Log(1+float64(likeCount)))
	if heat > tearHeatCommentCap {
		return tearHeatCommentCap
	}
	return heat
}
