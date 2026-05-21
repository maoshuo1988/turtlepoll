package services

import (
	"bbs-go/internal/models/constants"
	"bbs-go/internal/models/resp"
	"errors"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"

	"bbs-go/internal/models/models"
	"bbs-go/internal/repositories"
)

var UserDislikeService = newUserDislikeService()

func newUserDislikeService() *userDislikeService {
	return &userDislikeService{}
}

type userDislikeService struct {
}

func (s *userDislikeService) Get(id int64) *models.UserDislike {
	return repositories.UserDislikeRepository.Get(sqls.DB(), id)
}

func (s *userDislikeService) Take(where ...interface{}) *models.UserDislike {
	return repositories.UserDislikeRepository.Take(sqls.DB(), where...)
}

func (s *userDislikeService) Find(cnd *sqls.Cnd) []models.UserDislike {
	return repositories.UserDislikeRepository.Find(sqls.DB(), cnd)
}

func (s *userDislikeService) FindOne(cnd *sqls.Cnd) *models.UserDislike {
	return repositories.UserDislikeRepository.FindOne(sqls.DB(), cnd)
}

func (s *userDislikeService) FindPageByParams(params *params.QueryParams) (list []models.UserDislike, paging *sqls.Paging) {
	return repositories.UserDislikeRepository.FindPageByParams(sqls.DB(), params)
}

func (s *userDislikeService) FindPageByCnd(cnd *sqls.Cnd) (list []models.UserDislike, paging *sqls.Paging) {
	return repositories.UserDislikeRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *userDislikeService) FindUserCenterDislikePage(userId int64, page, limit int) (list []resp.UserCenterDislikeResponse, paging *sqls.Paging, err error) {
	return repositories.UserDislikeRepository.FindUserCenterDislikePage(sqls.DB(), userId, page, limit)
}

func (s *userDislikeService) Create(t *models.UserDislike) error {
	return repositories.UserDislikeRepository.Create(sqls.DB(), t)
}

func (s *userDislikeService) Update(t *models.UserDislike) error {
	return repositories.UserDislikeRepository.Update(sqls.DB(), t)
}

func (s *userDislikeService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.UserDislikeRepository.Updates(sqls.DB(), id, columns)
}

func (s *userDislikeService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.UserDislikeRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *userDislikeService) Delete(id int64) {
	repositories.UserDislikeRepository.Delete(sqls.DB(), id)
}

func (s *userDislikeService) Exists(userId int64, entityType string, entityId int64) bool {
	return repositories.UserDislikeRepository.Exists(sqls.DB(), userId, entityType, entityId)
}

// CountDislikeByEntityIds 统计点踩数量（仅统计 status=1）
func (s *userDislikeService) CountDislikeByEntityIds(entityType string, entityIds []int64) map[int64]int64 {
	if len(entityIds) == 0 {
		return map[int64]int64{}
	}

	type row struct {
		EntityId int64 `gorm:"column:entity_id"`
		Cnt      int64 `gorm:"column:cnt"`
	}
	var rows []row
	_ = sqls.DB().Model(&models.UserDislike{}).
		Select("entity_id, COUNT(*) AS cnt").
		Where("entity_type = ?", entityType).
		Where("status = ?", 1).
		Where("entity_id IN ?", entityIds).
		Group("entity_id").
		Scan(&rows).Error

	ret := make(map[int64]int64, len(rows))
	for _, r := range rows {
		ret[r.EntityId] = r.Cnt
	}
	return ret
}

func (s *userDislikeService) CountDislike(entityType string, entityId int64) int64 {
	m := s.CountDislikeByEntityIds(entityType, []int64{entityId})
	return m[entityId]
}

// TopicDislike 话题点踩（当前版本仅支持 topic）
func (s *userDislikeService) TopicDislike(userId int64, topicId int64) error {
	topic := repositories.TopicRepository.Get(sqls.DB(), topicId)
	if topic == nil || topic.Status != constants.StatusOk {
		return errors.New("帖子不存在")
	}
	if topic.UserId == userId {
		return errors.New("cannot dislike your own topic")
	}

	now := dates.NowTimestamp()
	exists := repositories.UserDislikeRepository.Take(sqls.DB(),
		"user_id = ? and entity_id = ? and entity_type = ?",
		userId, topicId, constants.EntityTopic,
	)
	if exists != nil {
		return repositories.UserDislikeRepository.Updates(sqls.DB(), exists.Id, map[string]any{
			"status":      1,
			"create_time": now,
		})
	}
	return repositories.UserDislikeRepository.Create(sqls.DB(), &models.UserDislike{
		UserId:     userId,
		EntityType: constants.EntityTopic,
		EntityId:   topicId,
		Status:     1,
		CreateTime: now,
	})
}

// TopicCancelDislike 取消话题点踩
func (s *userDislikeService) TopicCancelDislike(userId int64, topicId int64) error {
	topic := repositories.TopicRepository.Get(sqls.DB(), topicId)
	if topic == nil || topic.Status != constants.StatusOk {
		return errors.New("帖子不存在")
	}
	exists := repositories.UserDislikeRepository.Take(sqls.DB(),
		"user_id = ? and entity_id = ? and entity_type = ?",
		userId, topicId, constants.EntityTopic,
	)
	if exists == nil {
		return nil
	}
	return repositories.UserDislikeRepository.Updates(sqls.DB(), exists.Id, map[string]any{
		"status": 0,
	})
}
