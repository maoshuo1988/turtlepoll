package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"
	"bbs-go/internal/pkg/event"
	"errors"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"

	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
)

var FavoriteService = newFavoriteService()

func newFavoriteService() *favoriteService {
	return &favoriteService{}
}

type favoriteService struct {
}

func (s *favoriteService) Get(id int64) *models.Favorite {
	return repositories.FavoriteRepository.Get(sqls.DB(), id)
}

func (s *favoriteService) Take(where ...interface{}) *models.Favorite {
	return repositories.FavoriteRepository.Take(sqls.DB(), where...)
}

func (s *favoriteService) Find(cnd *sqls.Cnd) []models.Favorite {
	return repositories.FavoriteRepository.Find(sqls.DB(), cnd)
}

func (s *favoriteService) FindOne(cnd *sqls.Cnd) *models.Favorite {
	return repositories.FavoriteRepository.FindOne(sqls.DB(), cnd)
}

func (s *favoriteService) FindPageByParams(params *params.QueryParams) (list []models.Favorite, paging *sqls.Paging) {
	return repositories.FavoriteRepository.FindPageByParams(sqls.DB(), params)
}

func (s *favoriteService) FindPageByCnd(cnd *sqls.Cnd) (list []models.Favorite, paging *sqls.Paging) {
	return repositories.FavoriteRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *favoriteService) FindUserCenterFavoritePage(userId int64, page, limit int) (list []resp.UserCenterFavoriteResponse, paging *sqls.Paging, err error) {
	return repositories.FavoriteRepository.FindUserCenterFavoritePage(sqls.DB(), userId, page, limit)
}

func (s *favoriteService) Create(t *models.Favorite) error {
	return repositories.FavoriteRepository.Create(sqls.DB(), t)
}

func (s *favoriteService) Update(t *models.Favorite) error {
	return repositories.FavoriteRepository.Update(sqls.DB(), t)
}

func (s *favoriteService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.FavoriteRepository.Updates(sqls.DB(), id, columns)
}

func (s *favoriteService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.FavoriteRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *favoriteService) Delete(id int64) {
	repositories.FavoriteRepository.Delete(sqls.DB(), id)
}

func (s *favoriteService) IsFavorited(userId int64, entityType string, entityId int64) bool {
	return repositories.FavoriteRepository.Take(sqls.DB(), "user_id = ? and entity_type = ? and entity_id = ?",
		userId, entityType, entityId) != nil
}

func (s *favoriteService) IsFavoritedByEntityIds(userId int64, entityType string, entityIds []int64) (favoritedEntityIds []int64) {
	if len(entityIds) == 0 {
		return nil
	}
	list := repositories.FavoriteRepository.Find(sqls.DB(), sqls.NewCnd().
		Eq("user_id", userId).
		In("entity_id", entityIds).
		Eq("entity_type", entityType))
	for _, favorite := range list {
		favoritedEntityIds = append(favoritedEntityIds, favorite.EntityId)
	}
	return
}

func (s *favoriteService) CountByEntityIds(entityType string, entityIds []int64) map[int64]int64 {
	if len(entityIds) == 0 {
		return map[int64]int64{}
	}

	type row struct {
		EntityId int64 `gorm:"column:entity_id"`
		Cnt      int64 `gorm:"column:cnt"`
	}
	var rows []row
	_ = sqls.DB().Model(&models.Favorite{}).
		Select("entity_id, COUNT(*) AS cnt").
		Where("entity_type = ?", entityType).
		Where("entity_id IN ?", entityIds).
		Group("entity_id").
		Scan(&rows).Error

	ret := make(map[int64]int64, len(rows))
	for _, row := range rows {
		ret[row.EntityId] = row.Cnt
	}
	return ret
}

func (s *favoriteService) GetBy(userId int64, entityType string, entityId int64) *models.Favorite {
	return repositories.FavoriteRepository.Take(sqls.DB(), "user_id = ? and entity_type = ? and entity_id = ?",
		userId, entityType, entityId)
}

// AddArticleFavorite 收藏文章
func (s *favoriteService) AddArticleFavorite(userId, articleId int64) error {
	article := repositories.ArticleRepository.Get(sqls.DB(), articleId)
	if article == nil || article.Status != constants.StatusOk {
		return errors.New("收藏的文章不存在")
	}
	return s.addFavorite(userId, constants.EntityArticle, articleId)
}

// AddTopicFavorite 收藏主题
func (s *favoriteService) AddTopicFavorite(userId, topicId int64) error {
	topic := repositories.TopicRepository.Get(sqls.DB(), topicId)
	if topic == nil || topic.Status != constants.StatusOk {
		return errors.New("收藏的话题不存在")
	}
	return s.addFavorite(userId, constants.EntityTopic, topicId)
}

func (s *favoriteService) addFavorite(userId int64, entityType string, entityId int64) error {
	if s.IsFavorited(userId, entityType, entityId) { // 已经收藏
		return nil
	}
	if err := repositories.FavoriteRepository.Create(sqls.DB(), &models.Favorite{
		UserId:     userId,
		EntityType: entityType,
		EntityId:   entityId,
		CreateTime: dates.NowTimestamp(),
	}); err != nil {
		return err
	}

	// 发送事件
	event.Send(event.UserFavoriteEvent{
		UserId:     userId,
		EntityId:   entityId,
		EntityType: entityType,
	})
	return nil
}
