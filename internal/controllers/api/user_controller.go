package api

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"
	"bbs-go/internal/pkg/common"
	"bbs-go/internal/pkg/config"
	"bbs-go/internal/pkg/errs"
	"bbs-go/internal/pkg/idcodec"
	"bbs-go/internal/pkg/locales"
	"bbs-go/internal/pkg/msg"
	"bbs-go/internal/pkg/validate"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/mlogclub/simple/web/params"
	"github.com/spf13/cast"

	"bbs-go/internal/cache"
	"bbs-go/internal/controllers/render"
	"bbs-go/internal/models/models"
	"bbs-go/internal/services"
)

type UserController struct {
	Ctx iris.Context
}

func (c *UserController) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/center/topics", "GetCenterTopics")
	b.Handle("GET", "/center/comments", "GetCenterComments")
	b.Handle("GET", "/center/favorites", "GetCenterFavorites")
	b.Handle("GET", "/center/dislike/list", "GetCenterDislikeList")
	b.Handle("POST", "/topic/hide/list", "PostTopicHideList")
	b.Handle("POST", "/topic/hide/{topicId}", "PostTopicHideBy")
	b.Handle("POST", "/topic/unhide/{topicId}", "PostTopicUnhideBy")
}

// 获取当前登录用户
func (c *UserController) GetCurrent() *web.JsonResult {
	if !config.Instance.Installed {
		return web.JsonSuccess()
	}
	user := common.GetCurrentUser(c.Ctx)
	if user != nil {
		return web.JsonData(render.BuildUserProfile(user))
	}
	return web.JsonSuccess()
}

// 用户详情
func (c *UserController) GetBy(userIdStr string) *web.JsonResult {
	userId := idcodec.Decode(userIdStr)
	user := cache.UserCache.Get(userId)
	if user != nil && user.Status != constants.StatusDeleted {
		return web.JsonData(render.BuildUserDetail(user))
	}
	return web.JsonErrorMsg(locales.Get("user.not_found"))
}

// 修改用户资料
func (c *UserController) PostEditBy(userIdStr string) *web.JsonResult {
	userId := idcodec.Decode(userIdStr)
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if user.Id != userId {
		return web.JsonErrorMsg(locales.Get("user.no_permission"))
	}
	var (
		nickname    = strings.TrimSpace(params.FormValue(c.Ctx, "nickname"))
		homePage    = params.FormValue(c.Ctx, "homePage")
		description = params.FormValue(c.Ctx, "description")
		gender      = strings.TrimSpace(params.FormValue(c.Ctx, "gender"))
		err         error
	)

	if len(nickname) == 0 {
		return web.JsonErrorMsg(locales.Get("user.nickname_empty"))
	}

	if strs.IsNotBlank(gender) {
		if gender != string(constants.GenderMale) && gender != string(constants.GenderFemale) {
			return web.JsonErrorMsg(locales.Get("user.gender_error"))
		}
	}

	if len(homePage) > 0 && validate.IsURL(homePage) != nil {
		return web.JsonErrorMsg(locales.Get("user.homepage_error"))
	}

	columns := map[string]any{
		"nickname":    nickname,
		"home_page":   homePage,
		"description": description,
		"gender":      gender,
	}
	err = services.UserService.Updates(user.Id, columns)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 修改头像
func (c *UserController) PostUpdateAvatar() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	avatar := strings.TrimSpace(params.FormValue(c.Ctx, "avatar"))
	if len(avatar) == 0 {
		return web.JsonErrorMsg(locales.Get("user.avatar_empty"))
	}
	err := services.UserService.UpdateAvatar(user.Id, avatar)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

func (c *UserController) PostUpdateNickname() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	nickname := strings.TrimSpace(params.FormValue(c.Ctx, "nickname"))
	if len(nickname) == 0 {
		return web.JsonErrorMsg(locales.Get("user.nickname_empty"))
	}
	err := services.UserService.UpdateNickname(user.Id, nickname)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

func (c *UserController) PostUpdateDescription() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	description := strings.TrimSpace(params.FormValue(c.Ctx, "description"))
	err := services.UserService.UpdateDescription(user.Id, description)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

func (c *UserController) PostUpdateGender() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	gender := strings.TrimSpace(params.FormValue(c.Ctx, "gender"))
	err := services.UserService.UpdateGender(user.Id, gender)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

func (c *UserController) PostUpdateBirthday() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	birthday := strings.TrimSpace(params.FormValue(c.Ctx, "birthday"))
	err := services.UserService.UpdateBirthday(user.Id, birthday)
	if err != nil {
		return web.JsonErrorMsg(err.Error())
	}
	return web.JsonSuccess()
}

// 设置用户名
func (c *UserController) PostSet_username() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	username := strings.TrimSpace(params.FormValue(c.Ctx, "username"))
	err := services.UserService.SetUsername(user.Id, username)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 设置邮箱
func (c *UserController) PostSet_email() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	email := strings.TrimSpace(params.FormValue(c.Ctx, "email"))
	err := services.UserService.SetEmail(user.Id, email)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 设置密码
func (c *UserController) PostSet_password() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	password := params.FormValue(c.Ctx, "password")
	rePassword := params.FormValue(c.Ctx, "rePassword")
	err := services.UserService.SetPassword(user.Id, password, rePassword)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 修改密码
func (c *UserController) PostUpdate_password() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	var (
		oldPassword = params.FormValue(c.Ctx, "oldPassword")
		password    = params.FormValue(c.Ctx, "password")
		rePassword  = params.FormValue(c.Ctx, "rePassword")
	)
	if err := services.UserService.UpdatePassword(user.Id, oldPassword, password, rePassword); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 设置背景图
func (c *UserController) PostSet_background_image() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	backgroundImage := params.FormValue(c.Ctx, "backgroundImage")
	if strs.IsBlank(backgroundImage) {
		return web.JsonErrorMsg(locales.Get("user.upload_image_required"))
	}
	if err := services.UserService.UpdateBackgroundImage(user.Id, backgroundImage); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 用户收藏
func (c *UserController) GetFavorites() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	cursor := params.FormValueInt64Default(c.Ctx, "cursor", 0)

	// 用户必须登录
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}

	// 查询列表
	limit := 20
	var favorites []models.Favorite
	if cursor > 0 {
		favorites = services.FavoriteService.Find(sqls.NewCnd().Where("user_id = ? and id < ?",
			user.Id, cursor).Desc("id").Limit(20))
	} else {
		favorites = services.FavoriteService.Find(sqls.NewCnd().Where("user_id = ?", user.Id).Desc("id").Limit(limit))
	}

	hasMore := false
	if len(favorites) > 0 {
		cursor = favorites[len(favorites)-1].Id
		hasMore = len(favorites) >= limit
	}

	return web.JsonCursorData(render.BuildFavorites(favorites), strconv.FormatInt(cursor, 10), hasMore)
}

// 获取最近3条未读消息
func (c *UserController) GetMsg_recent() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	var count int64 = 0
	var messages []models.Message
	if user != nil {
		count = services.MessageService.GetUnReadCount(user.Id)
		messages = services.MessageService.Find(sqls.NewCnd().Eq("user_id", user.Id).
			Eq("status", msg.StatusUnread).Limit(3).Desc("id"))
	}
	return web.NewEmptyRspBuilder().Put("count", count).Put("messages", render.BuildMessages(messages)).JsonResult()
}

// 用户消息
func (c *UserController) GetMessages() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(errs.NotLogin())
	}
	var (
		limit     = 20
		cursor, _ = params.GetInt64(c.Ctx, "cursor")
	)

	cnd := sqls.NewCnd().Eq("user_id", user.Id).Limit(limit).Desc("id")
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	list := services.MessageService.Find(cnd)

	var (
		nextCursor = cursor
		hasMore    = false
	)
	if len(list) > 0 {
		nextCursor = list[len(list)-1].Id
		hasMore = len(list) == limit
	}

	// 全部标记为已读
	services.MessageService.MarkRead(user.Id)

	return web.JsonCursorData(render.BuildMessages(list), cast.ToString(nextCursor), hasMore)
}

// 用户积分记录
func (c *UserController) GetScore_logs() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}
	var (
		limit     = 20
		cursor, _ = params.GetInt64(c.Ctx, "cursor")
	)
	cnd := sqls.NewCnd().Eq("user_id", user.Id).Limit(limit).Desc("id")
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	list := services.UserScoreLogService.Find(cnd)

	var (
		nextCursor = cursor
		hasMore    = false
	)
	if len(list) > 0 {
		nextCursor = list[len(list)-1].Id
		hasMore = len(list) == limit
	}

	return web.JsonCursorData(list, cast.ToString(nextCursor), hasMore)
}

// 积分排行
func (c *UserController) GetScoreRank() *web.JsonResult {
	users := cache.UserCache.GetScoreRank()
	var results []*resp.UserInfo
	for _, user := range users {
		results = append(results, render.BuildUserInfo(&user))
	}
	return web.JsonData(results)
}

func (c *UserController) GetCenterTopics() *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	userId := currentUser.Id

	page := params.FormValueIntDefault(c.Ctx, "page", 1)
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	cnd := sqls.NewCnd().
		Eq("user_id", userId).
		Eq("status", constants.StatusOk).
		Page(page, limit).
		Desc("create_time").
		Desc("id")

	topics, paging := services.TopicService.FindPageByCnd(cnd)
	return web.JsonPageData(buildUserCenterTopics(topics), paging)
}

func (c *UserController) PostTopicHideBy(topicIdStr string) *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	topicId, err := strconv.ParseInt(topicIdStr, 10, 64)
	if err != nil || topicId <= 0 {
		return web.JsonErrorMsg(locales.Get("common.not_found"))
	}

	topic := services.TopicService.Get(topicId)
	if topic == nil || topic.Status == constants.StatusDeleted {
		return web.JsonErrorMsg(locales.Get("common.not_found"))
	}
	if topic.UserId != currentUser.Id {
		return web.JsonErrorMsg(locales.Get("topic.no_permission"))
	}

	if err := services.TopicService.UpdateDisplayStatus(topicId, 1); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// 禁言
func (c *UserController) PostForbidden() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if !user.HasAnyRole(constants.RoleOwner, constants.RoleAdmin) {
		return web.JsonErrorMsg(locales.Get("user.no_permission"))
	}
	var (
		userId = common.GetID(c.Ctx, "userId")
		days   = params.FormValueIntDefault(c.Ctx, "days", 0)
		reason = params.FormValue(c.Ctx, "reason")
	)
	if userId < 0 {
		return web.JsonErrorMsg("param: userId required")
	}
	if days == -1 && !user.HasRole(constants.RoleOwner) {
		return web.JsonErrorMsg(locales.Get("user.no_permission"))
	}
	if days == 0 {
		services.UserService.RemoveForbidden(user.Id, userId, c.Ctx.Request())
	} else {
		if err := services.UserService.Forbidden(user.Id, userId, days, reason, c.Ctx.Request()); err != nil {
			return web.JsonError(err)
		}
	}
	return web.JsonSuccess()
}

// PostEmailVerify 请求邮箱验证邮件
func (c *UserController) PostSend_verify_email() *web.JsonResult {
	user := common.GetCurrentUser(c.Ctx)
	if user == nil {
		return web.JsonError(errs.NotLogin())
	}
	if err := services.UserService.SendEmailVerifyEmail(user.Id); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

// PostVerify_email 获取邮箱验证码
func (c *UserController) PostVerify_email() *web.JsonResult {
	token := params.FormValue(c.Ctx, "token")
	if strs.IsBlank(token) {
		return web.JsonErrorMsg("Illegal request")
	}
	var (
		email string
		err   error
	)
	if email, err = services.UserService.VerifyEmail(token); err != nil {
		return web.JsonError(err)
	}
	return web.NewEmptyRspBuilder().Put("email", email).JsonResult()
}

func (c *UserController) GetWx_bind_info() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}
	thirdUser := services.ThirdUserService.GetByUserId(user.Id, constants.ThirdTypeWeixin)
	if thirdUser != nil {
		return web.JsonData(map[string]any{
			"bind":     true,
			"nickname": thirdUser.Nickname,
			"avatar":   thirdUser.Avatar,
		})
	}
	return web.JsonData(map[string]any{
		"bind": false,
	})
}

func (c *UserController) GetGoogle_bind_info() *web.JsonResult {
	user, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}
	thirdUser := services.ThirdUserService.GetByUserId(user.Id, constants.ThirdTypeGoogle)
	if thirdUser != nil {
		return web.JsonData(map[string]any{
			"bind":     true,
			"nickname": thirdUser.Nickname,
			"avatar":   thirdUser.Avatar,
		})
	}
	return web.JsonData(map[string]any{
		"bind": false,
	})
}

func (c *UserController) GetCenterComments() *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	userId := currentUser.Id

	page := params.FormValueIntDefault(c.Ctx, "page", 1)
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	results, paging, err := services.CommentService.FindUserCenterCommentPage(userId, page, limit)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonPageData(results, paging)
}

func (c *UserController) PostTopicUnhideBy(topicIdStr string) *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	topicId, err := strconv.ParseInt(topicIdStr, 10, 64)
	if err != nil || topicId <= 0 {
		return web.JsonErrorMsg(locales.Get("common.not_found"))
	}

	topic := services.TopicService.Get(topicId)
	if topic == nil || topic.Status == constants.StatusDeleted {
		return web.JsonErrorMsg(locales.Get("common.not_found"))
	}
	if topic.UserId != currentUser.Id {
		return web.JsonErrorMsg(locales.Get("topic.no_permission"))
	}

	if err := services.TopicService.UpdateDisplayStatus(topicId, 0); err != nil {
		return web.JsonError(err)
	}
	return web.JsonSuccess()
}

func (c *UserController) GetCenterFavorites() *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	userId := currentUser.Id

	page := params.FormValueIntDefault(c.Ctx, "page", 1)
	limit := params.FormValueIntDefault(c.Ctx, "limit", 20)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	results, paging, err := services.FavoriteService.FindUserCenterFavoritePage(userId, page, limit)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonPageData(results, paging)
}

func (c *UserController) GetCenterDislikeList() *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	page, limit := getUserCenterPageLimit(c.Ctx)
	results, paging, err := services.UserDislikeService.FindUserCenterDislikePage(currentUser.Id, page, limit)
	if err != nil {
		return web.JsonError(err)
	}
	return web.JsonPageData(results, paging)
}

func (c *UserController) PostTopicHideList() *web.JsonResult {
	currentUser, err := common.CheckLogin(c.Ctx)
	if err != nil {
		return web.JsonError(err)
	}

	page, limit := getUserCenterPageLimit(c.Ctx)
	topics, paging := services.TopicService.FindUserHiddenTopicPage(currentUser.Id, page, limit)
	return web.JsonPageData(buildUserCenterHiddenTopics(topics), paging)
}

func getUserCenterPageLimit(ctx iris.Context) (page, limit int) {
	page = params.FormValueIntDefault(ctx, "page", 1)
	limit = params.FormValueIntDefault(ctx, "limit", 20)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	return
}

func buildUserCenterTopics(topics []models.Topic) []resp.UserCenterTopicResponse {
	if len(topics) == 0 {
		return nil
	}

	results := make([]resp.UserCenterTopicResponse, 0, len(topics))
	for _, topic := range topics {
		results = append(results, resp.UserCenterTopicResponse{
			Id:         topic.Id,
			UserId:     topic.UserId,
			Title:      topic.Title,
			Content:    topic.Content,
			CreateTime: topic.CreateTime,
		})
	}
	return results
}

func buildUserCenterHiddenTopics(topics []models.Topic) []resp.UserCenterHiddenTopicResponse {
	if len(topics) == 0 {
		return nil
	}

	results := make([]resp.UserCenterHiddenTopicResponse, 0, len(topics))
	for _, topic := range topics {
		createTime := ""
		if topic.CreateTime > 0 {
			createTime = time.Unix(topic.CreateTime, 0).Format("2006-01-02 15:04:05")
		}
		results = append(results, resp.UserCenterHiddenTopicResponse{
			Id:            topic.Id,
			UserId:        topic.UserId,
			Content:       topic.Content,
			Title:         topic.Title,
			CreateTime:    createTime,
			DisplayStatus: topic.DisplayStatus,
		})
	}
	return results
}
