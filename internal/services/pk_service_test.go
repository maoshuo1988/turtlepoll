package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"encoding/json"
	"testing"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/stretchr/testify/require"
)

func TestPKBetFormUnmarshalJSONAcceptsStringTopicId(t *testing.T) {
	var form PKBetForm
	if err := json.Unmarshal([]byte(`{"topicId":"1","side":"B","requestId":"req-1","amount":100}`), &form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.TopicId != 1 {
		t.Fatalf("expected topicId 1, got %d", form.TopicId)
	}
	if form.Side != "B" {
		t.Fatalf("expected side B, got %q", form.Side)
	}
	if form.RequestId != "req-1" {
		t.Fatalf("expected requestId req-1, got %q", form.RequestId)
	}
	if form.Amount != 100 {
		t.Fatalf("expected amount 100, got %d", form.Amount)
	}
}

func TestPKBetFormUnmarshalJSONAcceptsNumberTopicId(t *testing.T) {
	var form PKBetForm
	if err := json.Unmarshal([]byte(`{"topicId":1,"side":"A","requestId":"req-2","amount":"250"}`), &form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.TopicId != 1 {
		t.Fatalf("expected topicId 1, got %d", form.TopicId)
	}
	if form.Amount != 250 {
		t.Fatalf("expected amount 250, got %d", form.Amount)
	}
}

func TestPKBetFormUnmarshalJSONUsesDefaultAmount(t *testing.T) {
	var form PKBetForm
	if err := json.Unmarshal([]byte(`{"topicId":1,"side":"A","requestId":"req-3"}`), &form); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if form.Amount != PKBetAmountDefault {
		t.Fatalf("expected default amount %d, got %d", PKBetAmountDefault, form.Amount)
	}
}

func TestCountdownSecondsUsesSecondsForMillisecondTimestamps(t *testing.T) {
	round := &models.PKRound{
		LockTime:      1_700_003_000_000,
		EndTime:       1_700_006_000_000,
		NextRoundTime: 1_700_009_000_000,
	}

	// betting phase: now < lockTime
	require.EqualValues(t, 3000, countdownSeconds(round, 1_700_000_000_000))
	// locked phase: lockTime <= now < endTime
	require.EqualValues(t, 1500, countdownSeconds(round, 1_700_004_500_000))
	// cooldown phase: endTime <= now < nextRoundTime
	require.EqualValues(t, 500, countdownSeconds(round, 1_700_008_500_000))
	// after next round
	require.EqualValues(t, 0, countdownSeconds(round, 1_700_009_100_000))
}

func TestPKServiceCommentReplies_Validation(t *testing.T) {
	list, nextCursor, hasMore, err := PKService.CommentReplies(0, 0, 20, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "commentId is required")
	require.Nil(t, list)
	require.EqualValues(t, 0, nextCursor)
	require.False(t, hasMore)
}

func TestPKServiceCommentReplies_CursorAndFields(t *testing.T) {
	db := sqls.DB()
	if db == nil {
		t.Skip("sqls.DB() is nil; skipping integration-style comment replies test")
	}

	now := dates.NowTimestamp()
	topic := &models.PKTopic{
		Slug:            "pk-reply-test-topic",
		Title:           "reply test topic",
		SideAName:       "A",
		SideBName:       "B",
		Status:          "enabled",
		CurrentRoundId:  0,
		CurrentSeasonId: 0,
		CreateTime:      now,
		UpdateTime:      now,
	}
	require.NoError(t, db.Create(topic).Error)

	round := &models.PKRound{
		TopicId:       topic.Id,
		SeasonId:      0,
		RoundNo:       int(now % 100000),
		Phase:         "betting",
		StartTime:     now - 60,
		LockTime:      now + 60,
		EndTime:       now + 120,
		NextRoundTime: now + 180,
		CreateTime:    now,
		UpdateTime:    now,
	}
	require.NoError(t, db.Create(round).Error)

	parent := &models.Comment{
		UserId:      910001,
		EntityType:  constants.EntityPKTopic,
		EntityId:    topic.Id,
		Content:     "parent",
		ContentType: constants.ContentTypeText,
		Status:      constants.StatusOk,
		CreateTime:  now,
	}
	require.NoError(t, db.Create(parent).Error)

	parentMeta := &models.PKCommentMeta{
		CommentId:  parent.Id,
		TopicId:    topic.Id,
		RoundId:    round.Id,
		Side:       "A",
		HeatScore:  1,
		CreateTime: now,
		UpdateTime: now,
	}
	require.NoError(t, db.Create(parentMeta).Error)

	makeReply := func(uid int64, content, side string, heat float64, downvotes int64, status int, entityId, roundId int64) *models.Comment {
		reply := &models.Comment{
			UserId:      uid,
			EntityType:  constants.EntityComment,
			EntityId:    entityId,
			Content:     content,
			ContentType: constants.ContentTypeText,
			Status:      status,
			CreateTime:  now,
		}
		require.NoError(t, db.Create(reply).Error)
		require.NoError(t, db.Create(&models.PKCommentMeta{
			CommentId:     reply.Id,
			TopicId:       topic.Id,
			RoundId:       roundId,
			Side:          side,
			HeatScore:     heat,
			DownvoteCount: downvotes,
			CreateTime:    now,
			UpdateTime:    now,
		}).Error)
		return reply
	}

	r1 := makeReply(910002, "r1", "A", 3.3, 1, constants.StatusOk, parent.Id, round.Id)
	r2 := makeReply(910003, "r2", "B", 4.4, 2, constants.StatusOk, parent.Id, round.Id)
	r3 := makeReply(910004, "r3", "A", 5.5, 0, constants.StatusOk, parent.Id, round.Id)
	_ = makeReply(910005, "wrong-round", "B", 9.9, 9, constants.StatusOk, parent.Id, round.Id+999)
	_ = makeReply(910006, "wrong-status", "A", 2.2, 3, 1, parent.Id, round.Id)
	_ = makeReply(910007, "wrong-parent", "A", 2.1, 4, constants.StatusOk, parent.Id+999999, round.Id)

	viewer := int64(910099)
	require.NoError(t, db.Create(&models.UserLike{
		UserId:     viewer,
		EntityId:   r2.Id,
		EntityType: constants.EntityComment,
		CreateTime: now,
	}).Error)

	t.Cleanup(func() {
		db.Unscoped().Where("user_id = ?", viewer).Where("entity_type = ?", constants.EntityComment).Delete(&models.UserLike{})
		db.Unscoped().Where("comment_id IN ?", []int64{parent.Id, r1.Id, r2.Id, r3.Id}).Delete(&models.PKCommentMeta{})
		db.Unscoped().Where("entity_type = ? AND entity_id = ?", constants.EntityComment, parent.Id).Delete(&models.Comment{})
		db.Unscoped().Delete(&models.Comment{}, parent.Id)
		db.Unscoped().Delete(&models.PKRound{}, round.Id)
		db.Unscoped().Delete(&models.PKTopic{}, topic.Id)
	})

	list1, cursor1, hasMore1, err := PKService.CommentReplies(parent.Id, 0, 2, viewer)
	require.NoError(t, err)
	require.Len(t, list1, 2)
	require.True(t, hasMore1)

	comment1, ok := list1[0]["comment"].(models.Comment)
	require.True(t, ok)
	require.Equal(t, r1.Id, comment1.Id)
	require.Equal(t, "A", list1[0]["option"])
	require.Equal(t, "A", list1[0]["side"])
	require.Equal(t, 3.3, list1[0]["heatScore"])
	require.EqualValues(t, 1, list1[0]["downvoteCount"])
	require.Equal(t, false, list1[0]["liked"])

	comment2, ok := list1[1]["comment"].(models.Comment)
	require.True(t, ok)
	require.Equal(t, r2.Id, comment2.Id)
	require.Equal(t, "B", list1[1]["option"])
	require.Equal(t, "B", list1[1]["side"])
	require.Equal(t, 4.4, list1[1]["heatScore"])
	require.EqualValues(t, 2, list1[1]["downvoteCount"])
	require.Equal(t, true, list1[1]["liked"])
	require.Equal(t, r2.Id, cursor1)

	list2, cursor2, hasMore2, err := PKService.CommentReplies(parent.Id, cursor1, 2, viewer)
	require.NoError(t, err)
	require.Len(t, list2, 1)
	require.False(t, hasMore2)
	comment3, ok := list2[0]["comment"].(models.Comment)
	require.True(t, ok)
	require.Equal(t, r3.Id, comment3.Id)
	require.Equal(t, r3.Id, cursor2)
}
