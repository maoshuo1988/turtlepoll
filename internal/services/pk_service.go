package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/models/req"
	"bbs-go/internal/repositories"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	PKBetAmountMin     int64 = 1
	PKBetAmountDefault int64 = 100
	pkRoundBettingSec        = 48 * 3600 * 1000
	pkRoundLockedSec         = 24 * 3600 * 1000
	pkRoundCooldownSec       = 10 * 60 * 1000
	pkSeasonSec              = 30 * 24 * 3600 * 1000

	TearEventTypePK      = "pk"
	TearLockTypeInteract = "INTERACT"
	TearLockTypeBet      = "BET"
	TearSnapshotSettle   = "SETTLE"
	TearSnapshotCkpt     = "CHECKPOINT"

	PKFlowLogMarker = "PK_FLOW"
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

	ListImage    string `json:"listImage"`
	SideABgImage string `json:"sideABgImage"`
	SideBBgImage string `json:"sideBBgImage"`
	SideABgColor string `json:"sideABgColor"`
	SideBBgColor string `json:"sideBBgColor"`
}

type PKBetForm struct {
	TopicId   int64  `json:"topicId"`
	Side      string `json:"side"`
	RequestId string `json:"requestId"`
	Amount    int64  `json:"amount"`
}

func (f *PKBetForm) UnmarshalJSON(data []byte) error {
	var raw struct {
		TopicId   json.RawMessage `json:"topicId"`
		Amount    json.RawMessage `json:"amount"`
		Side      string          `json:"side"`
		RequestId string          `json:"requestId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	f.Side = raw.Side
	f.RequestId = raw.RequestId
	f.Amount = PKBetAmountDefault
	if len(raw.TopicId) != 0 && string(raw.TopicId) != "null" {
		var topicId int64
		if err := json.Unmarshal(raw.TopicId, &topicId); err == nil {
			f.TopicId = topicId
		} else {
			var topicIdStr string
			if err := json.Unmarshal(raw.TopicId, &topicIdStr); err != nil {
				return err
			}
			topicIdStr = strings.TrimSpace(topicIdStr)
			if topicIdStr != "" {
				parsedTopicId, err := strconv.ParseInt(topicIdStr, 10, 64)
				if err != nil {
					return err
				}
				f.TopicId = parsedTopicId
			}
		}
	}

	amount, err := parsePKBetAmount(raw.Amount)
	if err != nil {
		return err
	}
	if amount > 0 {
		f.Amount = amount
	}
	return nil
}

func parsePKBetAmount(raw json.RawMessage) (int64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	var amount int64
	if err := json.Unmarshal(raw, &amount); err == nil {
		return amount, nil
	}

	var amountStr string
	if err := json.Unmarshal(raw, &amountStr); err != nil {
		return 0, err
	}
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" {
		return 0, nil
	}
	return strconv.ParseInt(amountStr, 10, 64)
}

type PKDownvoteForm struct {
	CommentId int64  `json:"commentId"`
	RequestId string `json:"requestId"`
}

type PKSettleForm struct {
	TopicId      int64  `json:"topicId"`
	RoundId      int64  `json:"roundId"`
	RequestId    string `json:"requestId"`
	SnapshotType string `json:"snapshotType"`
	FreezeSource string `json:"freezeSource"`
}

type PKRecordOptionForm struct {
	TopicId    int64  `json:"topicId"`
	RoundId    int64  `json:"roundId"`
	Option     string `json:"option"`
	ActionType string `json:"actionType"`
	RequestId  string `json:"requestId"`
	EntityType string `json:"entityType"`
	EntityId   int64  `json:"entityId"`
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
	if form.Amount <= 0 {
		form.Amount = PKBetAmountDefault
	}
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
	if form.Amount < PKBetAmountMin {
		return nil, errors.New("invalid amount")
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
		if err := s.ensureCampLockForBet(tx, topic.Id, round.Id, userId, form.Side, now); err != nil {
			return err
		}
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
			Amount:     form.Amount,
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
		if err := s.recordTearBetStat(tx, topic.Id, round.Id, userId, form.Side, bet.Amount, now); err != nil {
			return err
		}
		if err := s.recordTearInteraction(tx, topic.Id, round.Id, userId, form.Side, "bet", "pk_bet", bet.Id, heat, form.RequestId, now); err != nil {
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
	if _, _, err := s.computeHeatSnapshot(sqls.DB(), topic, round, TearSnapshotCkpt, "ON_DEMAND"); err != nil {
		return nil, err
	}
	options := s.buildHeatOptions(topic, round)
	heatA, heatB := float64(0), float64(0)
	for _, option := range options {
		side, _ := option["option"].(string)
		hTotal, _ := option["hTotal"].(float64)
		if side == PKSideA {
			heatA = hTotal
		}
		if side == PKSideB {
			heatB = hTotal
		}
	}
	return map[string]any{
		"roundId":          round.Id,
		"phase":            s.phaseByTime(round, dates.NowTimestamp()),
		"heatA":            heatA,
		"heatB":            heatB,
		"options":          options,
		"leader":           leaderOf(round),
		"streakStatus":     streakStatus(topic.LastWinner, round),
		"countdownSeconds": countdownSeconds(round, dates.NowTimestamp()),
	}, nil
}

func (s *pkService) CreateComment(userId int64, form req.CreateCommentForm, topicId int64, side string) (*models.Comment, map[string]any, string, error) {
	side = normalizePKSide(side)
	if topicId <= 0 {
		return nil, nil, "", errors.New("topicId is required")
	}
	if side != PKSideA && side != PKSideB {
		return nil, nil, "", errors.New("invalid side")
	}
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ? AND status = ?", topicId, PKTopicStatusEnabled)
	if topic == nil {
		return nil, nil, "", errors.New("pk topic not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", topic.CurrentRoundId)
	if round == nil {
		return nil, nil, "", errors.New("pk round not found")
	}
	if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
		return nil, nil, "", errors.New("pk round is cooldown")
	}
	if bet := repositories.PKRepository.TakeBet(sqls.DB(), "round_id = ? AND user_id = ?", round.Id, userId); bet != nil && bet.Side != side {
		return nil, nil, "", errors.New("side must match your bet")
	}
	now := dates.NowTimestamp()
	if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		return s.ensureCampLockForInteract(tx, topic.Id, round.Id, userId, side, now)
	}); err != nil {
		return nil, nil, "", err
	}
	form.EntityType = constants.EntityPKTopic
	form.EntityId = topic.Id
	comment, err := CommentService.Publish(userId, form)
	if err != nil {
		return nil, nil, "", err
	}
	if err := s.attachCommentMeta(comment, topic, round, side, "comment"); err != nil {
		return nil, nil, "", err
	}
	heat, _ := s.RecalcRoundHeat(round.Id)
	return comment, heat, side, nil
}

func (s *pkService) ReplyComment(userId int64, form req.CreateCommentForm, commentId int64) (*models.Comment, map[string]any, string, error) {
	meta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", commentId)
	if meta == nil {
		return nil, nil, "", errors.New("pk comment not found")
	}
	round := repositories.PKRepository.TakeRound(sqls.DB(), "id = ?", meta.RoundId)
	if round == nil {
		return nil, nil, "", errors.New("pk round not found")
	}
	if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
		return nil, nil, "", errors.New("pk round is cooldown")
	}
	topic := repositories.PKRepository.TakeTopic(sqls.DB(), "id = ?", meta.TopicId)
	if topic == nil {
		return nil, nil, "", errors.New("pk topic not found")
	}
	optionAtAction := s.getCampOption(sqls.DB(), topic.Id, round.Id, userId)
	if optionAtAction == "" {
		optionAtAction = meta.Side
	}
	now := dates.NowTimestamp()
	if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		return s.ensureCampLockForInteract(tx, topic.Id, round.Id, userId, optionAtAction, now)
	}); err != nil {
		return nil, nil, "", err
	}
	form.EntityType = constants.EntityComment
	form.EntityId = commentId
	comment, err := CommentService.Publish(userId, form)
	if err != nil {
		return nil, nil, "", err
	}
	if err := s.attachCommentMeta(comment, topic, round, optionAtAction, "reply"); err != nil {
		return nil, nil, "", err
	}
	heat, _ := s.RecalcRoundHeat(round.Id)
	return comment, heat, optionAtAction, nil
}

func (s *pkService) CommentOptionAtAction(commentId int64) string {
	if commentId <= 0 {
		return ""
	}
	meta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", commentId)
	if meta == nil {
		return ""
	}
	return normalizePKSide(meta.Side)
}

func (s *pkService) LikeComment(userId, commentId int64, requestId string) (map[string]any, error) {
	requestId = strings.TrimSpace(requestId)
	if commentId <= 0 {
		return nil, errors.New("commentId is required")
	}
	if requestId == "" {
		return nil, errors.New("requestId is required")
	}
	meta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", commentId)
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
	camp := s.getCampOption(sqls.DB(), meta.TopicId, meta.RoundId, userId)
	if camp != "" && camp != meta.Side {
		return nil, errors.New("TEAR_CAMP_CONFLICT")
	}
	now := dates.NowTimestamp()
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.ensureCampLockForInteract(tx, meta.TopicId, meta.RoundId, userId, meta.Side, now); err != nil {
			return err
		}
		if err := UserLikeService.CommentLike(userId, commentId); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already") {
				return nil
			}
			return err
		}
		return s.recordTearInteraction(tx, meta.TopicId, meta.RoundId, userId, meta.Side, "like", constants.EntityComment, commentId, 1, requestId, now)
	})
	if err != nil {
		return nil, err
	}
	heat, _ := s.RecalcRoundHeat(round.Id)
	return map[string]any{"liked": true, "optionAtAction": meta.Side, "heat": heat}, nil
}

func (s *pkService) SettleForUser(userId int64, form PKSettleForm) (map[string]any, error) {
	form.SnapshotType = strings.TrimSpace(form.SnapshotType)
	form.FreezeSource = strings.TrimSpace(form.FreezeSource)
	if form.SnapshotType == "" {
		form.SnapshotType = "SETTLE"
	}
	if form.FreezeSource == "" {
		form.FreezeSource = "ON_DEMAND"
	}
	if form.TopicId <= 0 && form.RoundId <= 0 {
		return nil, errors.New("topicId or roundId is required")
	}

	var round *models.PKRound
	var topic *models.PKTopic
	now := dates.NowTimestamp()
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		topic, round, err = s.resolveTopicRound(tx, form.TopicId, form.RoundId)
		if err != nil {
			return err
		}
		lockedRound, err := repositories.PKRepository.TakeRoundForUpdate(tx, round.Id)
		if err != nil {
			return errors.New("pk round not found")
		}
		round = lockedRound
		s.syncRoundPhase(round, now)
		if round.SettledAt == 0 {
			if now < round.EndTime {
				return errors.New("pk round is not ready to settle")
			}
			if _, _, err := s.computeHeatSnapshot(tx, topic, round, TearSnapshotSettle, form.FreezeSource); err != nil {
				return err
			}
			if err := s.settleRound(tx, topic, round, now); err != nil {
				return err
			}
			round = repositories.PKRepository.TakeRound(tx, "id = ?", round.Id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var item *models.PKSettlementItem
	settle := &models.PKSettlementItem{}
	if err := sqls.DB().Where("round_id = ? AND user_id = ?", round.Id, userId).Take(settle).Error; err == nil {
		item = settle
	}

	return map[string]any{
		"topicId":      topic.Id,
		"roundId":      round.Id,
		"snapshotType": form.SnapshotType,
		"freezeSource": form.FreezeSource,
		"winner":       round.Winner,
		"settledAt":    round.SettledAt,
		"options":      s.buildHeatOptions(topic, round),
		"settlement":   item,
	}, nil
}

func (s *pkService) HeatRank(topicId, roundId int64, scope string, viewerUserId int64, page, pageSize int) (map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	topic, round, err := s.resolveTopicRound(sqls.DB(), topicId, roundId)
	if err != nil {
		return nil, err
	}

	type heatRankRow struct {
		UserId          int64   `gorm:"column:user_id"`
		Option          string  `gorm:"column:bet_option"`
		TotalHeat       float64 `gorm:"column:total_heat"`
		ActionCount     int64   `gorm:"column:action_count"`
		FirstActionTime int64   `gorm:"column:first_action_time"`
	}
	var rows []heatRankRow
	db := sqls.DB().Table("t_tear_user_event_stat").
		Select("user_id, bet_option, heat_contribution AS total_heat, action_count, create_time AS first_action_time").
		Where("event_type = ? AND topic_id = ? AND round_id = ?", TearEventTypePK, topic.Id, round.Id)
	if strings.EqualFold(strings.TrimSpace(scope), "MY_SIDE") && viewerUserId > 0 {
		viewerOption := s.getCampOption(sqls.DB(), topic.Id, round.Id, viewerUserId)
		if viewerOption == "" {
			return map[string]any{"topicId": topic.Id, "roundId": round.Id, "phase": s.phaseByTime(round, dates.NowTimestamp()), "options": s.buildHeatOptions(topic, round), "list": []map[string]any{}, "count": int64(0), "page": page, "pageSize": pageSize, "leaderSide": leaderOf(round)}, nil
		}
		db = db.Where("bet_option = ?", viewerOption)
	}

	var count int64
	if err := sqls.DB().Table("(?) AS t", db).Count(&count).Error; err != nil {
		return nil, err
	}
	if err := db.Order("total_heat desc, first_action_time asc, user_id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	userIds := make([]int64, 0, len(rows))
	for _, row := range rows {
		userIds = append(userIds, row.UserId)
	}
	userMap := map[int64]models.User{}
	if len(userIds) > 0 {
		var users []models.User
		if err := sqls.DB().Where("id IN ?", userIds).Find(&users).Error; err == nil {
			for _, u := range users {
				userMap[u.Id] = u
			}
		}
	}

	list := make([]map[string]any, 0, len(rows))
	for idx, row := range rows {
		u := userMap[row.UserId]
		list = append(list, map[string]any{
			"rank":            (page-1)*pageSize + idx + 1,
			"userId":          row.UserId,
			"username":        u.Nickname,
			"nickname":        u.Nickname,
			"avatar":          u.Avatar,
			"option":          row.Option,
			"totalHeat":       row.TotalHeat,
			"heat":            row.TotalHeat,
			"actionCount":     row.ActionCount,
			"firstActionTime": row.FirstActionTime,
		})
	}

	return map[string]any{
		"topicId":    topic.Id,
		"roundId":    round.Id,
		"phase":      s.phaseByTime(round, dates.NowTimestamp()),
		"options":    s.buildHeatOptions(topic, round),
		"list":       list,
		"count":      count,
		"page":       page,
		"pageSize":   pageSize,
		"leaderSide": leaderOf(round),
	}, nil
}

func (s *pkService) HeatMe(userId, topicId, roundId int64) (map[string]any, error) {
	topic, round, err := s.resolveTopicRound(sqls.DB(), topicId, roundId)
	if err != nil {
		return nil, err
	}

	bet := repositories.PKRepository.TakeBet(sqls.DB(), "round_id = ? AND user_id = ?", round.Id, userId)
	stat := &models.TearUserEventStat{}
	_ = sqls.DB().Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", TearEventTypePK, topic.Id, round.Id, userId).Take(stat).Error
	myHeat := stat.HeatContribution
	myActionCount := stat.ActionCount
	mySideHeat := map[string]any{}
	if stat.BetOption != "" {
		mySideHeat[stat.BetOption] = map[string]any{"heat": stat.HeatContribution, "count": stat.ActionCount}
	}

	myRank := int64(0)
	if myHeat > 0 {
		type rankVal struct {
			Rank int64 `gorm:"column:rank"`
		}
		var r rankVal
		_ = sqls.DB().Raw(`
			SELECT COUNT(1) + 1 AS rank
			FROM (
				SELECT user_id, SUM(heat) AS total_heat
				FROM t_pk_action
				WHERE round_id = ?
				GROUP BY user_id
			) t
			WHERE t.total_heat > ?
		`, round.Id, myHeat).Scan(&r).Error
		if r.Rank > 0 {
			myRank = r.Rank
		} else {
			myRank = 1
		}
	}

	return map[string]any{
		"topicId":           topic.Id,
		"roundId":           round.Id,
		"phase":             s.phaseByTime(round, dates.NowTimestamp()),
		"options":           s.buildHeatOptions(topic, round),
		"myOption":          stat.BetOption,
		"receivedLikeCount": stat.LikeCount,
		"myCommentCount":    stat.CommentCount + stat.ReplyCount,
		"myBetAmount":       stat.BetAmount,
		"estimatedPayout":   betPayoutPreview(bet, round),
		"myHeat":            myHeat,
		"myActionCount":     myActionCount,
		"myRank":            myRank,
		"mySideHeat":        mySideHeat,
		"myBet":             bet,
	}, nil
}

func (s *pkService) OddsCurrent(topicId, roundId int64) (map[string]any, error) {
	topic, round, err := s.resolveTopicRound(sqls.DB(), topicId, roundId)
	if err != nil {
		return nil, err
	}
	options := s.buildHeatOptions(topic, round)
	return map[string]any{
		"topicId": topic.Id,
		"roundId": round.Id,
		"phase":   s.phaseByTime(round, dates.NowTimestamp()),
		"options": options,
		"oddsA":   calcPKOdds(round.PoolA, round.PoolB, PKSideA),
		"oddsB":   calcPKOdds(round.PoolA, round.PoolB, PKSideB),
	}, nil
}

func (s *pkService) RecordOption(userId int64, form PKRecordOptionForm) (map[string]any, error) {
	form.Option = normalizePKSide(form.Option)
	form.ActionType = strings.TrimSpace(form.ActionType)
	form.RequestId = strings.TrimSpace(form.RequestId)
	form.EntityType = strings.TrimSpace(form.EntityType)
	if form.Option != PKSideA && form.Option != PKSideB {
		return nil, errors.New("invalid option")
	}
	if form.ActionType == "" {
		return nil, errors.New("actionType is required")
	}
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}
	if form.EntityType == "" {
		form.EntityType = "pk_option"
	}

	var action *models.PKAction
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		topic, round, err := s.resolveTopicRound(tx, form.TopicId, form.RoundId)
		if err != nil {
			return err
		}
		if s.phaseByTime(round, dates.NowTimestamp()) == PKPhaseCooldown {
			return errors.New("pk round is cooldown")
		}
		if form.EntityId <= 0 {
			form.EntityId = round.Id
		}
		action = &models.PKAction{
			TopicId:    topic.Id,
			RoundId:    round.Id,
			UserId:     userId,
			Side:       form.Option,
			ActionType: form.ActionType,
			EntityType: form.EntityType,
			EntityId:   form.EntityId,
			Amount:     1,
			Heat:       1,
			AntiSpam:   1,
			RequestId:  form.RequestId,
			CreateTime: dates.NowTimestamp(),
		}
		if err := repositories.PKRepository.CreateAction(tx, action); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return s.recordTearInteraction(tx, topic.Id, round.Id, userId, form.Option, form.ActionType, form.EntityType, form.EntityId, 1, form.RequestId, dates.NowTimestamp())
			}
			return err
		}
		return s.recordTearInteraction(tx, topic.Id, round.Id, userId, form.Option, form.ActionType, form.EntityType, form.EntityId, 1, form.RequestId, dates.NowTimestamp())
	})
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"recorded":       true,
		"optionAtAction": form.Option,
		"actionType":     form.ActionType,
		"requestId":      form.RequestId,
		"action":         action,
	}, nil
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
			"option":        m.Side,
			"side":          m.Side,
			"heatScore":     m.HeatScore,
			"downvoteCount": m.DownvoteCount,
			"liked":         liked[c.Id],
		})
	}
	return ret, nextCursor, hasMore, nil
}

func (s *pkService) CommentReplies(commentId, cursor int64, pageSize int, userId int64) ([]map[string]any, int64, bool, error) {
	if commentId <= 0 {
		return nil, cursor, false, errors.New("commentId is required")
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	parentMeta := repositories.PKRepository.TakeCommentMeta(sqls.DB(), "comment_id = ?", commentId)
	if parentMeta == nil {
		return nil, cursor, false, errors.New("pk comment not found")
	}

	db := sqls.DB().Table("t_comment AS c").
		Select("c.*").
		Joins("JOIN t_pk_comment_meta AS m ON m.comment_id = c.id").
		Where("c.entity_type = ? AND c.entity_id = ? AND c.status = ?", constants.EntityComment, commentId, constants.StatusOk).
		Where("m.topic_id = ? AND m.round_id = ?", parentMeta.TopicId, parentMeta.RoundId)
	if cursor > 0 {
		db = db.Where("c.id > ?", cursor)
	}
	db = db.Order("c.id asc")

	var comments []models.Comment
	if err := db.Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, cursor, false, err
	}
	nextCursor := cursor
	if len(comments) > 0 {
		nextCursor = comments[len(comments)-1].Id
	}
	hasMore := len(comments) >= pageSize

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
			"option":        m.Side,
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
	form.Cover = strings.TrimSpace(form.Cover)
	form.ListImage = strings.TrimSpace(form.ListImage)
	form.SideABgImage = strings.TrimSpace(form.SideABgImage)
	form.SideBBgImage = strings.TrimSpace(form.SideBBgImage)
	form.SideABgColor = strings.TrimSpace(form.SideABgColor)
	form.SideBBgColor = strings.TrimSpace(form.SideBBgColor)
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
			topic.ListImage = form.ListImage
			topic.SideABgImage = form.SideABgImage
			topic.SideBBgImage = form.SideBBgImage
			topic.SideABgColor = form.SideABgColor
			topic.SideBBgColor = form.SideBBgColor
			topic.UpdateTime = now
			return repositories.PKRepository.UpdateTopic(tx, topic)
		}
		if exists := repositories.PKRepository.TakeTopic(tx, "slug = ?", form.Slug); exists != nil {
			return errors.New("slug already exists")
		}
		topic = &models.PKTopic{
			Slug:         form.Slug,
			Title:        form.Title,
			SideAName:    form.SideAName,
			SideBName:    form.SideBName,
			Status:       form.Status,
			Sort:         form.Sort,
			Cover:        form.Cover,
			ListImage:    form.ListImage,
			SideABgImage: form.SideABgImage,
			SideBBgImage: form.SideBBgImage,
			SideABgColor: form.SideABgColor,
			SideBBgColor: form.SideBBgColor,
			CreateTime:   now,
			UpdateTime:   now,
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
	return s.recalcRoundHeat(sqls.DB(), roundId)
}

func (s *pkService) recalcRoundHeat(db *gorm.DB, roundId int64) (map[string]any, error) {
	round := repositories.PKRepository.TakeRound(db, "id = ?", roundId)
	if round == nil {
		return nil, errors.New("pk round not found")
	}
	var actions []models.PKAction
	if err := db.Where("round_id = ?", round.Id).Find(&actions).Error; err != nil {
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
	if err := db.Where("round_id = ?", round.Id).Find(&metas).Error; err != nil {
		return nil, err
	}
	commentCount := int64(len(metas))
	likeCount := int64(0)
	for i := range metas {
		c := repositories.CommentRepository.Get(db, metas[i].CommentId)
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
		_ = repositories.PKRepository.UpdateCommentMeta(db, &metas[i])
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
	if err := repositories.PKRepository.UpdateRound(db, round); err != nil {
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
	slog.Info("pk cron scan topics",
		slog.String("marker", PKFlowLogMarker),
		slog.Int("topicCount", len(topics)),
		slog.Int64("now", now),
	)
	for i := range topics {
		if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
			topic := repositories.PKRepository.TakeTopic(tx, "id = ?", topics[i].Id)
			if topic == nil {
				slog.Warn("pk cron topic missing",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topics[i].Id),
				)
				return nil
			}
			if err := s.ensureTopicRuntime(tx, topic, now); err != nil {
				slog.Error("pk cron ensure runtime failed",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Any("err", err),
				)
				return err
			}
			round, err := repositories.PKRepository.TakeRoundForUpdate(tx, topic.CurrentRoundId)
			if err != nil {
				slog.Error("pk cron load current round failed",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("currentRoundId", topic.CurrentRoundId),
					slog.Any("err", err),
				)
				return err
			}
			oldPhase := round.Phase
			oldRoundID := round.Id
			s.syncRoundPhase(round, now)
			if oldPhase != round.Phase {
				slog.Info("pk round phase changed",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("roundId", round.Id),
					slog.String("from", oldPhase),
					slog.String("to", round.Phase),
					slog.Int64("now", now),
					slog.Int64("lockTime", round.LockTime),
					slog.Int64("endTime", round.EndTime),
					slog.Int64("nextRoundTime", round.NextRoundTime),
				)
				round.UpdateTime = now
				if err := repositories.PKRepository.UpdateRound(tx, round); err != nil {
					slog.Error("pk round phase update failed",
						slog.String("marker", PKFlowLogMarker),
						slog.Int64("topicId", topic.Id),
						slog.Int64("roundId", round.Id),
						slog.Any("err", err),
					)
					return err
				}
			}
			if round.Phase == PKPhaseCooldown && round.SettledAt == 0 {
				slog.Info("pk round settle trigger",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("roundId", round.Id),
					slog.Int64("now", now),
				)
				if err := s.settleRound(tx, topic, round, now); err != nil {
					slog.Error("pk round settle failed",
						slog.String("marker", PKFlowLogMarker),
						slog.Int64("topicId", topic.Id),
						slog.Int64("roundId", round.Id),
						slog.Any("err", err),
					)
					return err
				}
				slog.Info("pk round settle done",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("roundId", round.Id),
				)
			}
			if round.Phase == PKPhaseSettled && now >= round.NextRoundTime {
				slog.Info("pk next round trigger",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("prevRoundId", round.Id),
					slog.Int("prevRoundNo", round.RoundNo),
					slog.Int64("now", now),
					slog.Int64("nextRoundTime", round.NextRoundTime),
				)
				if err := s.createNextRound(tx, topic, round, now); err != nil {
					slog.Error("pk next round create failed",
						slog.String("marker", PKFlowLogMarker),
						slog.Int64("topicId", topic.Id),
						slog.Int64("prevRoundId", round.Id),
						slog.Any("err", err),
					)
					return err
				}
				slog.Info("pk next round create done",
					slog.String("marker", PKFlowLogMarker),
					slog.Int64("topicId", topic.Id),
					slog.Int64("prevRoundId", oldRoundID),
					slog.Int64("currentRoundId", topic.CurrentRoundId),
				)
				return nil
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
	hasBet := false
	if userId > 0 && round != nil {
		myBet = repositories.PKRepository.TakeBet(db, "round_id = ? AND user_id = ?", round.Id, userId)
		if myBet != nil {
			mySide = myBet.Side
			hasBet = true
		}
	}
	oddsA, oddsB := 0.0, 0.0
	countdown := int64(0)
	leader := ""
	streak := ""
	canSettle := false
	settleDisabledReason := ""
	now := dates.NowTimestamp()
	if round != nil {
		oddsA = calcPKOdds(round.PoolA, round.PoolB, PKSideA)
		oddsB = calcPKOdds(round.PoolA, round.PoolB, PKSideB)
		countdown = countdownSeconds(round, now)
		leader = leaderOf(round)
		streak = streakStatus(topic.LastWinner, round)

		if userId <= 0 {
			settleDisabledReason = "NOT_LOGIN"
		} else if round.SettledAt > 0 {
			settleDisabledReason = "ROUND_ALREADY_SETTLED"
		} else if now < round.EndTime {
			settleDisabledReason = "ROUND_NOT_ENDED"
		} else {
			canSettle = true
		}
	} else if userId <= 0 {
		settleDisabledReason = "NOT_LOGIN"
	} else {
		settleDisabledReason = "ROUND_NOT_READY"
	}

	if !canSettle && settleDisabledReason == "" {
		settleDisabledReason = "ROUND_NOT_READY"
	}
	return map[string]any{
		"topic":                topic,
		"round":                round,
		"season":               season,
		"oddsA":                oddsA,
		"oddsB":                oddsB,
		"leader":               leader,
		"streakStatus":         streak,
		"countdownSeconds":     countdown,
		"mySide":               mySide,
		"myBet":                myBet,
		"hasBet":               hasBet,
		"canSettle":            canSettle,
		"settleDisabledReason": settleDisabledReason,
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
		if err := repositories.PKRepository.CreateAction(tx, &models.PKAction{
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
		}); err != nil {
			return err
		}
		return s.recordTearInteraction(tx, topic.Id, round.Id, comment.UserId, side, actionType, constants.EntityComment, comment.Id, 3, "", now)
	})
}

func (s *pkService) settleRound(tx *gorm.DB, topic *models.PKTopic, round *models.PKRound, now int64) error {
	if round.SettledAt > 0 {
		return nil
	}
	options, snapshotTime, err := s.computeHeatSnapshot(tx, topic, round, TearSnapshotSettle, "ON_DEMAND")
	if err != nil {
		return err
	}
	heatA := float64(0)
	heatB := float64(0)
	for _, option := range options {
		side, _ := option["option"].(string)
		hTotal, _ := option["hTotal"].(float64)
		if side == PKSideA {
			heatA = hTotal
		}
		if side == PKSideB {
			heatB = hTotal
		}
	}
	winner := PKSideDraw
	if heatA > heatB {
		winner = PKSideA
	} else if heatB > heatA {
		winner = PKSideB
	}
	round.HeatA = heatA
	round.HeatB = heatB
	round.CommentCount = int64(len(options))
	round.Winner = winner
	round.SettledAt = now
	round.Phase = PKPhaseCooldown
	round.UpdateTime = snapshotTime
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
		slog.Info("pk next round needs new season",
			slog.String("marker", PKFlowLogMarker),
			slog.Int64("topicId", topic.Id),
			slog.Int64("currentSeasonId", topic.CurrentSeasonId),
			slog.Bool("seasonNil", season == nil),
		)
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
		slog.Warn("pk next round skipped due to duplicate roundNo",
			slog.String("marker", PKFlowLogMarker),
			slog.Int64("topicId", topic.Id),
			slog.Int("nextRoundNo", nextNo),
		)
		return nil
	}
	round := newPKRound(topic.Id, season.Id, nextNo, now)
	if err := repositories.PKRepository.CreateRound(tx, round); err != nil {
		return err
	}
	slog.Info("pk next round row created",
		slog.String("marker", PKFlowLogMarker),
		slog.Int64("topicId", topic.Id),
		slog.Int64("seasonId", season.Id),
		slog.Int64("roundId", round.Id),
		slog.Int("roundNo", round.RoundNo),
		slog.Int64("startTime", round.StartTime),
		slog.Int64("nextRoundTime", round.NextRoundTime),
	)
	topic.CurrentRoundId = round.Id
	topic.UpdateTime = now
	if err := repositories.PKRepository.UpdateTopic(tx, topic); err != nil {
		return err
	}
	slog.Info("pk topic currentRound switched",
		slog.String("marker", PKFlowLogMarker),
		slog.Int64("topicId", topic.Id),
		slog.Int64("currentRoundId", topic.CurrentRoundId),
		slog.Int64("currentSeasonId", topic.CurrentSeasonId),
	)
	return nil
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
	virtual := float64(PKBetAmountDefault)
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

func betPayoutPreview(bet *models.PKBet, round *models.PKRound) int64 {
	if bet == nil || round == nil {
		return 0
	}
	if bet.Side == PKSideA {
		if round.PoolA <= 0 {
			return bet.Amount
		}
		return bet.Amount + int64(math.Floor(float64(round.PoolB)*float64(bet.Amount)/float64(round.PoolA)))
	}
	if bet.Side == PKSideB {
		if round.PoolB <= 0 {
			return bet.Amount
		}
		return bet.Amount + int64(math.Floor(float64(round.PoolA)*float64(bet.Amount)/float64(round.PoolB)))
	}
	return 0
}

func (s *pkService) resolveTopicRound(db *gorm.DB, topicId, roundId int64) (*models.PKTopic, *models.PKRound, error) {
	if roundId > 0 {
		round := repositories.PKRepository.TakeRound(db, "id = ?", roundId)
		if round == nil {
			return nil, nil, errors.New("pk round not found")
		}
		topic := repositories.PKRepository.TakeTopic(db, "id = ?", round.TopicId)
		if topic == nil {
			return nil, nil, errors.New("pk topic not found")
		}
		if topicId > 0 && topic.Id != topicId {
			return nil, nil, errors.New("topicId does not match roundId")
		}
		return topic, round, nil
	}
	if topicId <= 0 {
		return nil, nil, errors.New("topicId or roundId is required")
	}
	topic := repositories.PKRepository.TakeTopic(db, "id = ?", topicId)
	if topic == nil {
		return nil, nil, errors.New("pk topic not found")
	}
	round := repositories.PKRepository.TakeRound(db, "id = ?", topic.CurrentRoundId)
	if round == nil {
		return nil, nil, errors.New("pk round not found")
	}
	return topic, round, nil
}

func (s *pkService) buildHeatOptions(topic *models.PKTopic, round *models.PKRound) []map[string]any {
	if topic == nil || round == nil {
		return []map[string]any{}
	}
	if snapshots, err := s.loadHeatSnapshot(sqls.DB(), topic.Id, round.Id, TearSnapshotSettle); err == nil && len(snapshots) > 0 {
		return snapshots
	}
	if snapshots, err := s.loadHeatSnapshot(sqls.DB(), topic.Id, round.Id, TearSnapshotCkpt); err == nil && len(snapshots) > 0 {
		return snapshots
	}
	return []map[string]any{
		{
			"option":    PKSideA,
			"name":      topic.SideAName,
			"heat":      round.HeatA,
			"pool":      round.PoolA,
			"betCount":  round.BetCountA,
			"odds":      calcPKOdds(round.PoolA, round.PoolB, PKSideA),
			"isLeading": round.HeatA > round.HeatB,
		},
		{
			"option":    PKSideB,
			"name":      topic.SideBName,
			"heat":      round.HeatB,
			"pool":      round.PoolB,
			"betCount":  round.BetCountB,
			"odds":      calcPKOdds(round.PoolA, round.PoolB, PKSideB),
			"isLeading": round.HeatB > round.HeatA,
		},
	}
}

func (s *pkService) loadHeatSnapshot(db *gorm.DB, topicId, roundId int64, snapshotType string) ([]map[string]any, error) {
	return TearHeatService.GetHeatSnapshot(db, TearHeatSnapshotQuery{
		EventType:    TearEventTypePK,
		EventId:      topicId,
		RoundId:      roundId,
		SnapshotType: snapshotType,
	})
}

func (s *pkService) computeHeatSnapshot(tx *gorm.DB, topic *models.PKTopic, round *models.PKRound, snapshotType, freezeSource string) ([]map[string]any, int64, error) {
	if topic == nil || round == nil {
		return nil, 0, errors.New("pk topic/round is required")
	}
	return TearHeatService.ComputeHeatSnapshot(tx, TearHeatSnapshotComputeRequest{
		EventType:    TearEventTypePK,
		EventId:      topic.Id,
		TopicId:      topic.Id,
		RoundId:      round.Id,
		SnapshotType: snapshotType,
		FreezeSource: freezeSource,
	})
}

func (s *pkService) getCampOption(db *gorm.DB, topicId, roundId, userId int64) string {
	if userId <= 0 || roundId <= 0 {
		return ""
	}
	member := &models.TearCampMember{}
	if err := db.Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", TearEventTypePK, topicId, roundId, userId).Take(member).Error; err != nil {
		return ""
	}
	return normalizePKSide(member.Option)
}

func (s *pkService) ensureCampLockForBet(tx *gorm.DB, topicId, roundId, userId int64, option string, now int64) error {
	option = normalizePKSide(option)
	member := &models.TearCampMember{}
	err := tx.Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", TearEventTypePK, topicId, roundId, userId).Take(member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.TearCampMember{
				EventType:       TearEventTypePK,
				EventId:         topicId,
				TopicId:         topicId,
				RoundId:         roundId,
				UserId:          userId,
				Option:          option,
				LockType:        TearLockTypeBet,
				FirstActionTime: now,
				CreateTime:      now,
				UpdateTime:      now,
			}).Error
		}
		return err
	}
	if normalizePKSide(member.Option) != option {
		if strings.EqualFold(member.LockType, TearLockTypeBet) {
			return errors.New("TEAR_CAMP_LOCKED_BY_BET")
		}
		return errors.New("TEAR_CAMP_CONFLICT")
	}
	member.LockType = TearLockTypeBet
	member.UpdateTime = now
	if member.FirstActionTime == 0 {
		member.FirstActionTime = now
	}
	return tx.Save(member).Error
}

func (s *pkService) ensureCampLockForInteract(tx *gorm.DB, topicId, roundId, userId int64, option string, now int64) error {
	option = normalizePKSide(option)
	member := &models.TearCampMember{}
	err := tx.Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", TearEventTypePK, topicId, roundId, userId).Take(member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&models.TearCampMember{
				EventType:       TearEventTypePK,
				EventId:         topicId,
				TopicId:         topicId,
				RoundId:         roundId,
				UserId:          userId,
				Option:          option,
				LockType:        TearLockTypeInteract,
				FirstActionTime: now,
				CreateTime:      now,
				UpdateTime:      now,
			}).Error
		}
		return err
	}
	if normalizePKSide(member.Option) != option {
		if strings.EqualFold(member.LockType, TearLockTypeBet) {
			return errors.New("TEAR_CAMP_LOCKED_BY_BET")
		}
		return errors.New("TEAR_CAMP_CONFLICT")
	}
	if member.FirstActionTime == 0 {
		member.FirstActionTime = now
	}
	member.UpdateTime = now
	return tx.Save(member).Error
}

func (s *pkService) recordTearBetStat(tx *gorm.DB, topicId, roundId, userId int64, betOption string, betAmount int64, now int64) error {
	stat := &models.TearUserEventStat{
		EventType:  "pk",
		EventId:    topicId,
		TopicId:    topicId,
		RoundId:    roundId,
		UserId:     userId,
		BetOption:  betOption,
		BetAmount:  betAmount,
		CreateTime: now,
		UpdateTime: now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "event_type"}, {Name: "round_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"event_id":    topicId,
			"topic_id":    topicId,
			"bet_option":  betOption,
			"bet_amount":  betAmount,
			"update_time": now,
		}),
	}).Create(stat).Error
}

func (s *pkService) recordTearInteraction(tx *gorm.DB, topicId, roundId, userId int64, option, actionType, entityType string, entityId int64, heat float64, requestId string, now int64) error {
	log := &models.TearInteractLog{
		EventType:        "pk",
		EventId:          topicId,
		TopicId:          topicId,
		RoundId:          roundId,
		UserId:           userId,
		OptionAtAction:   option,
		ActionType:       actionType,
		EntityType:       entityType,
		EntityId:         entityId,
		HeatContribution: heat,
		RequestId:        requestId,
		CreateTime:       now,
	}
	ret := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(log)
	if ret.Error != nil {
		return ret.Error
	}
	if ret.RowsAffected == 0 {
		return nil
	}

	commentInc := int64(0)
	replyInc := int64(0)
	likeInc := int64(0)
	if actionType == "comment" {
		commentInc = 1
	}
	if actionType == "reply" {
		replyInc = 1
	}
	if actionType == "like" {
		likeInc = 1
	}

	stat := &models.TearUserEventStat{
		EventType:        "pk",
		EventId:          topicId,
		TopicId:          topicId,
		RoundId:          roundId,
		UserId:           userId,
		ActionCount:      1,
		CommentCount:     commentInc,
		ReplyCount:       replyInc,
		LikeCount:        likeInc,
		HeatContribution: heat,
		CreateTime:       now,
		UpdateTime:       now,
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "event_type"}, {Name: "round_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"event_id":          topicId,
			"topic_id":          topicId,
			"action_count":      gorm.Expr("t_tear_user_event_stat.action_count + ?", 1),
			"comment_count":     gorm.Expr("t_tear_user_event_stat.comment_count + ?", commentInc),
			"reply_count":       gorm.Expr("t_tear_user_event_stat.reply_count + ?", replyInc),
			"like_count":        gorm.Expr("t_tear_user_event_stat.like_count + ?", likeInc),
			"heat_contribution": gorm.Expr("t_tear_user_event_stat.heat_contribution + ?", heat),
			"update_time":       now,
		}),
	}).Create(stat).Error
}
