package api

import (
	"bbs-go/internal/controllers/render"
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/services"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
)

type PredictController struct {
	Ctx iris.Context
}

func (c *PredictController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/markets", "GetMarkets")
	b.Handle("GET", "/my/markets", "GetMyMarkets")
	b.Handle("GET", "/markets/by-name", "GetMarketsByName")
	b.Handle("GET", "/markets/by-tag", "GetMarketsByTag")
	b.Handle("GET", "/bet-settle-result", "GetBetSettleResult")
	b.Handle("GET", "/heat", "GetHeat")
	b.Handle("GET", "/comments", "GetComments")
	b.Handle("GET", "/comment/replies", "GetCommentReplies")
	b.Handle("GET", "/heat/rank", "GetHeatRank")
	b.Handle("GET", "/heat/me", "GetHeatMe")
	b.Handle("GET", "/odds/current", "GetOddsCurrent")
	b.Handle("POST", "/context/update", "PostContextUpdate")
	b.Handle("GET", "/context/hot", "GetContextHot")
	b.Handle("GET", "/tags/hot", "GetTagsHot")
	b.Handle("POST", "/coin/bet", "PostCoinBet")
	b.Handle("POST", "/coin/settle", "PostCoinSettle")
	b.Handle("POST", "/tear/settle", "PostTearSettle")
	b.Handle("GET", "/coin/leaderboard", "GetCoinLeaderboard")
	b.Handle("POST", "/comment/create", "PostCommentCreate")
	b.Handle("POST", "/comment/reply", "PostCommentReply")
	b.Handle("POST", "/like", "PostLike")
	b.Handle("POST", "/unlike", "PostUnlike")
}

func (c *PredictController) GetMarkets() *web.JsonResult {
	ret := (&FootballController{Ctx: c.Ctx}).GetMarkets()
	c.attachTearSettlementForMarketList(ret)
	return ret
}

func (c *PredictController) GetMarketsByName() *web.JsonResult {
	ret := (&FootballController{Ctx: c.Ctx}).GetMarketsBy_name()
	c.attachTearSettlementForMarketList(ret)
	return ret
}

func (c *PredictController) GetMarketsByTag() *web.JsonResult {
	ret := (&FootballController{Ctx: c.Ctx}).GetMarketsBy_tag()
	c.attachTearSettlementForMarketList(ret)
	return ret
}

// GetMyMarkets 查询当前用户参与下注的市场列表。
// 返回结构与 /api/predict/markets 对齐：market/context/schedule/matchPhase/hasBet/betSettleResult/tearSettlement。
func (c *PredictController) GetMyMarkets() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	p := params.NewQueryParams(c.Ctx)
	status, err := normalizePredictMyMarketStatus(c.Ctx.URLParamDefault("status", ""))
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	betMarketSub := sqls.DB().Model(&models.PredictBet{}).
		Select("DISTINCT market_id").
		Where("user_id = ?", user.Id)

	q := p.Cnd.Build(sqls.DB().Model(&models.PredictMarket{}).Where("id IN (?)", betMarketSub))
	if status != "" {
		if status == "PENDING" {
			q = q.Where("status IN ?", []string{"CLOSED", "CLOSE"})
		} else {
			q = q.Where("status = ?", status)
		}
	}

	var list []models.PredictMarket
	if err := q.
		Order("CASE status WHEN 'OPEN' THEN 0 WHEN 'CLOSED' THEN 1 WHEN 'CLOSE' THEN 1 ELSE 2 END, close_time asc, id desc").
		Find(&list).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	count := p.Cnd.Build(sqls.DB().Model(&models.PredictMarket{}).Where("id IN (?)", betMarketSub))
	if status != "" {
		if status == "PENDING" {
			count = count.Where("status IN ?", []string{"CLOSED", "CLOSE"})
		} else {
			count = count.Where("status = ?", status)
		}
	}
	var total int64
	if err := count.Count(&total).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	marketIds := make([]int64, 0, len(list))
	for _, m := range list {
		if m.Id > 0 {
			marketIds = append(marketIds, m.Id)
		}
	}

	ctxMap := make(map[int64]models.PredictContext, len(marketIds))
	if len(marketIds) > 0 {
		var ctxList []models.PredictContext
		sqls.DB().Where("market_id in (?)", marketIds).Find(&ctxList)
		for _, mc := range ctxList {
			ctxMap[mc.MarketId] = mc
		}
	}

	scheduleMap := scheduleMetaByMarketIds(marketIds)
	betSettleResultMap := make(map[int64]string, len(marketIds))
	hasBetMap := make(map[int64]bool, len(marketIds))
	if len(marketIds) > 0 {
		type betRow struct {
			MarketId     int64
			SettleResult string
			SettleTime   int64
			CreateTime   int64
		}
		var betRows []betRow
		sqls.DB().Model(&models.PredictBet{}).
			Select("market_id, settle_result, settle_time, create_time").
			Where("user_id = ? AND market_id in (?)", user.Id, marketIds).
			Find(&betRows)

		hasWin := make(map[int64]bool, len(marketIds))
		hasLose := make(map[int64]bool, len(marketIds))
		latestScore := make(map[int64]int64, len(marketIds))
		latestVal := make(map[int64]string, len(marketIds))
		for _, br := range betRows {
			hasBetMap[br.MarketId] = true
			v := strings.ToUpper(strings.TrimSpace(br.SettleResult))
			if v == "WIN" {
				hasWin[br.MarketId] = true
				continue
			}
			if v == "LOSE" {
				hasLose[br.MarketId] = true
			}
			if v == "" {
				continue
			}
			score := br.SettleTime
			if score <= 0 {
				score = br.CreateTime
			}
			if score > latestScore[br.MarketId] {
				latestScore[br.MarketId] = score
				latestVal[br.MarketId] = v
			}
		}
		for _, m := range list {
			mid := m.Id
			if hasWin[mid] {
				betSettleResultMap[mid] = "WIN"
				continue
			}
			if hasLose[mid] {
				betSettleResultMap[mid] = "LOSE"
				continue
			}
			betSettleResultMap[mid] = latestVal[mid]
		}
	}

	respList := make([]map[string]any, 0, len(list))
	for _, m := range list {
		schedule := scheduleMap[m.Id]
		respList = append(respList, map[string]any{
			"market":          m,
			"context":         ctxMap[m.Id],
			"schedule":        schedule,
			"matchPhase":      matchPhaseFromSchedule(schedule),
			"betSettleResult": betSettleResultMap[m.Id],
			"hasBet":          hasBetMap[m.Id],
		})
	}

	ret := web.JsonData(map[string]any{
		"list":   respList,
		"total":  total,
		"status": status,
	})
	c.attachTearSettlementForMarketList(ret)
	return ret
}

func normalizePredictMyMarketStatus(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if s == "" {
		return "", nil
	}
	switch s {
	case "OPEN", "进行中", "IN_PROGRESS", "ONGOING", "ACTIVE":
		return "OPEN", nil
	case "CLOSED", "CLOSE", "待结算", "PENDING":
		return "PENDING", nil
	case "SETTLED", "已结算", "DONE":
		return "SETTLED", nil
	default:
		return "", errors.New("invalid status")
	}
}

func (c *PredictController) GetBetSettleResult() *web.JsonResult {
	ret := (&FootballController{Ctx: c.Ctx}).GetBet_settle_result()
	c.attachTearSettlementForBetSettleResult(ret)
	return ret
}

func (c *PredictController) GetHeat() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	market := &models.PredictMarket{}
	if err := sqls.DB().Take(market, "id = ?", marketId).Error; err != nil {
		return web.JsonErrorMsg("predict market not found")
	}

	options, snapshotTime, err := services.TearHeatService.ComputeHeatSnapshot(sqls.DB(), services.TearHeatSnapshotComputeRequest{
		EventType:    constants.EntityPredictMarket,
		EventId:      marketId,
		TopicId:      marketId,
		RoundId:      0,
		SnapshotType: services.TearSnapshotCkpt,
		FreezeSource: "ON_DEMAND",
	})
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	leader := ""
	maxHeat := -1.0
	for _, item := range options {
		if v, ok := item["hTotal"].(float64); ok {
			if v > maxHeat {
				maxHeat = v
				leader, _ = item["option"].(string)
			}
		}
	}

	return web.JsonData(map[string]any{
		"marketId":       marketId,
		"marketType":     market.MarketType,
		"status":         market.Status,
		"options":        options,
		"snapshotTime":   snapshotTime,
		"snapshotType":   services.TearSnapshotCkpt,
		"leaderOption":   leader,
		"totalHeatValue": maxHeat,
	})
}

func (c *PredictController) GetComments() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	cursor, _ := params.GetInt64(c.Ctx, "cursor")
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	option := strings.ToUpper(strings.TrimSpace(c.Ctx.URLParamDefault("option", "")))
	if option != "" && option != services.PredictOptionA && option != services.PredictOptionB && option != services.PredictOptionDraw {
		return web.JsonErrorMsg("option must be A, B or DRAW")
	}

	q := sqls.DB().Model(&models.Comment{}).
		Joins("JOIN t_predict_comment_meta pcm ON pcm.comment_id = t_comment.id").
		Where("t_comment.entity_type = ? AND t_comment.entity_id = ? AND t_comment.status = ? AND pcm.market_id = ?", constants.EntityPredictMarket, marketId, constants.StatusOk, marketId)
	if option != "" {
		q = q.Where("pcm.option = ?", option)
	}
	if cursor > 0 {
		q = q.Where("t_comment.id < ?", cursor)
	}

	var comments []models.Comment
	if err := q.Order("t_comment.id desc").Limit(pageSize).Find(&comments).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	nextCursor := cursor
	hasMore := false
	if len(comments) > 0 {
		nextCursor = comments[len(comments)-1].Id
		hasMore = len(comments) >= pageSize
	}

	currentUser := common.GetCurrentUser(c.Ctx)
	return web.JsonCursorData(render.BuildComments(comments, currentUser, true, false), strconv.FormatInt(nextCursor, 10), hasMore)
}

func (c *PredictController) GetCommentReplies() *web.JsonResult {
	return (&CommentController{Ctx: c.Ctx}).GetReplies()
}

func (c *PredictController) GetHeatRank() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	page, _ := params.GetInt(c.Ctx, "page")
	if page <= 0 {
		page = 1
	}
	pageSize, _ := params.GetInt(c.Ctx, "pageSize")
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	viewer := common.GetCurrentUser(c.Ctx)
	scope := strings.ToUpper(strings.TrimSpace(c.Ctx.URLParamDefault("scope", "ALL")))
	viewerOption := ""
	if viewer != nil {
		viewerOption = services.PredictCampLockService.GetOption(sqls.DB(), marketId, viewer.Id)
	}

	items, err := services.TearHeatService.GetUserHeatItems(sqls.DB(), constants.EntityPredictMarket, marketId, 0)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	filtered := make([]services.TearHeatUserItem, 0, len(items))
	for _, item := range items {
		if scope == "MY_SIDE" {
			if viewerOption == "" || !strings.EqualFold(item.Option, viewerOption) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].TotalHeat == filtered[j].TotalHeat {
			return filtered[i].UserId < filtered[j].UserId
		}
		return filtered[i].TotalHeat > filtered[j].TotalHeat
	})

	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	segment := filtered[start:end]

	userIds := make([]int64, 0, len(segment))
	for _, item := range segment {
		userIds = append(userIds, item.UserId)
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

	list := make([]map[string]any, 0, len(segment))
	for idx, item := range segment {
		u := userMap[item.UserId]
		list = append(list, map[string]any{
			"rank":        start + idx + 1,
			"userId":      item.UserId,
			"nickname":    u.Nickname,
			"avatar":      u.Avatar,
			"option":      item.Option,
			"totalHeat":   item.TotalHeat,
			"commentHeat": item.CommentHeat,
			"likeHeat":    item.LikeHeat,
			"coinHeat":    item.CoinHeat,
		})
	}

	return web.JsonData(map[string]any{
		"marketId": marketId,
		"scope":    scope,
		"myOption": viewerOption,
		"list":     list,
		"count":    len(filtered),
		"page":     page,
		"pageSize": pageSize,
	})
}

func (c *PredictController) GetHeatMe() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}

	items, err := services.TearHeatService.GetUserHeatItems(sqls.DB(), constants.EntityPredictMarket, marketId, 0)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TotalHeat == items[j].TotalHeat {
			return items[i].UserId < items[j].UserId
		}
		return items[i].TotalHeat > items[j].TotalHeat
	})

	myRank := int64(0)
	myOption := ""
	myHeat := 0.0
	myCommentHeat := 0.0
	myLikeHeat := 0.0
	myCoinHeat := 0.0
	for idx, item := range items {
		if item.UserId == user.Id {
			myRank = int64(idx + 1)
			myOption = item.Option
			myHeat = item.TotalHeat
			myCommentHeat = item.CommentHeat
			myLikeHeat = item.LikeHeat
			myCoinHeat = item.CoinHeat
			break
		}
	}

	stat := &models.TearUserEventStat{}
	_ = sqls.DB().Where("event_type = ? AND topic_id = ? AND round_id = ? AND user_id = ?", constants.EntityPredictMarket, marketId, 0, user.Id).Take(stat).Error

	return web.JsonData(map[string]any{
		"marketId":          marketId,
		"userId":            user.Id,
		"myOption":          myOption,
		"myHeat":            myHeat,
		"myRank":            myRank,
		"commentHeat":       myCommentHeat,
		"likeHeat":          myLikeHeat,
		"coinHeat":          myCoinHeat,
		"myActionCount":     stat.ActionCount,
		"myCommentCount":    stat.CommentCount + stat.ReplyCount,
		"receivedLikeCount": stat.LikeCount,
		"myBetAmount":       stat.BetAmount,
	})
}

func (c *PredictController) GetOddsCurrent() *web.JsonResult {
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}
	market := &models.PredictMarket{}
	if err := sqls.DB().Take(market, "id = ?", marketId).Error; err != nil {
		return web.JsonErrorMsg("predict market not found")
	}

	oddsA, oddsB, effA, effB, total := services.CalcClampedOdds(market.BaseA, market.BaseB, market.PoolA, market.PoolB)
	ret := map[string]any{
		"marketId":     market.Id,
		"marketType":   market.MarketType,
		"status":       market.Status,
		"oddsA":        oddsA,
		"oddsB":        oddsB,
		"effA":         effA,
		"effB":         effB,
		"totalEffPool": total,
	}
	if services.NormalizePredictMarketType(market.MarketType) == services.PredictMarketType1x2 {
		oddsA3, oddsB3, oddsDraw, effA3, effB3, effDraw, total3 := services.CalcClampedOdds3(
			market.BaseA,
			market.BaseB,
			market.BaseDraw,
			market.PoolA,
			market.PoolB,
			market.PoolDraw,
		)
		ret["oddsA"] = oddsA3
		ret["oddsB"] = oddsB3
		ret["oddsDraw"] = oddsDraw
		ret["effA"] = effA3
		ret["effB"] = effB3
		ret["effDraw"] = effDraw
		ret["totalEffPool"] = total3
	}
	return web.JsonData(ret)
}

func (c *PredictController) PostContextUpdate() *web.JsonResult {
	return (&FootballController{Ctx: c.Ctx}).PostPredict_contextUpdate()
}

func (c *PredictController) GetContextHot() *web.JsonResult {
	return (&FootballController{Ctx: c.Ctx}).GetPredict_contextHot()
}

func (c *PredictController) GetTagsHot() *web.JsonResult {
	return (&FootballController{Ctx: c.Ctx}).GetPredict_tagsHot()
}

func (c *PredictController) PostCoinBet() *web.JsonResult {
	return (&CoinController{Ctx: c.Ctx}).PostBet()
}

func (c *PredictController) PostCoinSettle() *web.JsonResult {
	return (&CoinController{Ctx: c.Ctx}).PostSettle()
}

func (c *PredictController) PostTearSettle() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	marketId, _ := params.GetInt64(c.Ctx, "marketId")
	if marketId <= 0 {
		return web.JsonErrorMsg("marketId is required")
	}

	log, err := services.PredictCommentRewardService.RunForMarket(marketId, false)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	market := &models.PredictMarket{}
	if err := sqls.DB().Take(market, "id = ?", marketId).Error; err != nil {
		return web.JsonErrorMsg(err.Error())
	}

	return web.JsonData(map[string]any{
		"marketId":       marketId,
		"rewardLog":      log,
		"tearSettlement": c.buildTearSettlementInfo(market, log),
	})
}

func (c *PredictController) GetCoinLeaderboard() *web.JsonResult {
	return (&CoinController{Ctx: c.Ctx}).GetLeaderboard()
}

func (c *PredictController) PostCommentCreate() *web.JsonResult {
	return (&CommentController{Ctx: c.Ctx}).PostCreate()
}

func (c *PredictController) PostCommentReply() *web.JsonResult {
	return (&CommentController{Ctx: c.Ctx}).PostReply()
}

func (c *PredictController) PostLike() *web.JsonResult {
	return (&LikeController{Ctx: c.Ctx}).PostLike()
}

func (c *PredictController) PostUnlike() *web.JsonResult {
	return (&LikeController{Ctx: c.Ctx}).PostUnlike()
}

func tearTsToSeconds(ts int64) int64 {
	if ts <= 0 {
		return ts
	}
	if ts > 1_000_000_000_000 {
		return ts / 1000
	}
	return ts
}

func (c *PredictController) buildTearSettlementInfo(market *models.PredictMarket, log *models.PredictCommentRewardLog) map[string]any {
	now := tearTsToSeconds(dates.NowTimestamp())

	settledAt := int64(0)
	deadlineAt := int64(0)
	status := "NONE"
	reason := ""
	rewardLogId := int64(0)
	winnerOption := ""

	if market != nil {
		settledAt = tearTsToSeconds(market.ResolvedAt)
		if settledAt <= 0 {
			settledAt = tearTsToSeconds(market.UpdateTime)
		}
		if settledAt > 0 {
			deadlineAt = settledAt + 3600
		}
	}
	if log != nil {
		rewardLogId = log.Id
		winnerOption = log.WinnerOption
		status = strings.ToUpper(strings.TrimSpace(log.Status))
		reason = log.Reason
		if log.SettledAt > 0 {
			settledAt = tearTsToSeconds(log.SettledAt)
		}
		if log.DeadlineAt > 0 {
			deadlineAt = tearTsToSeconds(log.DeadlineAt)
		}
	}

	canSettle := false
	if market == nil || market.Status != "SETTLED" {
		reason = "MARKET_NOT_SETTLED"
	} else {
		switch status {
		case "PAID":
			reason = "ALREADY_PAID"
		case "PROCESSING":
			reason = "PROCESSING"
		case "EXPIRED":
			reason = "DEADLINE_EXPIRED"
		case "FAILED":
			reason = "FAILED_NEED_RETRY"
		default:
			if deadlineAt > 0 && now > deadlineAt {
				reason = "DEADLINE_EXPIRED"
			} else {
				canSettle = true
			}
		}
	}

	remain := int64(0)
	if deadlineAt > now {
		remain = deadlineAt - now
	}

	return map[string]any{
		"canSettle":     canSettle,
		"status":        status,
		"reason":        reason,
		"settledAt":     settledAt,
		"deadlineAt":    deadlineAt,
		"remainSeconds": remain,
		"rewardLogId":   rewardLogId,
		"winnerOption":  winnerOption,
	}
}

func (c *PredictController) attachTearSettlementForMarketList(ret *web.JsonResult) {
	if ret == nil || !ret.Success {
		return
	}
	data, ok := ret.Data.(map[string]any)
	if !ok {
		return
	}
	rawList, ok := data["list"]
	if !ok {
		return
	}
	list, ok := rawList.([]map[string]any)
	if !ok {
		return
	}

	marketIds := make([]int64, 0, len(list))
	marketById := make(map[int64]*models.PredictMarket, len(list))
	for i := range list {
		m, ok := list[i]["market"].(models.PredictMarket)
		if !ok || m.Id <= 0 {
			continue
		}
		mm := m
		marketById[m.Id] = &mm
		marketIds = append(marketIds, m.Id)
	}

	logByMarket := map[int64]*models.PredictCommentRewardLog{}
	if len(marketIds) > 0 {
		var logs []models.PredictCommentRewardLog
		if err := sqls.DB().Where("market_id IN ?", marketIds).Find(&logs).Error; err == nil {
			for i := range logs {
				lg := logs[i]
				logByMarket[lg.MarketId] = &lg
			}
		}
	}

	for i := range list {
		m, ok := list[i]["market"].(models.PredictMarket)
		if !ok || m.Id <= 0 {
			continue
		}
		list[i]["tearSettlement"] = c.buildTearSettlementInfo(marketById[m.Id], logByMarket[m.Id])
	}
	data["list"] = list
	ret.Data = data
}

func (c *PredictController) attachTearSettlementForBetSettleResult(ret *web.JsonResult) {
	if ret == nil || !ret.Success {
		return
	}
	data, ok := ret.Data.(map[string]any)
	if !ok {
		return
	}
	marketId, ok := data["marketId"].(int64)
	if !ok || marketId <= 0 {
		return
	}
	market := &models.PredictMarket{}
	if err := sqls.DB().Take(market, "id = ?", marketId).Error; err != nil {
		return
	}
	log := &models.PredictCommentRewardLog{}
	if err := sqls.DB().Take(log, "market_id = ?", marketId).Error; err != nil {
		log = nil
	}
	data["tearSettlement"] = c.buildTearSettlementInfo(market, log)
	ret.Data = data
}
