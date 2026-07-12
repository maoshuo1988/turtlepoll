package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/repositories"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

const (
	MessageNotifyTemplateDisabled = 0
	MessageNotifyTemplateEnabled  = 1

	MessageNotifyStatusUnread = 0
	MessageNotifyStatusRead   = 1
)

const (
	MessageNotifyBusinessDarkMarket      = "dark_market"
	MessageNotifyBusinessTearSquare      = "tear_square"
	MessageNotifyBusinessIntel           = "intel"
	MessageNotifyBusinessSystem          = "system"
	MessageNotifyBusinessReward          = "reward"
	MessageNotifyBusinessUndergroundBank = "underground_bank"
)

var MessageNotifyService = newMessageNotifyService()

func newMessageNotifyService() *messageNotifyService {
	return &messageNotifyService{}
}

type messageNotifyService struct{}

type MessageNotifyPushInput struct {
	BusinessCode   string
	TemplateCode   string
	UserId         int64
	Params         map[string]string
	ExtraData      map[string]any
	BizId          string
	IdempotencyKey string
}

type MessageNotifyPushResult struct {
	RecordId int64                           `json:"recordId"`
	Created  bool                            `json:"created"`
	Record   *models.UserMessageNotifyRecord `json:"record"`
}

type MessageNotifyListResult struct {
	Results     []models.UserMessageNotifyRecord `json:"results"`
	Cursor      string                           `json:"cursor"`
	HasMore     bool                             `json:"hasMore"`
	UnreadCount int64                            `json:"unreadCount"`
}

type MessageNotifyUnreadCountResult struct {
	TotalUnread    int64            `json:"totalUnread"`
	BusinessUnread map[string]int64 `json:"businessUnread"`
}

type MessageNotifyReadResult struct {
	Updated bool                            `json:"updated"`
	Record  *models.UserMessageNotifyRecord `json:"record"`
}

func (s *messageNotifyService) PushByTemplate(input MessageNotifyPushInput) (*MessageNotifyPushResult, error) {
	if config.Instance != nil && config.Instance.MessageNotify.Enabled != nil && !*config.Instance.MessageNotify.Enabled {
		return nil, errors.New("MESSAGE_NOTIFY_DISABLED")
	}
	input.BusinessCode = strings.TrimSpace(input.BusinessCode)
	input.TemplateCode = strings.TrimSpace(input.TemplateCode)
	input.BizId = strings.TrimSpace(input.BizId)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.UserId <= 0 {
		return nil, errors.New("USER_ID_REQUIRED")
	}
	if input.BusinessCode == "" || input.TemplateCode == "" {
		return nil, errors.New("TEMPLATE_NOT_FOUND")
	}

	if input.IdempotencyKey != "" {
		if existing := repositories.UserMessageNotifyRecordRepository.FindOne(sqls.DB(), sqls.NewCnd().
			Eq("user_id", input.UserId).
			Eq("idempotency_key", input.IdempotencyKey)); existing != nil {
			return &MessageNotifyPushResult{RecordId: existing.Id, Created: false, Record: existing}, nil
		}
	}

	tpl := repositories.MessageNotifyTemplateRepository.GetByCode(sqls.DB(), input.BusinessCode, input.TemplateCode)
	if tpl == nil {
		return nil, errors.New("TEMPLATE_NOT_FOUND")
	}
	if tpl.Status != MessageNotifyTemplateEnabled {
		return nil, errors.New("TEMPLATE_DISABLED")
	}
	if input.Params == nil {
		input.Params = map[string]string{}
	}
	required, err := parseTemplateParams(tpl.TemplateParams)
	if err != nil {
		return nil, err
	}
	for _, key := range required {
		if strings.TrimSpace(input.Params[key]) == "" {
			return nil, fmt.Errorf("TEMPLATE_PARAM_MISSING:%s", key)
		}
	}

	subject, err := renderMessageNotifyTemplate(tpl.SubjectTemplate, input.Params)
	if err != nil {
		return nil, err
	}
	body, err := renderMessageNotifyTemplate(tpl.BodyTemplate, input.Params)
	if err != nil {
		return nil, err
	}
	detailUrl, err := renderMessageNotifyTemplate(tpl.DetailUrlTemplate, input.Params)
	if err != nil {
		return nil, err
	}

	now := dates.NowTimestamp()
	record := &models.UserMessageNotifyRecord{
		BusinessCode:   input.BusinessCode,
		TemplateCode:   input.TemplateCode,
		TemplateId:     tpl.Id,
		UserId:         input.UserId,
		Subject:        subject,
		Body:           body,
		DetailUrl:      detailUrl,
		Status:         MessageNotifyStatusUnread,
		TemplateParams: toJsonString(input.Params),
		ExtraData:      toJsonString(input.ExtraData),
		BizId:          input.BizId,
		IdempotencyKey: input.IdempotencyKey,
		CreateTime:     now,
		UpdateTime:     now,
	}
	if err := repositories.UserMessageNotifyRecordRepository.Create(sqls.DB(), record); err != nil {
		if input.IdempotencyKey != "" {
			if existing := repositories.UserMessageNotifyRecordRepository.FindOne(sqls.DB(), sqls.NewCnd().
				Eq("user_id", input.UserId).
				Eq("idempotency_key", input.IdempotencyKey)); existing != nil {
				return &MessageNotifyPushResult{RecordId: existing.Id, Created: false, Record: existing}, nil
			}
		}
		return nil, err
	}
	return &MessageNotifyPushResult{RecordId: record.Id, Created: true, Record: record}, nil
}

func (s *messageNotifyService) List(userId int64, businessCode string, status *int, cursor int64, limit int) (*MessageNotifyListResult, error) {
	if userId <= 0 {
		return nil, errors.New("USER_ID_REQUIRED")
	}
	limit = normalizeMessageNotifyLimit(limit)
	cnd := sqls.NewCnd().Eq("user_id", userId).Desc("id").Limit(limit)
	if strings.TrimSpace(businessCode) != "" {
		cnd.Eq("business_code", strings.TrimSpace(businessCode))
	}
	if status != nil {
		cnd.Eq("status", *status)
	}
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	list := repositories.UserMessageNotifyRecordRepository.Find(sqls.DB(), cnd)
	nextCursor := cursor
	hasMore := false
	if len(list) > 0 {
		nextCursor = list[len(list)-1].Id
		hasMore = len(list) == limit
	}
	return &MessageNotifyListResult{
		Results:     list,
		Cursor:      fmt.Sprintf("%d", nextCursor),
		HasMore:     hasMore,
		UnreadCount: s.GetUnreadCount(userId),
	}, nil
}

func (s *messageNotifyService) Get(userId, id int64) (*models.UserMessageNotifyRecord, error) {
	if userId <= 0 {
		return nil, errors.New("USER_ID_REQUIRED")
	}
	record := repositories.UserMessageNotifyRecordRepository.Get(sqls.DB(), id)
	if record == nil || record.UserId != userId {
		return nil, errors.New("MESSAGE_NOT_FOUND")
	}
	return record, nil
}

func (s *messageNotifyService) MarkRead(userId, id int64) (*MessageNotifyReadResult, error) {
	record, err := s.Get(userId, id)
	if err != nil {
		return nil, err
	}
	if record.Status == MessageNotifyStatusRead {
		return &MessageNotifyReadResult{Updated: false, Record: record}, nil
	}
	now := dates.NowTimestamp()
	err = repositories.UserMessageNotifyRecordRepository.Updates(sqls.DB(), record.Id, map[string]any{
		"status":      MessageNotifyStatusRead,
		"update_time": now,
	})
	if err != nil {
		return nil, err
	}
	record.Status = MessageNotifyStatusRead
	record.UpdateTime = now
	return &MessageNotifyReadResult{Updated: true, Record: record}, nil
}

func (s *messageNotifyService) UnreadCount(userId int64) (*MessageNotifyUnreadCountResult, error) {
	if userId <= 0 {
		return nil, errors.New("USER_ID_REQUIRED")
	}
	result := &MessageNotifyUnreadCountResult{
		TotalUnread:    s.GetUnreadCount(userId),
		BusinessUnread: map[string]int64{},
	}
	type row struct {
		BusinessCode string
		Count        int64
	}
	var rows []row
	if err := sqls.DB().Model(&models.UserMessageNotifyRecord{}).
		Select("business_code, count(*) as count").
		Where("user_id = ? and status = ?", userId, MessageNotifyStatusUnread).
		Group("business_code").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		result.BusinessUnread[item.BusinessCode] = item.Count
	}
	return result, nil
}

func (s *messageNotifyService) GetUnreadCount(userId int64) int64 {
	if userId <= 0 {
		return 0
	}
	return repositories.UserMessageNotifyRecordRepository.Count(sqls.DB(), sqls.NewCnd().
		Eq("user_id", userId).
		Eq("status", MessageNotifyStatusUnread))
}

func (s *messageNotifyService) FindTemplates(businessCode string, status *int, page, limit int) ([]models.MessageNotifyTemplate, *sqls.Paging) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	cnd := sqls.NewCnd().Page(page, limit).Desc("id")
	if strings.TrimSpace(businessCode) != "" {
		cnd.Eq("business_code", strings.TrimSpace(businessCode))
	}
	if status != nil {
		cnd.Eq("status", *status)
	}
	list := repositories.MessageNotifyTemplateRepository.Find(sqls.DB(), cnd)
	count := cnd.Count(sqls.DB(), &models.MessageNotifyTemplate{})
	return list, &sqls.Paging{Page: page, Limit: limit, Total: count}
}

func parseTemplateParams(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var params []string
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil, err
	}
	sort.Strings(params)
	return params, nil
}

func renderMessageNotifyTemplate(template string, params map[string]string) (string, error) {
	rendered := template
	for key, val := range params {
		rendered = strings.ReplaceAll(rendered, "{"+key+"}", val)
	}
	renderStrict := true
	if config.Instance != nil && config.Instance.MessageNotify.RenderStrict != nil {
		renderStrict = *config.Instance.MessageNotify.RenderStrict
	}
	if renderStrict {
		if strings.Contains(rendered, "{") || strings.Contains(rendered, "}") {
			return "", errors.New("TEMPLATE_RENDER_FAILED")
		}
	}
	return strings.TrimSpace(rendered), nil
}

func normalizeMessageNotifyLimit(limit int) int {
	defaultLimit := 20
	maxLimit := 100
	if config.Instance != nil {
		if config.Instance.MessageNotify.DefaultPageSize > 0 {
			defaultLimit = config.Instance.MessageNotify.DefaultPageSize
		}
		if config.Instance.MessageNotify.MaxPageSize > 0 {
			maxLimit = config.Instance.MessageNotify.MaxPageSize
		}
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit
}

func toJsonString(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (s *messageNotifyService) pushWithTx(tx *gorm.DB, input MessageNotifyPushInput) (*MessageNotifyPushResult, error) {
	_ = tx
	return s.PushByTemplate(input)
}
