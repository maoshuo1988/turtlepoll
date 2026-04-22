package services

import (
	"bbs-go/internal/cache"
	"bbs-go/internal/models/models"
	"bbs-go/internal/pkg/biztime"
	"bbs-go/internal/pkg/event"
	"bbs-go/internal/repositories"
	"errors"
	"sync"
	"time"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web/params"
)

var CheckInService = newCheckInService()

func newCheckInService() *checkInService {
	return &checkInService{}
}

type checkInService struct {
	m sync.Mutex
}

func (s *checkInService) Get(id int64) *models.CheckIn {
	return repositories.CheckInRepository.Get(sqls.DB(), id)
}

func (s *checkInService) Take(where ...interface{}) *models.CheckIn {
	return repositories.CheckInRepository.Take(sqls.DB(), where...)
}

func (s *checkInService) Find(cnd *sqls.Cnd) []models.CheckIn {
	return repositories.CheckInRepository.Find(sqls.DB(), cnd)
}

func (s *checkInService) FindOne(cnd *sqls.Cnd) *models.CheckIn {
	return repositories.CheckInRepository.FindOne(sqls.DB(), cnd)
}

func (s *checkInService) FindPageByParams(params *params.QueryParams) (list []models.CheckIn, paging *sqls.Paging) {
	return repositories.CheckInRepository.FindPageByParams(sqls.DB(), params)
}

func (s *checkInService) FindPageByCnd(cnd *sqls.Cnd) (list []models.CheckIn, paging *sqls.Paging) {
	return repositories.CheckInRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *checkInService) Count(cnd *sqls.Cnd) int64 {
	return repositories.CheckInRepository.Count(sqls.DB(), cnd)
}

func (s *checkInService) Create(t *models.CheckIn) error {
	return repositories.CheckInRepository.Create(sqls.DB(), t)
}

func (s *checkInService) Update(t *models.CheckIn) error {
	return repositories.CheckInRepository.Update(sqls.DB(), t)
}

func (s *checkInService) Updates(id int64, columns map[string]interface{}) error {
	return repositories.CheckInRepository.Updates(sqls.DB(), id, columns)
}

func (s *checkInService) UpdateColumn(id int64, name string, value interface{}) error {
	return repositories.CheckInRepository.UpdateColumn(sqls.DB(), id, name, value)
}

func (s *checkInService) Delete(id int64) {
	repositories.CheckInRepository.Delete(sqls.DB(), id)
}

func (s *checkInService) CheckIn(userId int64) error {
	s.m.Lock()
	defer s.m.Unlock()
	var (
		checkIn         = s.GetByUserId(userId)
		dayName         = dates.GetDay(time.Now())
		yesterdayName   = dates.GetDay(time.Now().Add(-time.Hour * 24))
		consecutiveDays = 1
		err             error
	)

	if checkIn != nil && checkIn.LatestDayName == dayName {
		return errors.New("你已签到")
	}

	if checkIn != nil && checkIn.LatestDayName == yesterdayName {
		consecutiveDays = checkIn.ConsecutiveDays + 1
	}

	if checkIn == nil {
		err = s.Create(&models.CheckIn{
			Model:           models.Model{},
			UserId:          userId,
			LatestDayName:   dayName,
			ConsecutiveDays: consecutiveDays,
			CreateTime:      dates.NowTimestamp(),
			UpdateTime:      dates.NowTimestamp(),
		})
	} else {
		checkIn.LatestDayName = dayName
		checkIn.ConsecutiveDays = consecutiveDays
		checkIn.UpdateTime = dates.NowTimestamp()
		err = s.Update(checkIn)
	}
	if err == nil {
		// 清理签到排行榜缓存
		cache.UserCache.RefreshCheckInRank()
		// 发送事件（用于任务系统等异步处理）
		event.Send(event.CheckInEvent{
			UserId:  userId,
			DayName: dayName,
		})

		// 兼容保留旧签到入口上的每日登录加成发放逻辑。
		// spark_multiplier 已收口到登录结算主链路，这里只保留 signin_bonus 的历史行为，避免回归。
		func() {
			defer func() { _ = recover() }()
			petId := int64(consecutiveDays)
			_ = PetSigninBonusService.GrantByCheckIn(userId, s.GetByUserId(userId), petId)
		}()
	}
	return err
}

// EnsureLoginStreak 在登录结算时同步签到日切口径。
//
// 约束：
// - 同一北京时间自然日重复调用不会重复累加 streak。
// - 只维护签到状态与连续登录天数，不发放任何奖励。
func (s *checkInService) EnsureLoginStreak(userId int64) (*models.CheckIn, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if userId <= 0 {
		return nil, errors.New("userId is required")
	}

	now := biztime.NowInCST()
	dayName := biztime.DayNameCST(now)
	yesterdayName := biztime.DayNameCST(now.Add(-24 * time.Hour))
	checkIn := s.GetByUserId(userId)

	if checkIn != nil && checkIn.LatestDayName == dayName {
		return checkIn, nil
	}

	consecutiveDays := 1
	if checkIn != nil && checkIn.LatestDayName == yesterdayName {
		consecutiveDays = checkIn.ConsecutiveDays + 1
	}

	nowTs := dates.NowTimestamp()
	if checkIn == nil {
		checkIn = &models.CheckIn{
			Model:           models.Model{},
			UserId:          userId,
			LatestDayName:   dayName,
			ConsecutiveDays: consecutiveDays,
			CreateTime:      nowTs,
			UpdateTime:      nowTs,
		}
		if err := s.Create(checkIn); err != nil {
			return nil, err
		}
	} else {
		checkIn.LatestDayName = dayName
		checkIn.ConsecutiveDays = consecutiveDays
		checkIn.UpdateTime = nowTs
		if err := s.Update(checkIn); err != nil {
			return nil, err
		}
	}

	cache.UserCache.RefreshCheckInRank()
	event.Send(event.CheckInEvent{
		UserId:  userId,
		DayName: dayName,
	})
	return checkIn, nil
}

func (s *checkInService) GetByUserId(userId int64) *models.CheckIn {
	return s.FindOne(sqls.NewCnd().Eq("user_id", userId))
}
