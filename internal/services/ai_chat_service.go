package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/deepseek"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

const (
	aiRoleUser      = "user"
	aiRoleAssistant = "assistant"
	aiSceneChat     = "chat"
)

var AIChatService = newAIChatService()

func newAIChatService() *aiChatService {
	return &aiChatService{}
}

type aiChatService struct{}

type AIChatForm struct {
	Content     string `json:"content" form:"content"`
	Scene       string `json:"scene" form:"scene"`
	ContextType string `json:"contextType" form:"contextType"`
	ContextId   int64  `json:"contextId" form:"contextId"`
}

type AIChatResult struct {
	Message            *models.AIMessage `json:"message"`
	UserMessage        *models.AIMessage `json:"userMessage"`
	BalanceAfter       int64             `json:"balanceAfter,omitempty"`
	StaminaLeft        int               `json:"staminaLeft"`
	MaxStamina         int               `json:"maxStamina"`
	NextRecoverAt      int64             `json:"nextRecoverAt"`
	StaminaCost        int               `json:"staminaCost"`
	PromptTokens       int               `json:"promptTokens"`
	CompletionTokens   int               `json:"completionTokens"`
	TotalTokens        int               `json:"totalTokens"`
	Degraded           bool              `json:"degraded"`
	DailyRemaining     int               `json:"dailyRemaining"`
	DailyMessageLimit  int               `json:"dailyMessageLimit"`
	InsufficientPrompt string            `json:"insufficientPrompt,omitempty"`
}

func (s *aiChatService) Chat(ctx context.Context, userId int64, form AIChatForm) (*AIChatResult, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	conf := config.Instance
	if conf == nil || !conf.AIChat.Enabled || !conf.DeepSeek.Enabled {
		return nil, errors.New("ai chat is disabled")
	}

	content := strings.TrimSpace(form.Content)
	if content == "" {
		return nil, errors.New("content is required")
	}
	maxInputChars := conf.AIChat.MaxInputChars
	if maxInputChars <= 0 {
		maxInputChars = 500
	}
	if len([]rune(content)) > maxInputChars {
		return nil, fmt.Errorf("content exceeds max length %d", maxInputChars)
	}
	scene := strings.TrimSpace(form.Scene)
	if scene == "" {
		scene = aiSceneChat
	}
	staminaCost := conf.AIChat.DefaultStaminaCost
	if staminaCost <= 0 {
		staminaCost = 1
	}

	staminaStatus, enough, err := AIStaminaService.HasEnough(userId, staminaCost)
	if err != nil {
		return nil, err
	}
	limit := staminaStatus.DailyLimit
	if limit > 0 && staminaStatus.DailyUsedCount >= limit {
		return nil, errors.New("daily ai chat limit reached")
	}
	if !enough {
		return &AIChatResult{
			StaminaLeft:        staminaStatus.Stamina,
			MaxStamina:         staminaStatus.MaxStamina,
			NextRecoverAt:      staminaStatus.NextRecoverAt,
			StaminaCost:        staminaCost,
			DailyRemaining:     staminaStatus.DailyRemaining,
			DailyMessageLimit:  limit,
			InsufficientPrompt: "小龟睡着啦~ 喂它一颗苹果(5 龟币)就能继续聊咯",
		}, errors.New("AI_STAMINA_NOT_ENOUGH")
	}

	history, err := s.recentHistory(userId, scene, conf.AIChat.MaxHistoryMessages)
	if err != nil {
		return nil, err
	}
	messages := buildChatMessages(history, content)
	client := deepseek.New(conf.DeepSeek.BaseURL, conf.DeepSeek.APIKey, time.Duration(conf.DeepSeek.TimeoutSeconds)*time.Second, conf.DeepSeek.MaxRetries)
	modelName := conf.DeepSeek.DefaultModel
	if strings.TrimSpace(modelName) == "" {
		modelName = "deepseek-v4-flash"
	}

	resp, err := client.Chat(ctx, deepseek.ChatRequest{
		Model:    modelName,
		Messages: messages,
		Thinking: map[string]any{"type": "disabled"},
	})
	if err != nil {
		return &AIChatResult{
			StaminaLeft:       staminaStatus.Stamina,
			MaxStamina:        staminaStatus.MaxStamina,
			NextRecoverAt:     staminaStatus.NextRecoverAt,
			StaminaCost:       staminaCost,
			Degraded:          true,
			DailyRemaining:    staminaStatus.DailyRemaining,
			DailyMessageLimit: limit,
			Message: &models.AIMessage{
				UserId:      userId,
				Role:        aiRoleAssistant,
				Scene:       scene,
				Content:     "小龟现在有点困，等会儿再来找我聊吧。",
				ModelName:   modelName,
				StaminaCost: 0,
				ContextType: form.ContextType,
				ContextId:   form.ContextId,
				CreateTime:  dates.NowTimestamp(),
			},
		}, nil
	}

	answer := strings.TrimSpace(resp.Choices[0].Message.Content)
	now := dates.NowTimestamp()
	var result AIChatResult
	err = sqls.DB().Transaction(func(tx *gorm.DB) error {
		userMsg := &models.AIMessage{
			UserId:      userId,
			Role:        aiRoleUser,
			Scene:       scene,
			Content:     content,
			ModelName:   modelName,
			StaminaCost: staminaCost,
			ContextType: form.ContextType,
			ContextId:   form.ContextId,
			CreateTime:  now,
		}
		if err := tx.Create(userMsg).Error; err != nil {
			return err
		}
		assistantMsg := &models.AIMessage{
			UserId:           userId,
			Role:             aiRoleAssistant,
			Scene:            scene,
			Content:          answer,
			ModelName:        modelName,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			ContextType:      form.ContextType,
			ContextId:        form.ContextId,
			RequestId:        resp.ID,
			CreateTime:       now,
		}
		if err := tx.Create(assistantMsg).Error; err != nil {
			return err
		}
		afterStamina, err := AIStaminaService.ConsumeForChatTx(tx, userId, assistantMsg.Id, staminaCost)
		if err != nil {
			return err
		}
		result = AIChatResult{
			Message:           assistantMsg,
			UserMessage:       userMsg,
			StaminaLeft:       afterStamina.Stamina,
			MaxStamina:        afterStamina.MaxStamina,
			NextRecoverAt:     afterStamina.NextRecoverAt,
			StaminaCost:       staminaCost,
			PromptTokens:      resp.Usage.PromptTokens,
			CompletionTokens:  resp.Usage.CompletionTokens,
			TotalTokens:       resp.Usage.TotalTokens,
			DailyRemaining:    afterStamina.DailyRemaining,
			DailyMessageLimit: limit,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	_ = AIPushService.UpdateAIInteract(userId)
	return &result, nil
}

func (s *aiChatService) recentHistory(userId int64, scene string, maxMessages int) ([]models.AIMessage, error) {
	if maxMessages <= 0 {
		maxMessages = 8
	}
	var desc []models.AIMessage
	if err := sqls.DB().Where("user_id = ? and scene = ?", userId, scene).
		Order("create_time desc, id desc").
		Limit(maxMessages).
		Find(&desc).Error; err != nil {
		return nil, err
	}
	history := make([]models.AIMessage, len(desc))
	for i := range desc {
		history[len(desc)-1-i] = desc[i]
	}
	return history, nil
}

func buildChatMessages(history []models.AIMessage, content string) []deepseek.Message {
	messages := []deepseek.Message{
		{
			Role:    "system",
			Content: "你是龟投论坛里的小龟 AI 伙伴。回答要简洁、温和、有一点俏皮；不要自称 DeepSeek 或透露底层模型；不要承诺稳赚、梭哈或确定性投资收益；不知道历史事实时直接说还没记住。",
		},
	}
	for _, item := range history {
		role := item.Role
		if role != aiRoleUser && role != aiRoleAssistant {
			continue
		}
		messages = append(messages, deepseek.Message{Role: role, Content: item.Content})
	}
	messages = append(messages, deepseek.Message{Role: aiRoleUser, Content: content})
	return messages
}
