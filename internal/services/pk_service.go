package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/models/req"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var PKService = newPKService()

const (
	PKTopicStatusEnabled  = "enabled"
	PKTopicStatusDisabled = "disabled"

	PKSeasonStatusActive   = "active"
	PKSeasonStatusFinished = "finished"

	PKPhaseBetting  = "betting"
	PKPhaseLocked   = "locked"
	PKPhaseCooldown = "cooldown"
	PKPhaseSettled  = "settled"

	PKSideA    = "A"
	PKSideB    = "B"
	PKSideDraw = "draw"

	PKBetAmount        int64 = 100
	pkRoundBettingSec        = 48 * 3600
	pkRoundLockedSec         = 24 * 3600
	pkRoundCooldownSec       = 10 * 60
	pkSeasonSec              = 30 * 24 * 3600
)

type pkService struct{}

type PKTopicSaveForm struct {
	Id        int64  `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	SideAName string `json:"sideAName"`
	SideBName string `json:"sideBName"`
	Status    string `json:"status"`
	Sort      int    `json:"sort"`
	Cover     string `json:"cover"`
}

type PKBetForm struct {
	TopicId   int64  `json:"topicId"`
	Side      string `json:"side"`
	RequestId string `json:"requestId"`
}

type PKDownvoteForm struct {
	CommentId int64  `json:"commentId"`
	RequestId string `json:"requestId"`
}

func newPKService() *pkService {
	return &pkService{}
}

func (s *pkService) ListTopics(page, pageSize int, userId int64) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := repositories.PKRepository.DB()
	var topics []models.PKTopic
	q := db.Model(&models.PKTopic{}).Where("status = ?", PKTopicStatusEnabled)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if err := q.Order("sort asc, id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&topics).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(topics))
	for i := range topics {
		list = append(list, s.buildTopicItem(db, &topics[i], userId))
	}
	return map[string]any{"list": list, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) TopicDetail(topicId int64, slug string, userId int64) (map[string]any, error) {
	topic, err := s.findTopic(topicId, slug)
	if err != nil {
		return nil, err
	}
	db := repositories.PKRepository.DB()
	item := s.buildTopicItem(db, topic, userId)
	var recent []models.PKRound
	_ = db.Where("topic_id = ? AND winner <> ''", topic.Id).Order("round_no desc").Limit(10).Find(&recent).Error
	item["recentRounds"] = recent
	item["stats"] = map[string]any{
		"totalRounds":       topic.TotalRounds,
		"winsA":             topic.WinsA,
		"winsB":             topic.WinsB,
		"currentStreakSide": topic.CurrentStreakSide,
		"currentStreak":     topic.CurrentStreak,
		"maxStreakA":        topic.MaxStreakA,
		"maxStreakB":        topic.MaxStreakB,
	}
	return item, nil
}

func (s *pkService) PlaceBet(userId int64, form PKBetForm) (map[string]any, error) {
	form.Side = normalizePKSide(form.Side)
	form.RequestId = strings.TrimSpace(form.RequestId)
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if form.TopicId <= 0 {
		return nil, errors.New("topicId is required")
	}
	if form.Side != PKSideA && form.Side != PKSideB {
		return nil, errors.New("invalid side")
	}
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}

	var bet *models.PKBet
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		topic := repositories.PKRepository.TakeTopic(tx, "id = ? AND status = ?", form.TopicId, PKTopicStatusEnabled)
		if topic == nil {
			return errors.New("pk topic not found")
		}
		round, err := repositories.PKRepository.TakeRoundForUpdate(tx, topic.CurrentRoundId)
		if err != nil {
			return errors.New("pk round not found")
		}
		now := dates.NowTimestamp()
		s.syncRoundPhase(round, now)
		if round.Phase != PKPhaseBetting {
			return errors.New("pk round is not betting")
		}
		if existing := repositories.PKRepository.TakeBet(tx, "round_id = ? AND user_id = ? AND request_id = ?", round.Id, userId, form.RequestId); existing != nil {
			bet = existing
			return nil
		}
		if repositories.PKRepository.TakeBet(tx, "round_id = ? AND user_id = ?", round.Id, userId) != nil {
			return errors.New("already bet in this round")
		}
		bet = &models.PKBet{
			TopicId:    topic.Id,
			RoundId:    round.Id,
			UserId:     userId,
			Side:       form.Side,
			Amount:     PKBetAmount,
			RequestId:  form.RequestId,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.PKRepository.CreateBet(tx, bet); err != nil {
			return err
		}
		if err := UserCoinService.SpendToPool(tx, userId, "PK_BET_STAKE_IN", bet.Id, bet.Amount, fmt.Sprintf("pk bet: topicId=%d roundId=%d side=%s", topic.Id, round.Id, form.Side)); err != nil {
			return err
		}
		heat := s.betHeat(bet.Amount)
		if err := repositories.PKRepository.CreateAction(tx, &models.PKAction{
			TopicId:    topic.Id,
			RoundId:    round.Id,
			UserId:     userId,
			Side:       form.Side,
			ActionType: "bet",
			EntityType: "pk_bet",
			EntityId:   bet.Id,
			Amount:     bet.Amount,
			Heat:       heat,
			AntiSpam:   1,
			RequestId:  form.RequestId,
			CreateTime: now,
		}); err != nil {
			return err
		}
		if form.Side == PKSideA {
			round.PoolA += bet.Amount
			round.BetCountA++
			round.HeatA += heat
		} else {
			round.PoolB += bet.Amount
			round.BetCountB++
			round.HeatB += heat
		}
		round.UpdateTime = now
		return repositories.PKRepository.UpdateRound(tx, round)
	})
	if err != nil {
		return nil, err
	}
	uc, _ := UserCoinService.GetOrCreate(userId)
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", bet.RoundId)
	return map[string]any{
		"bet":      bet,
		"round":    round,
		"userCoin": uc,
		"oddsA":    calcPKOdds(round.PoolA, round.PoolB, PKSideA),
		"oddsB":    calcPKOdds(round.PoolA, round.PoolB, PKSideB),
	}, nil
}

func (s *pkService) Heat(topicId int64) (map[string]any, error) {
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ?", topicId)
	if topic == nil {
		return nil, errors.New("pk topic not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", topic.CurrentRoundId)
	if round == nil {
		return nil, errors.New("pk round not found")
	}
	return map[string]any{
		"roundId":          round.Id,
		"phase":            s.phaseByTime(round, dates.NowTimestamp()),
		"heatA":            round.HeatA,
		"heatB":            round.HeatB,
		"leader":           leaderOf(round),
		"streakStatus":     streakStatus(topic.LastWinner, round),
		"countdownSeconds": countdownSeconds(round, dates.NowTimestamp()),
	}, nil
}

func (s *pkService) CreateComment(userId int64, form req.CreateCommentForm, topicId int64, side string) (*models.Comment, map[string]any, error) {
	side = normalizePKSide(side)
	if topicId <= 0 {
		return nil, nil, errors.New("topicId is required")
	}
	if side != PKSideA && side != PKSideB {
		return nil, nil, errors.New("invalid side")
	}
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ? AND status = ?", topicId, PKTopicStatusEnabled)
	if topic == nil {
		return nil, nil, errors.New("pk topic not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", topic.CurrentRoundId)
	if round == nil {
		return nil, nil, errors.New("pk round not found")
	}
	if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
		return nil, nil, errors.New("pk round is cooldown")
	}
	if bet := repositories.PKRepository.TakeBet(sqls.DB(), "round_id = ? AND user_id = ?", round.Id, userId); bet != nil && bet.Side != side {
		return nil, nil, errors.New("side must match your bet")
	}
	form.EntityType = constants.EntityPKTopic
	form.EntityId = topic.Id
	comment, err := CommentService.Publish(userId, form)
	if err != nil {
		return nil, nil, err
	}
	if err := s.attachCommentMeta(comment, topic, round, side, "comment"); err != nil {
		return nil, nil, err
	}
	heat, _ := s.RecalcRoundHeat(round.Id)
	return comment, heat, nil
}

func (s *pkService) ReplyComment(userId int64, form req.CreateCommentForm, commentId int64) (*models.Comment, map[string]any, error) {
	meta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", commentId)
	if meta == nil {
		return nil, nil, errors.New("pk comment not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", meta.RoundId)
	if round == nil {
		return nil, nil, errors.New("pk round not found")
	}
	if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
		return nil, nil, errors.New("pk round is cooldown")
	}
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ?", meta.TopicId)
	if topic == nil {
		return nil, nil, errors.New("pk topic not found")
	}
	form.EntityType = constants.EntityComment
	form.EntityId = commentId
	comment, err := CommentService.Publish(userId, form)
	if err != nil {
		return nil, nil, err
	}
	if err := s.attachCommentMeta(comment, topic, round, meta.Side, "reply"); err != nil {
		return nil, nil, err
	}
	heat, _ := s.RecalcRoundHeat(round.Id)
	return comment, heat, nil
}

func (s *pkService) Downvote(userId int64, form PKDownvoteForm) (map[string]any, error) {
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.CommentId <= 0 {
		return nil, errors.New("commentId is required")
	}
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}
	meta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", form.CommentId)
	if meta == nil {
		return nil, errors.New("pk comment not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", meta.RoundId)
	if round == nil {
		return nil, errors.New("pk round not found")
	}
	if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
		return nil, errors.New("pk round is cooldown")
	}
	bet := repositories.PKRepository.TakeBet(sqls.DB(), "round_id = ? AND user_id = ?", round.Id, userId)
	if bet == nil {
		return nil, errors.New("bet required before downvote")
	}
	if bet.Side == meta.Side {
		return nil, errors.New("cannot downvote your side")
	}
	now := dates.NowTimestamp()
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if repositories.PKRepository.TakeCommentMeta(tx, "comment_id = ?", form.CommentId) == nil {
			return errors.New("pk comment not found")
		}
		if existing := repositories.PKRepository.TakeBet(tx, "round_id = ? AND user_id = ?", round.Id, userId); existing == nil {
			return errors.New("bet required before downvote")
		}
		if action := repositories.PKRepository.TakeTopic(tx, "id = ?", meta.TopicId); action == nil {
			return errors.New("pk topic not found")
		}
		if err := repositories.PKRepository.CreateAction(tx, &models.PKAction{
			TopicId:    meta.TopicId,
			RoundId:    meta.RoundId,
			UserId:     userId,
			Side:       bet.Side,
			ActionType: "downvote",
			EntityType: constants.EntityComment,
			EntityId:   form.CommentId,
			Amount:     1,
			Heat:       2,
			AntiSpam:   1,
			RequestId:  form.RequestId,
			CreateTime: now,
		}); err != nil {
			return err
		}
		meta.DownvoteCount++
		meta.UpdateTime = now
		return repositories.PKRepository.UpdateCommentMeta(tx, meta)
	})
	if err != nil {
		return nil, err
	}
	heat, err := s.RecalcRoundHeat(round.Id)
	if err != nil {
		return nil, err
	}
	return heat, nil
}

func (s *pkService) Comments(topicId int64, side string, cursor int64, sort string, userId int64) ([]map[string]any, int64, bool, error) {
	side = normalizePKSide(side)
	if side != PKSideA && side != PKSideB {
		return nil, cursor, false, errors.New("invalid side")
	}
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ?", topicId)
	if topic == nil {
		return nil, cursor, false, errors.New("pk topic not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", topic.CurrentRoundId)
	if round == nil {
		return nil, cursor, false, errors.New("pk round not found")
	}
	limit := 20
	db := sqls.DB().Table("t_comment AS c").
		Select("c.*").
		Joins("JOIN t_pk_comment_meta AS m ON m.comment_id = c.id").
		Where("m.topic_id = ? AND m.round_id = ? AND m.side = ? AND c.status = ?", topic.Id, round.Id, side, constants.StatusOk)
	if cursor > 0 {
		db = db.Where("c.id < ?", cursor)
	}
	if sort == "heat" {
		db = db.Order("m.heat_score desc, c.id desc")
	} else {
		db = db.Order("c.id desc")
	}
	var comments []models.Comment
	if err := db.Limit(limit).Find(&comments).Error; err != nil {
		return nil, cursor, false, err
	}
	nextCursor := cursor
	if len(comments) > 0 {
		nextCursor = comments[len(comments)-1].Id
	}
	hasMore := len(comments) >= limit
	ids := make([]int64, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.Id)
	}
	metaMap := map[int64]models.PKCommentMeta{}
	if len(ids) > 0 {
		var metas []models.PKCommentMeta
		_ = sqls.DB().Where("comment_id IN ?", ids).Find(&metas).Error
		for _, m := range metas {
			metaMap[m.CommentId] = m
		}
	}
	liked := map[int64]bool{}
	if userId > 0 && len(ids) > 0 {
		for _, id := range UserLikeService.IsLiked(userId, constants.EntityComment, ids) {
			liked[id] = true
		}
	}
	ret := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		m := metaMap[c.Id]
		ret = append(ret, map[string]any{
			"comment":       c,
			"side":          m.Side,
			"heatScore":     m.HeatScore,
			"downvoteCount": m.DownvoteCount,
			"liked":         liked[c.Id],
		})
	}
	return ret, nextCursor, hasMore, nil
}

func (s *pkService) History(topicId int64, page, pageSize int) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var list []models.PKRound
	q := sqls.DB().Model(&models.PKRound{}).Where("topic_id = ? AND winner <> ''", topicId)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if err := q.Order("round_no desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": list, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) Seasons(topicId int64, page, pageSize int) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var list []models.PKSeason
	q := sqls.DB().Model(&models.PKSeason{}).Where("topic_id = ?", topicId)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if err := q.Order("season_no desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": list, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) MyBets(userId int64, page, pageSize int) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	var bets []models.PKBet
	q := sqls.DB().Model(&models.PKBet{}).Where("user_id = ?", userId)
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if err := q.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bets).Error; err != nil {
		return nil, err
	}
	ret := make([]map[string]any, 0, len(bets))
	for _, b := range bets {
		ret = append(ret, map[string]any{
			"bet":   b,
			"topic": repositories.PKRepository.TakeTopic(sqls.DB(), "id = ?", b.TopicId),
			"round": repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", b.RoundId),
		})
	}
	return map[string]any{"list": ret, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) AdminListTopics(page, pageSize int, status, q string) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := sqls.DB().Model(&models.PKTopic{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("title LIKE ? OR slug LIKE ? OR side_a_name LIKE ? OR side_b_name LIKE ?", like, like, like, like)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, err
	}
	var topics []models.PKTopic
	if err := db.Order("sort asc, id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&topics).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(topics))
	for i := range topics {
		list = append(list, s.buildTopicItem(sqls.DB(), &topics[i], 0))
	}
	return map[string]any{"list": list, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) SaveTopic(form PKTopicSaveForm) (*models.PKTopic, error) {
	form.Slug = strings.TrimSpace(form.Slug)
	form.Title = strings.TrimSpace(form.Title)
	form.SideAName = strings.TrimSpace(form.SideAName)
	form.SideBName = strings.TrimSpace(form.SideBName)
	form.Status = strings.TrimSpace(form.Status)
	if form.Title == "" {
		return nil, errors.New("title is required")
	}
	if form.SideAName == "" || form.SideBName == "" {
		return nil, errors.New("sides are required")
	}
	if form.Status == "" {
		form.Status = PKTopicStatusEnabled
	}
	if form.Status != PKTopicStatusEnabled && form.Status != PKTopicStatusDisabled {
		return nil, errors.New("invalid status")
	}
	if form.Slug == "" {
		form.Slug = fmt.Sprintf("pk-%d", dates.NowTimestamp())
	}
	now := dates.NowTimestamp()
	var topic *models.PKTopic
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if form.Id > 0 {
			topic = repositories.PKRepository.TakeTopic(tx, "id = ?", form.Id)
			if topic == nil {
				return errors.New("pk topic not found")
			}
			if exists := repositories.PKRepository.TakeTopic(tx, "slug = ? AND id <> ?", form.Slug, form.Id); exists != nil {
				return errors.New("slug already exists")
			}
			topic.Slug = form.Slug
			topic.Title = form.Title
			topic.SideAName = form.SideAName
			topic.SideBName = form.SideBName
			topic.Status = form.Status
			topic.Sort = form.Sort
			topic.Cover = form.Cover
			topic.UpdateTime = now
			return repositories.PKRepository.UpdateTopic(tx, topic)
		}
		if exists := repositories.PKRepository.TakeTopic(tx, "slug = ?", form.Slug); exists != nil {
			return errors.New("slug already exists")
		}
		topic = &models.PKTopic{
			Slug:       form.Slug,
			Title:      form.Title,
			SideAName:  form.SideAName,
			SideBName:  form.SideBName,
			Status:     form.Status,
			Sort:       form.Sort,
			Cover:      form.Cover,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.PKRepository.CreateTopic(tx, topic); err != nil {
			return err
		}
		return s.ensureTopicRuntime(tx, topic, now)
	})
	return topic, err
}

func (s *pkService) SetTopicStatus(topicId int64, status string) (*models.PKTopic, error) {
	if status != PKTopicStatusEnabled && status != PKTopicStatusDisabled {
		return nil, errors.New("invalid status")
	}
	var topic *models.PKTopic
	now := dates.NowTimestamp()
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		topic = repositories.PKRepository.TakeTopic(tx, "id = ?", topicId)
		if topic == nil {
			return errors.New("pk topic not found")
		}
		topic.Status = status
		topic.UpdateTime = now
		if status == PKTopicStatusEnabled {
			if err := s.ensureTopicRuntime(tx, topic, now); err != nil {
				return err
			}
		}
		return repositories.PKRepository.UpdateTopic(tx, topic)
	})
	return topic, err
}

func (s *pkService) AdminRounds(page, pageSize int, topicId int64, phase, winner string) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := sqls.DB().Model(&models.PKRound{})
	if topicId > 0 {
		db = db.Where("topic_id = ?", topicId)
	}
	if phase != "" {
		db = db.Where("phase = ?", phase)
	}
	if winner != "" {
		db = db.Where("winner = ?", winner)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, err
	}
	var rounds []models.PKRound
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rounds).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rounds, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) AdminSeasons(page, pageSize int, topicId int64, status string) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := sqls.DB().Model(&models.PKSeason{})
	if topicId > 0 {
		db = db.Where("topic_id = ?", topicId)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return nil, err
	}
	var seasons []models.PKSeason
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&seasons).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": seasons, "count": count, "page": page, "pageSize": pageSize}, nil
}

func (s *pkService) RecalcRoundHeat(roundId int64) (map[string]any, error) {
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", roundId)
	if round == nil {
		return nil, errors.New("pk round not found")
	}
	var actions []models.PKAction
	if err := sqls.DB().Where("round_id = ?", round.Id).Find(&actions).Error; err != nil {
		return nil, err
	}
	heatA, heatB := float64(0), float64(0)
	downvoteCount := int64(0)
	for _, a := range actions {
		if a.ActionType == "comment" || a.ActionType == "reply" {
			continue
		}
		if a.ActionType == "downvote" {
			downvoteCount++
		}
		if a.Side == PKSideA {
			heatA += a.Heat
		} else if a.Side == PKSideB {
			heatB += a.Heat
		}
	}

	var metas []models.PKCommentMeta
	if err := sqls.DB().Where("round_id = ?", round.Id).Find(&metas).Error; err != nil {
		return nil, err
	}
	commentCount := int64(len(metas))
	likeCount := int64(0)
	for i := range metas {
		c := repositories.CommentRepository.Get(sqls.DB(), metas[i].CommentId)
		if c == nil || c.Status != constants.StatusOk {
			continue
		}
		replyCount := c.CommentCount
		likeCount += c.LikeCount
		commentHeat := math.Min(3+math.Log(1+float64(c.LikeCount)+1.5*float64(replyCount)), 20)
		if metas[i].DownvoteCount > 0 {
			commentHeat = commentHeat * (1 / (1 + 0.2*float64(metas[i].DownvoteCount)))
		}
		metas[i].HeatScore = commentHeat
		metas[i].UpdateTime = dates.NowTimestamp()
		_ = repositories.PKRepository.UpdateCommentMeta(sqls.DB(), &metas[i])
		if metas[i].Side == PKSideA {
			heatA += commentHeat
		} else if metas[i].Side == PKSideB {
			heatB += commentHeat
		}
	}
	round.HeatA = heatA
	round.HeatB = heatB
	round.CommentCount = commentCount
	round.LikeCount = likeCount
	round.DownvoteCount = downvoteCount
	round.UpdateTime = dates.NowTimestamp()
	if err := repositories.PKRepository.UpdateRound(sqls.DB(), round); err != nil {
		return nil, err
	}
	return map[string]any{"round": round, "heatA": heatA, "heatB": heatB}, nil
}

func (s *pkService) CronTick() error {
	now := dates.NowTimestamp()
	var topics []models.PKTopic
	if err := sqls.DB().Where("status = ?", PKTopicStatusEnabled).Find(&topics).Error; err != nil {
		return err
	}
	for i := range topics {
		if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
			topic := repositories.PKRepository.TakeTopic(tx, "id = ?", topics[i].Id)
			if topic == nil {
				return nil
			}
			if err := s.ensureTopicRuntime(tx, topic, now); err != nil {
				return err
			}
			round, err := repositories.PKRepository.TakeRoundForUpdate(tx, topic.CurrentRoundId)
			if err != nil {
				return err
			}
			oldPhase := round.Phase
			s.syncRoundPhase(round, now)
			if oldPhase != round.Phase {
				round.UpdateTime = now
				if err := repositories.PKRepository.UpdateRound(tx, round); err != nil {
					return err
				}
			}
			if round.Phase == PKPhaseCooldown && round.SettledAt == 0 {
				if err := s.settleRound(tx, topic, round, now); err != nil {
					return err
				}
			}
			if round.Phase == PKPhaseCooldown && now >= round.NextRoundTime {
				return s.createNextRound(tx, topic, round, now)
			}
			return repositories.PKRepository.UpdateTopic(tx, topic)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *pkService) findTopic(topicId int64, slug string) (*models.PKTopic, error) {
	if topicId <= 0 && strings.TrimSpace(slug) == "" {
		return nil, errors.New("topicId or slug is required")
	}
	db := sqls.DB()
	var topic *models.PKTopic
	if topicId > 0 {
		topic = repositories.PKRepository.TakeTopic(db, "id = ?", topicId)
	} else {
		topic = repositories.PKRepository.TakeTopic(db, "slug = ?", strings.TrimSpace(slug))
	}
	if topic == nil {
		return nil, errors.New("pk topic not found")
	}
	return topic, nil
}

func (s *pkService) buildTopicItem(db *gorm.DB, topic *models.PKTopic, userId int64) map[string]any {
	round := repositories.PKRepository.TakeRound(db, "id = ?", topic.CurrentRoundId)
	season := repositories.PKRepository.TakeSeason(db, "id = ?", topic.CurrentSeasonId)
	var myBet *models.PKBet
	mySide := ""
	if userId > 0 && round != nil {
		myBet = repositories.PKRepository.TakeBet(db, "round_id = ? AND user_id = ?", round.Id, userId)
		if myBet != nil {
			mySide = myBet.Side
		}
	}
	oddsA, oddsB := 0.0, 0.0
	countdown := int64(0)
	leader := ""
	streak := ""
	if round != nil {
		oddsA = calcPKOdds(round.PoolA, round.PoolB, PKSideA)
		oddsB = calcPKOdds(round.PoolA, round.PoolB, PKSideB)
		countdown = countdownSeconds(round, dates.NowTimestamp())
		leader = leaderOf(round)
		streak = streakStatus(topic.LastWinner, round)
	}
	return map[string]any{
		"topic":            topic,
		"round":            round,
		"season":           season,
		"oddsA":            oddsA,
		"oddsB":            oddsB,
		"leader":           leader,
		"streakStatus":     streak,
		"countdownSeconds": countdown,
		"mySide":           mySide,
		"myBet":            myBet,
	}
}

func (s *pkService) ensureTopicRuntime(tx *gorm.DB, topic *models.PKTopic, now int64) error {
	if topic.CurrentSeasonId > 0 && topic.CurrentRoundId > 0 {
		return nil
	}
	seasonNo := 1
	season := repositories.PKRepository.TakeSeason(tx, "topic_id = ? AND status = ?", topic.Id, PKSeasonStatusActive)
	if season == nil {
		season = &models.PKSeason{
			TopicId:    topic.Id,
			SeasonNo:   seasonNo,
			StartTime:  now,
			EndTime:    now + pkSeasonSec,
			Status:     PKSeasonStatusActive,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.PKRepository.CreateSeason(tx, season); err != nil {
			return err
		}
	}
	round := repositories.PKRepository.TakeRound(tx, "topic_id = ? AND phase IN ?", topic.Id, []string{PKPhaseBetting, PKPhaseLocked, PKPhaseCooldown})
	if round == nil {
		round = newPKRound(topic.Id, season.Id, 1, now)
		if err := repositories.PKRepository.CreateRound(tx, round); err != nil {
			return err
		}
	}
	topic.CurrentSeasonId = season.Id
	topic.CurrentRoundId = round.Id
	topic.UpdateTime = now
	return nil
}

func (s *pkService) attachCommentMeta(comment *models.Comment, topic *models.PKTopic, round *models.PKRound, side, actionType string) error {
	now := dates.NowTimestamp()
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		if repositories.PKRepository.TakeCommentMeta(tx, "comment_id = ?", comment.Id) != nil {
			return nil
		}
		meta := &models.PKCommentMeta{
			CommentId:    comment.Id,
			TopicId:      topic.Id,
			RoundId:      round.Id,
			Side:         side,
			QualityScore: 1,
			HeatScore:    3,
			CreateTime:   now,
			UpdateTime:   now,
		}
		if err := repositories.PKRepository.CreateCommentMeta(tx, meta); err != nil {
			return err
		}
		return repositories.PKRepository.CreateAction(tx, &models.PKAction{
			TopicId:    topic.Id,
			RoundId:    round.Id,
			UserId:     comment.UserId,
			Side:       side,
			ActionType: actionType,
			EntityType: constants.EntityComment,
			EntityId:   comment.Id,
			Amount:     1,
			Heat:       3,
			AntiSpam:   1,
			CreateTime: now,
		})
	})
}

func (s *pkService) settleRound(tx *gorm.DB, topic *models.PKTopic, round *models.PKRound, now int64) error {
	if round.SettledAt > 0 {
		return nil
	}
	if _, err := s.RecalcRoundHeat(round.Id); err != nil {
		return err
	}
	round = repositories.PKRepository.TakeRound(tx, "id = ?", round.Id)
	winner := PKSideDraw
	if round.HeatA > round.HeatB {
		winner = PKSideA
	} else if round.HeatB > round.HeatA {
		winner = PKSideB
	}
	round.Winner = winner
	round.SettledAt = now
	round.Phase = PKPhaseCooldown
	round.UpdateTime = now
	if err := repositories.PKRepository.UpdateRound(tx, round); err != nil {
		return err
	}
	if err := s.generateSettlement(tx, round, winner, now); err != nil {
		return err
	}
	return s.updateTopicAndSeasonAfterRound(tx, topic, round, winner, now)
}

func (s *pkService) generateSettlement(tx *gorm.DB, round *models.PKRound, winner string, now int64) error {
	var existing int64
	if err := tx.Model(&models.PKSettlementItem{}).Where("round_id = ?", round.Id).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	var bets []models.PKBet
	if err := tx.Where("round_id = ?", round.Id).Find(&bets).Error; err != nil {
		return err
	}
	winPool, losePool := round.PoolA, round.PoolB
	if winner == PKSideB {
		winPool, losePool = round.PoolB, round.PoolA
	}
	for i := range bets {
		result := "lose"
		payout := int64(0)
		if winner == PKSideDraw {
			result = "draw"
			payout = bets[i].Amount
		} else if bets[i].Side == winner {
			result = "win"
			payout = bets[i].Amount
			if winPool > 0 {
				payout += int64(math.Floor(float64(losePool) * float64(bets[i].Amount) / float64(winPool)))
			}
		}
		item := &models.PKSettlementItem{
			TopicId:      round.TopicId,
			RoundId:      round.Id,
			BetId:        bets[i].Id,
			UserId:       bets[i].UserId,
			Side:         bets[i].Side,
			Result:       result,
			StakeAmount:  bets[i].Amount,
			PayoutAmount: payout,
			Paid:         payout == 0,
			CreateTime:   now,
			UpdateTime:   now,
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		bets[i].SettleResult = result
		bets[i].Payout = payout
		bets[i].SettledAt = now
		bets[i].UpdateTime = now
		if err := repositories.PKRepository.UpdateBet(tx, &bets[i]); err != nil {
			return err
		}
		if payout > 0 {
			bizType := "PK_PAYOUT"
			if result == "draw" {
				bizType = "PK_DRAW_REFUND"
			}
			if err := UserCoinService.PayFromPoolToUser(tx, bets[i].UserId, bizType, item.Id, payout, fmt.Sprintf("pk settle: roundId=%d result=%s", round.Id, result)); err != nil {
				return err
			}
			item.Paid = true
			item.UpdateTime = now
			if err := tx.Save(item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *pkService) updateTopicAndSeasonAfterRound(tx *gorm.DB, topic *models.PKTopic, round *models.PKRound, winner string, now int64) error {
	season := repositories.PKRepository.TakeSeason(tx, "id = ?", round.SeasonId)
	if season != nil {
		season.TotalRounds++
		if winner == PKSideA {
			season.WinsA++
		} else if winner == PKSideB {
			season.WinsB++
		}
		if now >= season.EndTime {
			season.Status = PKSeasonStatusFinished
			season.Champion = PKSideDraw
			if season.WinsA > season.WinsB {
				season.Champion = PKSideA
			} else if season.WinsB > season.WinsA {
				season.Champion = PKSideB
			}
		}
		season.UpdateTime = now
		if err := repositories.PKRepository.UpdateSeason(tx, season); err != nil {
			return err
		}
	}
	topic.TotalRounds++
	if winner == PKSideA {
		topic.WinsA++
	} else if winner == PKSideB {
		topic.WinsB++
	}
	if winner == PKSideA || winner == PKSideB {
		if topic.CurrentStreakSide == winner {
			topic.CurrentStreak++
		} else {
			topic.CurrentStreakSide = winner
			topic.CurrentStreak = 1
		}
		if winner == PKSideA && topic.CurrentStreak > topic.MaxStreakA {
			topic.MaxStreakA = topic.CurrentStreak
		}
		if winner == PKSideB && topic.CurrentStreak > topic.MaxStreakB {
			topic.MaxStreakB = topic.CurrentStreak
		}
		topic.LastWinner = winner
	}
	topic.UpdateTime = now
	return repositories.PKRepository.UpdateTopic(tx, topic)
}

func (s *pkService) createNextRound(tx *gorm.DB, topic *models.PKTopic, prev *models.PKRound, now int64) error {
	season := repositories.PKRepository.TakeSeason(tx, "id = ?", topic.CurrentSeasonId)
	if season == nil || season.Status == PKSeasonStatusFinished || now >= season.EndTime {
		seasonNo := 1
		var latest models.PKSeason
		if err := tx.Where("topic_id = ?", topic.Id).Order("season_no desc").Limit(1).Find(&latest).Error; err == nil && latest.Id > 0 {
			seasonNo = latest.SeasonNo + 1
		}
		season = &models.PKSeason{
			TopicId:    topic.Id,
			SeasonNo:   seasonNo,
			StartTime:  now,
			EndTime:    now + pkSeasonSec,
			Status:     PKSeasonStatusActive,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.PKRepository.CreateSeason(tx, season); err != nil {
			return err
		}
		topic.CurrentSeasonId = season.Id
	}
	nextNo := prev.RoundNo + 1
	if repositories.PKRepository.TakeRound(tx, "topic_id = ? AND round_no = ?", topic.Id, nextNo) != nil {
		return nil
	}
	round := newPKRound(topic.Id, season.Id, nextNo, now)
	if err := repositories.PKRepository.CreateRound(tx, round); err != nil {
		return err
	}
	topic.CurrentRoundId = round.Id
	topic.UpdateTime = now
	return repositories.PKRepository.UpdateTopic(tx, topic)
}

func newPKRound(topicId, seasonId int64, roundNo int, start int64) *models.PKRound {
	return &models.PKRound{
		TopicId:       topicId,
		SeasonId:      seasonId,
		RoundNo:       roundNo,
		Phase:         PKPhaseBetting,
		StartTime:     start,
		LockTime:      start + pkRoundBettingSec,
		EndTime:       start + pkRoundBettingSec + pkRoundLockedSec,
		NextRoundTime: start + pkRoundBettingSec + pkRoundLockedSec + pkRoundCooldownSec,
		CreateTime:    start,
		UpdateTime:    start,
	}
}

func (s *pkService) syncRoundPhase(round *models.PKRound, now int64) {
	round.Phase = s.phaseByTime(round, now)
}

func (s *pkService) phaseByTime(round *models.PKRound, now int64) string {
	if round.SettledAt > 0 || round.Winner != "" {
		if now < round.NextRoundTime {
			return PKPhaseCooldown
		}
		return PKPhaseSettled
	}
	if now >= round.EndTime {
		return PKPhaseCooldown
	}
	if now >= round.LockTime {
		return PKPhaseLocked
	}
	return PKPhaseBetting
}

func (s *pkService) betHeat(amount int64) float64 {
	return math.Sqrt(float64(amount)) * 0.5
}

func normalizePKSide(side string) string {
	return strings.ToUpper(strings.TrimSpace(side))
}

func calcPKOdds(poolA, poolB int64, side string) float64 {
	virtual := float64(PKBetAmount)
	effA := virtual + float64(poolA)
	effB := virtual + float64(poolB)
	if side == PKSideA {
		return math.Round(((effA+effB)/effA)*100) / 100
	}
	return math.Round(((effA+effB)/effB)*100) / 100
}

func leaderOf(round *models.PKRound) string {
	if round == nil {
		return ""
	}
	if round.HeatA > round.HeatB {
		return PKSideA
	}
	if round.HeatB > round.HeatA {
		return PKSideB
	}
	return PKSideDraw
}

func streakStatus(lastWinner string, round *models.PKRound) string {
	if lastWinner == "" || round == nil {
		return ""
	}
	leader := leaderOf(round)
	if leader == "" || leader == PKSideDraw {
		return ""
	}
	if leader == lastWinner {
		return "defending"
	}
	return "comeback"
}

func countdownSeconds(round *models.PKRound, now int64) int64 {
	if round == nil {
		return 0
	}
	var target int64
	if now < round.LockTime {
		target = round.LockTime
	} else if now < round.EndTime {
		target = round.EndTime
	} else if now < round.NextRoundTime {
		target = round.NextRoundTime
	}
	if target <= now {
		return 0
	}
	return target - now
}
