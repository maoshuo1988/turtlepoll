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

	"github.com/kataras/iris/v12"
	"github.com/mlogclub/simple/common/arrays"
	"github.com/mlogclub/simple/common/strs"
)

func BuildTopic(ctx iris.Context, topic *models.Topic) *resp.TopicResponse {
	resp := _buildTopic(topic, true)
	if resp == nil {
		return nil
	}

	resp.FavoriteCount = services.FavoriteService.CountByEntityIds(constants.EntityTopic, []int64{topic.Id})[topic.Id]
	resp.DislikeCount = services.UserDislikeService.CountDislike(constants.EntityTopic, topic.Id)

	if currentUser := common.GetCurrentUser(ctx); currentUser != nil {
		resp.Favorited = services.FavoriteService.IsFavorited(currentUser.Id, constants.EntityTopic, topic.Id)
		resp.DisLiked = len(services.UserDislikeService.IsDisliked(currentUser.Id, constants.EntityTopic, []int64{topic.Id})) > 0
		resp.Liked = services.UserLikeService.Exists(currentUser.Id, constants.EntityTopic, topic.Id)
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

	var topicIds []int64
	for _, topic := range topics {
		topicIds = append(topicIds, topic.Id)
	}

	favoriteCountMap := services.FavoriteService.CountByEntityIds(constants.EntityTopic, topicIds)
	var likedTopicIds []int64
	var favoritedTopicIds []int64
	var dislikedTopicIds []int64
	if currentUser := common.GetCurrentUser(ctx); currentUser != nil {
		likedTopicIds = services.UserLikeService.IsLiked(currentUser.Id, constants.EntityTopic, topicIds)
		favoritedTopicIds = services.FavoriteService.IsFavoritedByEntityIds(currentUser.Id, constants.EntityTopic, topicIds)
		dislikedTopicIds = services.UserDislikeService.IsDisliked(currentUser.Id, constants.EntityTopic, topicIds)
	}

	dislikeCountMap := services.UserDislikeService.CountDislikeByEntityIds(constants.EntityTopic, topicIds)

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
