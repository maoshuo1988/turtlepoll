package render

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/models"
	"bbs-go/internal/models/resp"
	"bbs-go/internal/pkg/common"
	html2 "bbs-go/internal/pkg/html"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/pkg/markdown"
	"bbs-go/internal/pkg/text"
	"bbs-go/internal/services"
	"html"
	"strconv"

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/arrays"
	"github.com/mlogclub/simple/common/strs"
)

func BuildTopic(ctx iris.Context, topic *models.Topic) *resp.TopicResponse {
	resp := _buildTopic(topic, true)
	if resp == nil {
		return nil
	}

	entityType := topicEntityType(topic.Type)
	resp.FavoriteCount = services.FavoriteService.CountByEntityIds(entityType, []int64{topic.Id})[topic.Id]
	resp.DislikeCount = services.UserDislikeService.CountDislike(entityType, topic.Id)

	if currentUser := common.GetCurrentUser(ctx); currentUser != nil {
		resp.Favorited = services.FavoriteService.IsFavorited(currentUser.Id, entityType, topic.Id)
		resp.DisLiked = len(services.UserDislikeService.IsDisliked(currentUser.Id, entityType, []int64{topic.Id})) > 0
		resp.Liked = services.UserLikeService.Exists(currentUser.Id, entityType, topic.Id)
	}

	if vote := services.VoteService.Get(topic.VoteId); vote != nil {
		resp.Vote = BuildVote(ctx, vote)
	}

	return resp
}

func BuildSimpleTopic(topic *models.Topic) *resp.TopicResponse {
	buildContent := topic.Type == constants.TopicTypeTweet // 动态时渲染内容
	return _buildTopic(topic, buildContent)
}

func BuildSimpleTopics(ctx iris.Context, topics []models.Topic) []resp.TopicResponse {
	if len(topics) == 0 {
		return nil
	}

	var (
		topicIds          []int64
		topicIdsByType    = map[string][]int64{}
		favoriteCountMap  = map[int64]int64{}
		dislikeCountMap   = map[int64]int64{}
		likedTopicIds     []int64
		favoritedTopicIds []int64
		dislikedTopicIds  []int64
		currentUser       = common.GetCurrentUser(ctx)
	)
	for _, topic := range topics {
		topicIds = append(topicIds, topic.Id)
		entityType := topicEntityType(topic.Type)
		topicIdsByType[entityType] = append(topicIdsByType[entityType], topic.Id)
	}

	for entityType, ids := range topicIdsByType {
		for topicId, count := range services.FavoriteService.CountByEntityIds(entityType, ids) {
			favoriteCountMap[topicId] += count
		}
		for topicId, count := range services.UserDislikeService.CountDislikeByEntityIds(entityType, ids) {
			dislikeCountMap[topicId] += count
		}
		if currentUser != nil {
			likedTopicIds = append(likedTopicIds, services.UserLikeService.IsLiked(currentUser.Id, entityType, ids)...)
			favoritedTopicIds = append(favoritedTopicIds, services.FavoriteService.IsFavoritedByEntityIds(currentUser.Id, entityType, ids)...)
			dislikedTopicIds = append(dislikedTopicIds, services.UserDislikeService.IsDisliked(currentUser.Id, entityType, ids)...)
		}
	}

	var responses []resp.TopicResponse
	for _, topic := range topics {
		item := BuildSimpleTopic(&topic)
		item.FavoriteCount = favoriteCountMap[topic.Id]
		item.Liked = arrays.Contains(topic.Id, likedTopicIds)
		item.Favorited = arrays.Contains(topic.Id, favoritedTopicIds)
		item.DisLiked = arrays.Contains(topic.Id, dislikedTopicIds)
		item.DislikeCount = dislikeCountMap[topic.Id]
		if vote := services.VoteService.Get(topic.VoteId); vote != nil {
			item.Vote = BuildVote(ctx, vote)
		}
		responses = append(responses, *item)
	}
	return responses
}

func _buildTopic(topic *models.Topic, buildContent bool) *resp.TopicResponse {
	if topic == nil {
		return nil
	}

	rsp := &resp.TopicResponse{}

	rsp.Id = idcodec.Encode(topic.Id)
	rsp.Type = topic.Type
	rsp.Title = topic.Title
	rsp.User = BuildUserInfoDefaultIfNull(topic.UserId)
	rsp.LastCommentTime = topic.LastCommentTime
	rsp.CreateTime = topic.CreateTime
	rsp.ViewCount = topic.ViewCount
	rsp.CommentCount = topic.CommentCount
	rsp.LikeCount = topic.LikeCount
	rsp.Recommend = topic.Recommend
	rsp.RecommendTime = topic.RecommendTime
	rsp.Sticky = topic.Sticky
	rsp.StickyTime = topic.StickyTime
	rsp.Status = topic.Status
	rsp.IpLocation = topic.IpLocation

	// 构建内容
	if buildContent {
		if topic.Type == constants.TopicTypeTopic {
			contentHtml := topic.Content
			if topic.ContentType == constants.ContentTypeMarkdown {
				contentHtml = markdown.ToHTML(topic.Content)
			}
			rsp.Content = handleHtmlContent(contentHtml)
		} else {
			rsp.Content = html.EscapeString(topic.Content)
		}
	} else {
		if topic.Type == constants.TopicTypeTopic {
			contentHtml := topic.Content
			if topic.ContentType == constants.ContentTypeMarkdown {
				contentHtml = markdown.ToHTML(topic.Content)
			}
			rsp.Summary = html2.GetSummary(contentHtml, 128)
		} else {
			rsp.Summary = text.GetSummary(topic.Content, 128)
		}
	}

	if topic.Type == constants.TopicTypeTweet {
		if strs.IsBlank(topic.Content) {
			rsp.Content = "分享图片"
		} else {
			rsp.Content = html.EscapeString(topic.Content)
		}
		rsp.ImageList = BuildImageList(topic.ImageList)
	}

	if topic.NodeId > 0 {
		node := services.TopicNodeService.Get(topic.NodeId)
		rsp.Node = BuildNode(node)
	}

	tags := services.TopicService.GetTopicTags(topic.Id)
	rsp.Tags = BuildTags(tags)

	return rsp
}

func topicEntityType(topicType constants.TopicType) string {
	return strconv.Itoa(int(topicType))
}
