package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/config"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	aiPushSceneWin               = "win"
	aiPushSceneLose              = "lose"
	aiPushSceneWinStreak         = "win_streak"
	aiPushSceneLoseStreak        = "lose_streak"
	aiPushSceneIdleBiggestWin    = "idle_biggest_win"
	aiPushSceneIdleFirstEgg      = "idle_first_egg"
	aiPushSceneIdleLongestStreak = "idle_longest_streak"

	aiMemoryVoteWinStreak     = "vote_win_streak"
	aiMemoryVoteLoseStreak    = "vote_lose_streak"
	aiMemoryBiggestWinAmount  = "biggest_win_amount"
	aiMemoryBiggestWinEvent   = "biggest_win_event"
	aiMemoryLongestWinStreak  = "longest_win_streak"
	aiMemoryLongestLoseStreak = "longest_lose_streak"
	aiMemoryFirstEggTurtle    = "first_egg_turtle"
)

var AIPushService = newAIPushService()

func newAIPushService() *aiPushService {
	return &aiPushService{hub: newAIPushHub()}
}

type aiPushService struct {
	hub *aiPushHub
}

type AIPushMessage struct {
	Id          int64  `json:"id"`
	Scene       string `json:"scene"`
	Content     string `json:"content"`
	ContextType string `json:"contextType"`
	ContextId   int64  `json:"contextId"`
	CreateTime  int64  `json:"createTime"`
}

type AIPushUnreadResult struct {
	Results []AIPushMessage `json:"results"`
}

type AIPushReadResult struct {
	Updated int64 `json:"updated"`
}

type AIPresenceForm struct {
	Page   string `json:"page" form:"page"`
	Active bool   `json:"active" form:"active"`
}

type AIPresenceResult struct {
	UserId     int64  `json:"userId"`
	Page       string `json:"page"`
	Active     bool   `json:"active"`
	LastSeenAt int64  `json:"lastSeenAt"`
}

type AISettlementEvent struct {
	UserId       int64
	BizType      string
	BizId        string
	ContextType  string
	ContextId    int64
	ProfitAmount int64
	SettleResult string
	EventTitle   string
	SettledAt    int64
}

type AIPushSettlementResult struct {
	Generated bool              `json:"generated"`
	Scene     string            `json:"scene,omitempty"`
	Message   *models.AIMessage `json:"message,omitempty"`
	Skipped   string            `json:"skipped,omitempty"`
}

func (s *aiPushService) ListUnread(userId int64, limit int) (*AIPushUnreadResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var messages []models.AIMessage
	if err := sqls.DB().Where("user_id = ? and role = ? and status = 0 and scene in ?", userId, aiRoleAssistant, aiPushScenes()).
		Order("create_time asc, id asc").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return &AIPushUnreadResult{Results: buildAIPushMessages(messages)}, nil
}

func (s *aiPushService) MarkRead(userId int64, ids []int64) (*AIPushReadResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	ids = normalizeIds(ids)
	if len(ids) == 0 {
		return &AIPushReadResult{}, nil
	}
	tx := sqls.DB().Model(&models.AIMessage{}).
		Where("user_id = ? and id in ? and role = ? and status = 0", userId, ids, aiRoleAssistant).
		Update("status", 1)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &AIPushReadResult{Updated: tx.RowsAffected}, nil
}

func (s *aiPushService) UpdatePresence(userId int64, form AIPresenceForm) (*AIPresenceResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	now := dates.NowTimestamp()
	page := strings.TrimSpace(form.Page)
	if len([]rune(page)) > 64 {
		page = string([]rune(page)[:64])
	}
	p := &models.UserAIPresence{
		UserId:      userId,
		Page:        page,
		Active:      form.Active,
		FirstSeenAt: now,
		LastSeenAt:  now,
		UpdateTime:  now,
	}
	if err := sqls.DB().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"page":         page,
			"active":       form.Active,
			"last_seen_at": now,
			"update_time":  now,
		}),
	}).Create(p).Error; err != nil {
		return nil, err
	}
	return &AIPresenceResult{UserId: userId, Page: page, Active: form.Active, LastSeenAt: now}, nil
}

func (s *aiPushService) UpdateAIInteract(userId int64) error {
	if userId <= 0 {
		return nil
	}
	now := dates.NowTimestamp()
	return sqls.DB().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"last_ai_interact_at": now,
			"last_seen_at":        now,
			"active":              true,
			"update_time":         now,
		}),
	}).Create(&models.UserAIPresence{
		UserId:           userId,
		Active:           true,
		FirstSeenAt:      now,
		LastSeenAt:       now,
		LastAIInteractAt: now,
		UpdateTime:       now,
	}).Error
}

func (s *aiPushService) OnSettlement(event AISettlementEvent) (*AIPushSettlementResult, error) {
	if event.UserId <= 0 {
		return nil, errors.New("userId is required")
	}
	event.BizType = strings.TrimSpace(event.BizType)
	event.BizId = strings.TrimSpace(event.BizId)
	if event.BizType == "" || event.BizId == "" {
		return nil, errors.New("bizType and bizId are required")
	}
	if event.ContextType == "" {
		event.ContextType = "predict_market"
	}
	if event.SettledAt <= 0 {
		event.SettledAt = dates.NowTimestamp()
	}
	if event.ProfitAmount == 0 || event.SettleResult == "refund" || event.SettleResult == "canceled" {
		return &AIPushSettlementResult{Skipped: "no_scene"}, nil
	}

	var message *models.AIMessage
	var scene string
	var skipped string
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		memory, err := s.updateSettlementMemoryTx(tx, event)
		if err != nil {
			return err
		}
		scene = decideSettlementScene(event, memory)
		if scene == "" {
			skipped = "no_scene"
			return nil
		}
		requestId := fmt.Sprintf("settle:%s:%s:%s", event.BizType, event.BizId, scene)
		existing := &models.AIMessage{}
		if err := tx.Where("user_id = ? and request_id = ?", event.UserId, requestId).Take(existing).Error; err == nil {
			message = existing
			skipped = "duplicate"
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		placeholders := map[string]string{
			"amount": fmt.Sprintf("%d", absInt64(event.ProfitAmount)),
			"n":      fmt.Sprintf("%d", memory.WinStreak),
			"event":  event.EventTitle,
		}
		if scene == aiPushSceneLoseStreak {
			placeholders["n"] = fmt.Sprintf("%d", memory.LoseStreak)
		}
		rendered, err := s.selectAndRenderTemplateTx(tx, event.UserId, scene, placeholders)
		if err != nil {
			return err
		}
		now := dates.NowTimestamp()
		msg := &models.AIMessage{
			UserId:      event.UserId,
			Role:        aiRoleAssistant,
			Scene:       scene,
			Content:     rendered.Content,
			StaminaCost: 0,
			ContextType: event.ContextType,
			ContextId:   event.ContextId,
			TemplateId:  rendered.TemplateId,
			RequestId:   requestId,
			Status:      0,
			CreateTime:  now,
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		if rendered.TemplateId > 0 {
			if err := s.recordTemplateViewTx(tx, event.UserId, rendered.TemplateId, now); err != nil {
				return err
			}
		}
		message = msg
		return nil
	})
	if err != nil {
		return nil, err
	}
	if message != nil && skipped == "" {
		s.hub.Publish(message.UserId, toAIPushMessage(*message))
		return &AIPushSettlementResult{Generated: true, Scene: scene, Message: message}, nil
	}
	return &AIPushSettlementResult{Generated: false, Scene: scene, Message: message, Skipped: skipped}, nil
}

func (s *aiPushService) Stream(userId int64, lastEventId int64) (<-chan AIPushMessage, func()) {
	ch, unregister := s.hub.Register(userId)
	go func() {
		if lastEventId <= 0 {
			return
		}
		var missed []models.AIMessage
		_ = sqls.DB().Where("user_id = ? and id > ? and role = ? and status = 0 and scene in ?", userId, lastEventId, aiRoleAssistant, aiPushScenes()).
			Order("id asc").
			Limit(50).
			Find(&missed).Error
		for _, msg := range missed {
			select {
			case ch <- toAIPushMessage(msg):
			default:
				return
			}
		}
	}()
	return ch, unregister
}

func (s *aiPushService) CronTick() error {
	now := dates.NowTimestamp()
	conf := config.Instance
	if conf == nil || !conf.AIChat.Enabled {
		return nil
	}
	idleTriggerSeconds := int64(conf.AIChat.IdleTriggerMinutes * 60)
	if idleTriggerSeconds <= 0 {
		idleTriggerSeconds = 600
	}
	idleCooldownSeconds := int64(conf.AIChat.IdlePushCooldownMinutes * 60)
	if idleCooldownSeconds <= 0 {
		idleCooldownSeconds = 7200
	}
	var candidates []models.UserAIPresence
	if err := sqls.DB().Where("active = ? and last_seen_at >= ? and last_ai_interact_at <= ? and (last_idle_push_at = 0 or last_idle_push_at <= ?)",
		true, now-60, now-idleTriggerSeconds, now-idleCooldownSeconds).
		Limit(100).
		Find(&candidates).Error; err != nil {
		return err
	}
	for _, p := range candidates {
		if err := s.pushIdleForUser(p.UserId, now); err != nil {
			continue
		}
	}
	return nil
}

func (s *aiPushService) pushIdleForUser(userId int64, now int64) error {
	var message *models.AIMessage
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		p := &models.UserAIPresence{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userId).Take(p).Error; err != nil {
			return err
		}
		today := todayCST()
		if p.IdlePushCountDate != today {
			p.IdlePushCountDate = today
			p.IdlePushCount = 0
		}
		if p.IdlePushCount >= idlePushDailyLimit() {
			return nil
		}
		scene, placeholders, memoryKey, err := s.decideIdleSceneTx(tx, userId)
		if err != nil || scene == "" {
			return err
		}
		requestId := fmt.Sprintf("idle:%d:%s:%d", userId, scene, today)
		existing := &models.AIMessage{}
		if err := tx.Where("user_id = ? and request_id = ?", userId, requestId).Take(existing).Error; err == nil {
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		rendered, err := s.selectAndRenderTemplateTx(tx, userId, scene, placeholders)
		if err != nil {
			return err
		}
		msg := &models.AIMessage{
			UserId:      userId,
			Role:        aiRoleAssistant,
			Scene:       scene,
			Content:     rendered.Content,
			StaminaCost: 0,
			TemplateId:  rendered.TemplateId,
			RequestId:   requestId,
			Status:      0,
			CreateTime:  now,
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		if rendered.TemplateId > 0 {
			if err := s.recordTemplateViewTx(tx, userId, rendered.TemplateId, now); err != nil {
				return err
			}
		}
		p.LastIdlePushAt = now
		p.IdlePushCount++
		p.UpdateTime = now
		if err := tx.Save(p).Error; err != nil {
			return err
		}
		if memoryKey != "" {
			if err := s.touchMemoryLastPushedAtTx(tx, userId, memoryKey, now); err != nil {
				return err
			}
		}
		message = msg
		return nil
	})
	if err != nil {
		return err
	}
	if message != nil {
		s.hub.Publish(userId, toAIPushMessage(*message))
	}
	return nil
}

func (s *aiPushService) decideIdleSceneTx(tx *gorm.DB, userId int64) (string, map[string]string, string, error) {
	amount, err := s.getMemoryStringForUpdate(tx, userId, aiMemoryBiggestWinAmount)
	if err != nil {
		return "", nil, "", err
	}
	event, err := s.getMemoryStringForUpdate(tx, userId, aiMemoryBiggestWinEvent)
	if err != nil {
		return "", nil, "", err
	}
	if amount != "" && event != "" {
		return aiPushSceneIdleBiggestWin, map[string]string{"amount": amount, "event": event}, aiMemoryBiggestWinAmount, nil
	}
	turtle, err := s.getMemoryStringForUpdate(tx, userId, aiMemoryFirstEggTurtle)
	if err != nil {
		return "", nil, "", err
	}
	if turtle != "" {
		return aiPushSceneIdleFirstEgg, map[string]string{"turtle_name": turtle}, aiMemoryFirstEggTurtle, nil
	}
	streak, err := s.getMemoryStringForUpdate(tx, userId, aiMemoryLongestWinStreak)
	if err != nil {
		return "", nil, "", err
	}
	if streak != "" && streak != "0" {
		return aiPushSceneIdleLongestStreak, map[string]string{"n": streak}, aiMemoryLongestWinStreak, nil
	}
	return "", nil, "", nil
}

type settlementMemorySnapshot struct {
	WinStreak  int
	LoseStreak int
}

func (s *aiPushService) updateSettlementMemoryTx(tx *gorm.DB, event AISettlementEvent) (*settlementMemorySnapshot, error) {
	winStreak := 0
	loseStreak := 0
	if event.ProfitAmount > 0 {
		current, err := s.getMemoryIntForUpdate(tx, event.UserId, aiMemoryVoteWinStreak)
		if err != nil {
			return nil, err
		}
		winStreak = current + 1
		if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryVoteWinStreak, fmt.Sprintf("%d", winStreak), eventMeta(event)); err != nil {
			return nil, err
		}
		if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryVoteLoseStreak, "0", eventMeta(event)); err != nil {
			return nil, err
		}
		longest, err := s.getMemoryIntForUpdate(tx, event.UserId, aiMemoryLongestWinStreak)
		if err != nil {
			return nil, err
		}
		if winStreak > longest {
			if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryLongestWinStreak, fmt.Sprintf("%d", winStreak), eventMeta(event)); err != nil {
				return nil, err
			}
		}
		biggest, err := s.getMemoryInt64ForUpdate(tx, event.UserId, aiMemoryBiggestWinAmount)
		if err != nil {
			return nil, err
		}
		if event.ProfitAmount > biggest {
			if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryBiggestWinAmount, fmt.Sprintf("%d", event.ProfitAmount), eventMeta(event)); err != nil {
				return nil, err
			}
			if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryBiggestWinEvent, event.EventTitle, eventMeta(event)); err != nil {
				return nil, err
			}
		}
	} else if event.ProfitAmount < 0 {
		current, err := s.getMemoryIntForUpdate(tx, event.UserId, aiMemoryVoteLoseStreak)
		if err != nil {
			return nil, err
		}
		loseStreak = current + 1
		if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryVoteLoseStreak, fmt.Sprintf("%d", loseStreak), eventMeta(event)); err != nil {
			return nil, err
		}
		if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryVoteWinStreak, "0", eventMeta(event)); err != nil {
			return nil, err
		}
		longest, err := s.getMemoryIntForUpdate(tx, event.UserId, aiMemoryLongestLoseStreak)
		if err != nil {
			return nil, err
		}
		if loseStreak > longest {
			if err := s.upsertMemoryTx(tx, event.UserId, aiMemoryLongestLoseStreak, fmt.Sprintf("%d", loseStreak), eventMeta(event)); err != nil {
				return nil, err
			}
		}
	}
	return &settlementMemorySnapshot{WinStreak: winStreak, LoseStreak: loseStreak}, nil
}

func decideSettlementScene(event AISettlementEvent, memory *settlementMemorySnapshot) string {
	if memory == nil {
		return ""
	}
	if memory.LoseStreak >= 5 {
		return aiPushSceneLoseStreak
	}
	if memory.WinStreak >= 3 {
		return aiPushSceneWinStreak
	}
	if event.ProfitAmount < 0 {
		return aiPushSceneLose
	}
	if event.ProfitAmount > 0 {
		return aiPushSceneWin
	}
	return ""
}

type renderedTemplate struct {
	TemplateId int64
	Content    string
}

func (s *aiPushService) selectAndRenderTemplateTx(tx *gorm.DB, userId int64, scene string, placeholders map[string]string) (*renderedTemplate, error) {
	var templates []models.DialogueTemplate
	if err := tx.Where("scene = ? and enabled = ?", scene, true).Find(&templates).Error; err != nil {
		return nil, err
	}
	if len(templates) > 0 {
		for _, tmpl := range s.orderTemplatesByExposureTx(tx, userId, templates) {
			content, ok := renderAITemplate(tmpl.Content, placeholders)
			if !ok {
				continue
			}
			return &renderedTemplate{TemplateId: tmpl.Id, Content: content}, nil
		}
	}
	fallback, ok := renderAITemplate(aiPushFallbackTemplate(scene), placeholders)
	if !ok {
		return nil, errors.New("ai push template cannot render")
	}
	return &renderedTemplate{Content: fallback}, nil
}

func (s *aiPushService) orderTemplatesByExposureTx(tx *gorm.DB, userId int64, templates []models.DialogueTemplate) []models.DialogueTemplate {
	ids := make([]int64, 0, len(templates))
	for _, tmpl := range templates {
		ids = append(ids, tmpl.Id)
	}
	var views []models.TemplateUserView
	_ = tx.Where("user_id = ? and template_id in ?", userId, ids).Find(&views).Error
	viewCount := make(map[int64]int, len(views))
	for _, view := range views {
		viewCount[view.TemplateId] = view.ViewCount
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + userId))
	ret := append([]models.DialogueTemplate(nil), templates...)
	r.Shuffle(len(ret), func(i, j int) {
		ret[i], ret[j] = ret[j], ret[i]
	})
	sort.SliceStable(ret, func(i, j int) bool {
		vi := viewCount[ret[i].Id]
		vj := viewCount[ret[j].Id]
		if vi != vj {
			return vi < vj
		}
		wi := ret[i].Weight
		wj := ret[j].Weight
		if wi <= 0 {
			wi = 1
		}
		if wj <= 0 {
			wj = 1
		}
		return wi > wj
	})
	return ret
}

func (s *aiPushService) recordTemplateViewTx(tx *gorm.DB, userId int64, templateId int64, now int64) error {
	if templateId <= 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "template_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"view_count":     gorm.Expr("t_template_user_view.view_count + 1"),
			"last_viewed_at": now,
		}),
	}).Create(&models.TemplateUserView{
		UserId:       userId,
		TemplateId:   templateId,
		ViewCount:    1,
		LastViewedAt: now,
	}).Error
}

func (s *aiPushService) getMemoryIntForUpdate(tx *gorm.DB, userId int64, key string) (int, error) {
	v, err := s.getMemoryStringForUpdate(tx, userId, key)
	if err != nil {
		return 0, err
	}
	var n int
	_, _ = fmt.Sscanf(v, "%d", &n)
	return n, nil
}

func (s *aiPushService) getMemoryInt64ForUpdate(tx *gorm.DB, userId int64, key string) (int64, error) {
	v, err := s.getMemoryStringForUpdate(tx, userId, key)
	if err != nil {
		return 0, err
	}
	var n int64
	_, _ = fmt.Sscanf(v, "%d", &n)
	return n, nil
}

func (s *aiPushService) getMemoryStringForUpdate(tx *gorm.DB, userId int64, key string) (string, error) {
	mem := &models.UserAIMemory{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? and flag_key = ?", userId, key).Take(mem).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(mem.FlagValue), nil
}

func (s *aiPushService) upsertMemoryTx(tx *gorm.DB, userId int64, key string, value string, meta string) error {
	now := dates.NowTimestamp()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "flag_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"flag_value":  value,
			"flag_meta":   meta,
			"update_time": now,
		}),
	}).Create(&models.UserAIMemory{
		UserId:     userId,
		FlagKey:    key,
		FlagValue:  value,
		FlagMeta:   meta,
		CreateTime: now,
		UpdateTime: now,
	}).Error
}

func (s *aiPushService) touchMemoryLastPushedAtTx(tx *gorm.DB, userId int64, key string, now int64) error {
	mem := &models.UserAIMemory{}
	if err := tx.Where("user_id = ? and flag_key = ?", userId, key).Take(mem).Error; err != nil {
		return err
	}
	meta := map[string]any{}
	if strings.TrimSpace(mem.FlagMeta) != "" {
		_ = json.Unmarshal([]byte(mem.FlagMeta), &meta)
	}
	meta["lastPushedAt"] = now
	b, _ := json.Marshal(meta)
	return tx.Model(mem).Updates(map[string]any{
		"flag_meta":   string(b),
		"update_time": now,
	}).Error
}

type aiPushHub struct {
	mu    sync.RWMutex
	conns map[int64]map[chan AIPushMessage]struct{}
}

func newAIPushHub() *aiPushHub {
	return &aiPushHub{conns: make(map[int64]map[chan AIPushMessage]struct{})}
}

func (h *aiPushHub) Register(userId int64) (chan AIPushMessage, func()) {
	ch := make(chan AIPushMessage, 16)
	h.mu.Lock()
	if h.conns[userId] == nil {
		h.conns[userId] = make(map[chan AIPushMessage]struct{})
	}
	h.conns[userId][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if conns := h.conns[userId]; conns != nil {
			delete(conns, ch)
			if len(conns) == 0 {
				delete(h.conns, userId)
			}
		}
		h.mu.Unlock()
	}
}

func (h *aiPushHub) Publish(userId int64, msg AIPushMessage) {
	h.mu.RLock()
	conns := h.conns[userId]
	for ch := range conns {
		select {
		case ch <- msg:
		default:
		}
	}
	h.mu.RUnlock()
}

func aiPushScenes() []string {
	return []string{
		aiPushSceneWin,
		aiPushSceneLose,
		aiPushSceneWinStreak,
		aiPushSceneLoseStreak,
		aiPushSceneIdleBiggestWin,
		aiPushSceneIdleFirstEgg,
		aiPushSceneIdleLongestStreak,
	}
}

func normalizeIds(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	var ret []int64
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ret = append(ret, id)
	}
	return ret
}

func buildAIPushMessages(messages []models.AIMessage) []AIPushMessage {
	ret := make([]AIPushMessage, 0, len(messages))
	for _, msg := range messages {
		ret = append(ret, toAIPushMessage(msg))
	}
	return ret
}

func toAIPushMessage(msg models.AIMessage) AIPushMessage {
	return AIPushMessage{
		Id:          msg.Id,
		Scene:       msg.Scene,
		Content:     msg.Content,
		ContextType: msg.ContextType,
		ContextId:   msg.ContextId,
		CreateTime:  msg.CreateTime,
	}
}

func renderAITemplate(content string, placeholders map[string]string) (string, bool) {
	rendered := content
	for key, value := range placeholders {
		rendered = strings.ReplaceAll(rendered, "{"+key+"}", value)
	}
	if strings.Contains(rendered, "{") || strings.Contains(rendered, "}") {
		return "", false
	}
	rendered = strings.TrimSpace(rendered)
	return rendered, rendered != ""
}

func aiPushFallbackTemplate(scene string) string {
	switch scene {
	case aiPushSceneWin:
		return "{amount} 入账。低调。"
	case aiPushSceneLose:
		return "这场亏了 {amount}，小龟建议下一把慢一点。"
	case aiPushSceneWinStreak:
		return "{n} 连了。小龟先帮你记一笔。"
	case aiPushSceneLoseStreak:
		return "{n} 连没中，先歇一歇也可以。"
	case aiPushSceneIdleBiggestWin:
		return "还记得 {event} 那次赢了 {amount} 吗？小龟还记着。"
	case aiPushSceneIdleFirstEgg:
		return "你第一次开蛋拿到的是 {turtle_name}，小龟还挺怀念。"
	case aiPushSceneIdleLongestStreak:
		return "你最长连胜是 {n} 场，小龟的小本本还记着。"
	default:
		return "小龟给你留了一条新消息。"
	}
}

func eventMeta(event AISettlementEvent) string {
	b, _ := json.Marshal(map[string]any{
		"bizType":      event.BizType,
		"bizId":        event.BizId,
		"contextType":  event.ContextType,
		"contextId":    event.ContextId,
		"profitAmount": event.ProfitAmount,
		"settleResult": event.SettleResult,
		"eventTitle":   event.EventTitle,
		"settledAt":    event.SettledAt,
	})
	return string(b)
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func todayCST() int64 {
	return int64(biztime.DayNameCST(time.Now()))
}

func idlePushDailyLimit() int {
	if config.Instance != nil && config.Instance.AIChat.IdlePushDailyLimit > 0 {
		return config.Instance.AIChat.IdlePushDailyLimit
	}
	return 2
}
