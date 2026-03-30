package services

import (
	"bbs-go/internal/models"
	"bbs-go/internal/repositories"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/mlogclub/simple/common/dates"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var BattleService = newBattleService()

func newBattleService() *battleService {
	return &battleService{}
}

type battleService struct{}

const (
	BattleStatusOpen     = "open"
	BattleStatusSealed   = "sealed"
	BattleStatusPending  = "pending"
	BattleStatusDisputed = "disputed"
	BattleStatusSettled  = "settled"

	BattleRoleBanker     = "banker"
	BattleRoleChallenger = "challenger"

	BattleResultBankerWins  = "banker_wins"
	BattleResultBankerLoses = "banker_loses"
	BattleResultVoid        = "void"
)

type CreateBattleForm struct {
	Title          string `json:"title"`
	BankerSide     string `json:"bankerSide"`
	ChallengerSide string `json:"challengerSide"`
	StakeAmount    int64  `json:"stakeAmount"`
	IsPublic       bool   `json:"isPublic"`
	InviteCode     string `json:"inviteCode"`
	SettleTime     int64  `json:"settleTime"`
	RequestId      string `json:"requestId"` // 幂等（同一用户重复创建可选用；当前先不做唯一）
}

type JoinBattleForm struct {
	BattleId   int64  `json:"battleId"`
	Amount     int64  `json:"amount"`
	RequestId  string `json:"requestId"`
	InviteCode string `json:"inviteCode"`
}

type BankerAddStakeForm struct {
	BattleId  int64  `json:"battleId"`
	Amount    int64  `json:"amount"`
	RequestId string `json:"requestId"`
}

type WithdrawForm struct {
	BattleId  int64  `json:"battleId"`
	RequestId string `json:"requestId"`
}

type ChallengeActionForm struct {
	BattleId  int64  `json:"battleId"`
	RequestId string `json:"requestId"`
	Remark    string `json:"remark"`
}

type AdminResolveForm struct {
	BattleId  int64  `json:"battleId"`
	RequestId string `json:"requestId"`
	Result    string `json:"result"` // banker_wins/banker_loses/void
	Remark    string `json:"remark"`
}

func (s *battleService) CreateBattle(bankerUserId int64, form CreateBattleForm) (*models.Battle, error) {
	if bankerUserId <= 0 {
		return nil, errors.New("bankerUserId is required")
	}
	form.Title = strings.TrimSpace(form.Title)
	form.BankerSide = strings.TrimSpace(form.BankerSide)
	form.ChallengerSide = strings.TrimSpace(form.ChallengerSide)
	if form.Title == "" {
		return nil, errors.New("title is required")
	}
	if form.BankerSide == "" || form.ChallengerSide == "" {
		return nil, errors.New("sides are required")
	}
	if form.StakeAmount < 100 {
		return nil, errors.New("stakeAmount must be >= 100")
	}
	if form.SettleTime <= 0 {
		return nil, errors.New("settleTime is required")
	}
	if !form.IsPublic {
		form.InviteCode = strings.TrimSpace(form.InviteCode)
		if form.InviteCode == "" {
			return nil, errors.New("inviteCode is required for private battle")
		}
	}

	now := dates.NowTimestamp()
	b := &models.Battle{
		Title:              form.Title,
		BankerUserId:       bankerUserId,
		BankerSide:         form.BankerSide,
		ChallengerSide:     form.ChallengerSide,
		IsPublic:           form.IsPublic,
		InviteCode:         form.InviteCode,
		Status:             BattleStatusOpen,
		SettleTime:         form.SettleTime,
		BankerStakeTotal:   form.StakeAmount,
		PoolPrincipalTotal: form.StakeAmount,
		CreateTime:         now,
		UpdateTime:         now,
	}

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		if err := repositories.BattleRepository.Create(tx, b); err != nil {
			return err
		}

		// 庄家押注：本金入池
		betId := b.Id
		if err := repositories.BattleBetRepository.Create(tx, &models.BattleBet{
			BattleId:   b.Id,
			UserId:     bankerUserId,
			Role:       BattleRoleBanker,
			Amount:     form.StakeAmount,
			EntryFee:   0,
			RequestId:  fmt.Sprintf("create-%d", b.Id),
			CreateTime: now,
			UpdateTime: now,
		}); err != nil {
			return err
		}

		if err := UserCoinService.SpendToPool(tx, bankerUserId, "BATTLE_STAKE_IN", betId, form.StakeAmount, fmt.Sprintf("battle stake in: battleId=%d", b.Id)); err != nil {
			return err
		}

		_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
			BattleId:   b.Id,
			UserId:     bankerUserId,
			Action:     "stake_in",
			RequestId:  fmt.Sprintf("create-%d", b.Id),
			Amount:     -form.StakeAmount,
			Remark:     "banker stake in",
			CreateTime: now,
		})

		// 确保系统账户存在（资金池/销毁）
		_, _ = repositories.UserCoinRepository.GetOrCreate(tx, models.BattlePoolUserId)
		_, _ = repositories.UserCoinRepository.GetOrCreate(tx, models.BattleBurnUserId)

		return nil
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

// JoinOrAddStake 挑战者加入/追加下注（支持多次，幂等通过 requestId）。
func (s *battleService) JoinOrAddStake(challengerUserId int64, form JoinBattleForm) (*models.Battle, *models.BattleBet, error) {
	if challengerUserId <= 0 {
		return nil, nil, errors.New("challengerUserId is required")
	}
	if form.BattleId <= 0 {
		return nil, nil, errors.New("battleId is required")
	}
	if form.Amount <= 0 {
		return nil, nil, errors.New("amount must be positive")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, nil, errors.New("requestId is required")
	}

	now := dates.NowTimestamp()
	var (
		battle *models.Battle
		bet    *models.BattleBet
	)

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		battle = b

		if b.BankerUserId == challengerUserId {
			return errors.New("banker cannot join as challenger")
		}
		if b.Status != BattleStatusOpen {
			return errors.New("battle is not open")
		}
		if !b.IsPublic {
			if strings.TrimSpace(form.InviteCode) == "" || strings.TrimSpace(form.InviteCode) != strings.TrimSpace(b.InviteCode) {
				return errors.New("invalid inviteCode")
			}
		}

		// 幂等：已存在同 requestId 的下注明细则直接返回
		exists := repositories.BattleBetRepository.TakeByUserRequest(tx, b.Id, challengerUserId, form.RequestId)
		if exists != nil {
			bet = exists
			return nil
		}

		// 容量校验：挑战者总额不能超过庄家押注总额
		challengerSum, err := repositories.BattleBetRepository.SumChallengerStake(tx, b.Id)
		if err != nil {
			return err
		}
		remaining := b.BankerStakeTotal - challengerSum
		if remaining <= 0 {
			return errors.New("battle is full")
		}
		if form.Amount > remaining {
			return fmt.Errorf("amount exceeds remaining capacity: %d", remaining)
		}

		// 入场费（公开 5%），向下取整
		entryFee := int64(0)
		if b.IsPublic {
			entryFee = int64(math.Floor(float64(form.Amount) * 0.05))
			if entryFee < 0 {
				entryFee = 0
			}
		}

		// 下注明细
		betRecord := &models.BattleBet{
			BattleId:   b.Id,
			UserId:     challengerUserId,
			Role:       BattleRoleChallenger,
			Amount:     form.Amount,
			EntryFee:   entryFee,
			RequestId:  form.RequestId,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.BattleBetRepository.Create(tx, betRecord); err != nil {
			return err
		}
		bet = betRecord

		// 1) 本金入池
		if err := UserCoinService.SpendToPool(tx, challengerUserId, "BATTLE_STAKE_IN", betRecord.Id, form.Amount, fmt.Sprintf("battle stake in: battleId=%d", b.Id)); err != nil {
			return err
		}
		_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
			BattleId:   b.Id,
			UserId:     challengerUserId,
			Action:     "stake_in",
			RequestId:  form.RequestId,
			Amount:     -form.Amount,
			Remark:     "challenger stake in",
			CreateTime: now,
		})

		// 2) 入场费转给庄家
		if entryFee > 0 {
			if err := UserCoinService.PayEntryFeeToBanker(tx, challengerUserId, b.BankerUserId, "BATTLE_ENTRY_FEE", betRecord.Id, entryFee, fmt.Sprintf("battle entry fee: battleId=%d", b.Id)); err != nil {
				return err
			}
			_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
				BattleId:   b.Id,
				UserId:     challengerUserId,
				Action:     "entry_fee",
				RequestId:  form.RequestId,
				Amount:     -entryFee,
				Remark:     "challenger entry fee",
				CreateTime: now,
			})
			_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
				BattleId:   b.Id,
				UserId:     b.BankerUserId,
				Action:     "entry_fee_income",
				RequestId:  form.RequestId,
				Amount:     entryFee,
				Remark:     "banker entry fee income",
				CreateTime: now,
			})
			b.EntryFeeTotal += entryFee
		}

		b.ChallengerStakeTotal += form.Amount
		b.PoolPrincipalTotal += form.Amount
		b.UpdateTime = now

		// 达到容量自动封盘
		if b.ChallengerStakeTotal >= b.BankerStakeTotal {
			b.Status = BattleStatusSealed
		}

		return repositories.BattleRepository.Update(tx, b)
	})
	if err != nil {
		return nil, nil, err
	}
	return battle, bet, nil
}

func (s *battleService) BankerAddStake(bankerUserId int64, form BankerAddStakeForm) (*models.Battle, error) {
	if bankerUserId <= 0 {
		return nil, errors.New("bankerUserId is required")
	}
	if form.BattleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	if form.Amount <= 0 {
		return nil, errors.New("amount must be positive")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}

	now := dates.NowTimestamp()
	var battle *models.Battle

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		battle = b
		if b.BankerUserId != bankerUserId {
			return errors.New("permission denied")
		}
		if b.Status == BattleStatusPending || b.Status == BattleStatusDisputed || b.Status == BattleStatusSettled {
			return errors.New("battle is not allowed to add stake")
		}
		if b.SettleTime <= now {
			return errors.New("battle already reached settle time")
		}

		// 幂等：存在同 requestId 的庄家 bet 记录则返回
		exists := repositories.BattleBetRepository.TakeByUserRequest(tx, b.Id, bankerUserId, form.RequestId)
		if exists != nil {
			return nil
		}

		betRecord := &models.BattleBet{
			BattleId:   b.Id,
			UserId:     bankerUserId,
			Role:       BattleRoleBanker,
			Amount:     form.Amount,
			EntryFee:   0,
			RequestId:  form.RequestId,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := repositories.BattleBetRepository.Create(tx, betRecord); err != nil {
			return err
		}
		if err := UserCoinService.SpendToPool(tx, bankerUserId, "BATTLE_STAKE_IN", betRecord.Id, form.Amount, fmt.Sprintf("battle banker add stake: battleId=%d", b.Id)); err != nil {
			return err
		}

		b.BankerStakeTotal += form.Amount
		b.PoolPrincipalTotal += form.Amount
		b.UpdateTime = now
		// 若之前封盘但还没到结算时间，允许重新 open
		if b.Status == BattleStatusSealed {
			b.Status = BattleStatusOpen
		}
		return repositories.BattleRepository.Update(tx, b)
	})
	if err != nil {
		return nil, err
	}
	return battle, nil
}

// SealIfNeeded open->sealed：到达结算时间或容量满
func (s *battleService) SealIfNeeded(tx *gorm.DB, b *models.Battle, now int64) (bool, error) {
	if b.Status != BattleStatusOpen {
		return false, nil
	}
	if b.SettleTime <= now || b.ChallengerStakeTotal >= b.BankerStakeTotal {
		b.Status = BattleStatusSealed
		b.UpdateTime = now
		return true, repositories.BattleRepository.Update(tx, b)
	}
	return false, nil
}

// MoveToPending sealed->pending：到达结算时间
func (s *battleService) MoveToPending(tx *gorm.DB, b *models.Battle, now int64) (bool, error) {
	if b.Status != BattleStatusSealed {
		return false, nil
	}
	if b.SettleTime <= now {
		b.Status = BattleStatusPending
		b.PendingDeadline = now + 24*3600
		b.UpdateTime = now
		return true, repositories.BattleRepository.Update(tx, b)
	}
	return false, nil
}

// DeclareResultByBanker 庄家宣布结果（pending 状态）
func (s *battleService) DeclareResultByBanker(bankerUserId, battleId int64, result string) (*models.Battle, error) {
	if bankerUserId <= 0 {
		return nil, errors.New("bankerUserId is required")
	}
	if battleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	result = strings.TrimSpace(strings.ToLower(result))
	if result != BattleResultBankerWins && result != BattleResultBankerLoses {
		return nil, errors.New("invalid result")
	}
	// 仲裁链路：庄家宣布后进入挑战者确认窗口（24h）。
	// - 所有挑战者 confirm 或 confirm 超时默认确认 => settled
	// - 任一挑战者 dispute => disputed（管理员 24h）
	now := dates.NowTimestamp()
	var battle *models.Battle
	if err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, battleId)
		if err != nil {
			return err
		}
		if b.BankerUserId != bankerUserId {
			return errors.New("permission denied")
		}
		if b.Status != BattleStatusPending {
			return errors.New("battle is not pending")
		}
		b.Result = result
		b.ResultBy = "banker"
		b.ResultTime = now
		b.ConfirmDeadline = now + 24*3600
		b.ConfirmedCount = 0
		b.UpdateTime = now
		if err := repositories.BattleRepository.Update(tx, b); err != nil {
			return err
		}
		battle = b
		return nil
	}); err != nil {
		return nil, err
	}
	return battle, nil
}

// ChallengeConfirm 挑战者确认（幂等）：记录动作；若确认人数达到挑战者人数则 settled 并生成结算单。
func (s *battleService) ChallengeConfirm(userId int64, form ChallengeActionForm) (*models.Battle, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if form.BattleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}
	now := dates.NowTimestamp()
	var battle *models.Battle
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		if b.Status != BattleStatusPending {
			return errors.New("battle is not pending")
		}
		if b.Result == "" {
			return errors.New("banker has not declared result")
		}
		// 必须是挑战者
		stake, err := repositories.BattleBetRepository.SumUserStake(tx, b.Id, userId)
		if err != nil {
			return err
		}
		if stake <= 0 {
			return errors.New("only challenger can confirm")
		}
		// 幂等：按 requestId
		if repositories.BattleChallengeActionRepository.TakeByRequest(tx, b.Id, userId, form.RequestId) != nil {
			battle = b
			return nil
		}
		// 已做过动作（confirm/dispute）则拒绝重复操作
		if repositories.BattleChallengeActionRepository.TakeByUser(tx, b.Id, userId) != nil {
			battle = b
			return nil
		}
		if err := repositories.BattleChallengeActionRepository.Create(tx, &models.BattleChallengeAction{
			BattleId:   b.Id,
			UserId:     userId,
			Action:     "confirm",
			RequestId:  form.RequestId,
			Remark:     strings.TrimSpace(form.Remark),
			CreateTime: now,
		}); err != nil {
			return err
		}

		// 统计确认人数
		var confirmed int64
		if err := tx.Model(&models.BattleChallengeAction{}).
			Where("battle_id = ? AND action = ?", b.Id, "confirm").
			Count(&confirmed).Error; err != nil {
			return err
		}
		b.ConfirmedCount = confirmed
		b.UpdateTime = now

		// 统计挑战者人数
		// 以下注用户维度去重
		type row struct{ UserId int64 }
		var rows []row
		if err := tx.Model(&models.BattleBet{}).
			Select("DISTINCT user_id as user_id").
			Where("battle_id = ? AND role = ?", b.Id, BattleRoleChallenger).
			Scan(&rows).Error; err != nil {
			return err
		}
		if confirmed >= int64(len(rows)) {
			b.Status = BattleStatusSettled
			b.UpdateTime = now
			if err := repositories.BattleRepository.Update(tx, b); err != nil {
				return err
			}
			if _, err := s.GenerateSettlement(tx, b); err != nil {
				return err
			}
			battle = b
			return nil
		}

		if err := repositories.BattleRepository.Update(tx, b); err != nil {
			return err
		}
		battle = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return battle, nil
}

// ChallengeDispute 挑战者异议（幂等）：进入 disputed，等待管理员裁决。
func (s *battleService) ChallengeDispute(userId int64, form ChallengeActionForm) (*models.Battle, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if form.BattleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}
	now := dates.NowTimestamp()
	var battle *models.Battle
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		if b.Status != BattleStatusPending {
			return errors.New("battle is not pending")
		}
		if b.Result == "" {
			return errors.New("banker has not declared result")
		}
		stake, err := repositories.BattleBetRepository.SumUserStake(tx, b.Id, userId)
		if err != nil {
			return err
		}
		if stake <= 0 {
			return errors.New("only challenger can dispute")
		}
		if repositories.BattleChallengeActionRepository.TakeByRequest(tx, b.Id, userId, form.RequestId) != nil {
			battle = b
			return nil
		}
		if repositories.BattleChallengeActionRepository.TakeByUser(tx, b.Id, userId) != nil {
			battle = b
			return nil
		}
		if err := repositories.BattleChallengeActionRepository.Create(tx, &models.BattleChallengeAction{
			BattleId:   b.Id,
			UserId:     userId,
			Action:     "dispute",
			RequestId:  form.RequestId,
			Remark:     strings.TrimSpace(form.Remark),
			CreateTime: now,
		}); err != nil {
			return err
		}
		b.Status = BattleStatusDisputed
		b.DisputeDeadline = now + 24*3600
		b.DisputedByUserId = userId
		b.UpdateTime = now
		if err := repositories.BattleRepository.Update(tx, b); err != nil {
			return err
		}
		battle = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return battle, nil
}

// AdminResolve 管理员裁决（disputed）：裁决结果并进入 settled + 生成结算单。
func (s *battleService) AdminResolve(adminUserId int64, form AdminResolveForm) (*models.Battle, error) {
	if adminUserId <= 0 {
		return nil, errors.New("adminUserId is required")
	}
	if form.BattleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}
	res := strings.TrimSpace(strings.ToLower(form.Result))
	if res != BattleResultBankerWins && res != BattleResultBankerLoses && res != BattleResultVoid {
		return nil, errors.New("invalid result")
	}
	now := dates.NowTimestamp()
	var battle *models.Battle
	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		if b.Status != BattleStatusDisputed {
			return errors.New("battle is not disputed")
		}
		// 幂等：用 ledger 记录 admin_resolve
		if repositories.BattleLedgerRepository.TakeIdempotent(tx, b.Id, adminUserId, "admin_resolve", form.RequestId) != nil {
			battle = b
			return nil
		}
		_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
			BattleId:   b.Id,
			UserId:     adminUserId,
			Action:     "admin_resolve",
			RequestId:  form.RequestId,
			Amount:     0,
			Remark:     strings.TrimSpace(form.Remark),
			CreateTime: now,
		})

		// 处罚：按 10% 从资金池 burn。
		// 口径：处罚 burn 与 void 规则 burn 是两条独立流水；均以 BattleLedger 幂等。
		penalty := int64(0)
		if b.PoolPrincipalTotal > 0 {
			penalty = int64(math.Floor(float64(b.PoolPrincipalTotal) * 0.10))
		}
		if penalty > 0 {
			// 用同一个 requestId 作为幂等点，避免重复裁决请求重复处罚
			exists := repositories.BattleLedgerRepository.TakeIdempotent(
				tx,
				b.Id,
				models.BattlePoolUserId,
				"penalty_burn",
				form.RequestId,
			)
			if exists == nil {
				if err := UserCoinService.BurnFromPool(
					tx,
					"BATTLE_PENALTY_BURN",
					b.Id,
					penalty,
					fmt.Sprintf("battle penalty burn: battleId=%d", b.Id),
				); err != nil {
					return err
				}
				_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
					BattleId:   b.Id,
					UserId:     models.BattlePoolUserId,
					Action:     "penalty_burn",
					RequestId:  form.RequestId,
					Amount:     -penalty,
					Remark:     "penalty burn from pool",
					CreateTime: now,
				})
				b.BurnTotal += penalty
			}
		}

		b.Result = res
		b.ResultBy = "admin"
		b.ResultTime = now
		b.Status = BattleStatusSettled
		b.UpdateTime = now
		if err := repositories.BattleRepository.Update(tx, b); err != nil {
			return err
		}
		if _, err := s.GenerateSettlement(tx, b); err != nil {
			return err
		}
		battle = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return battle, nil
}

// GenerateSettlement 生成结算单（幂等：battle_id 唯一）。
func (s *battleService) GenerateSettlement(tx *gorm.DB, b *models.Battle) (*models.BattleSettlement, error) {
	if b == nil {
		return nil, errors.New("battle is nil")
	}
	if b.Status != BattleStatusSettled {
		return nil, errors.New("battle is not settled")
	}
	if b.Result != BattleResultBankerWins && b.Result != BattleResultBankerLoses && b.Result != BattleResultVoid {
		return nil, errors.New("battle result is invalid")
	}

	existing := repositories.BattleSettlementRepository.TakeByBattleId(tx, b.Id)
	if existing != nil {
		return existing, nil
	}

	now := dates.NowTimestamp()
	st := &models.BattleSettlement{BattleId: b.Id, Result: b.Result, CreatedAt: now}
	if err := repositories.BattleSettlementRepository.Create(tx, st); err != nil {
		return nil, err
	}

	// 聚合挑战者下注
	type userSum struct {
		UserId int64
		Sum    int64
	}
	var cs []userSum
	if err := tx.Model(&models.BattleBet{}).
		Select("user_id as user_id, COALESCE(SUM(amount),0) as sum").
		Where("battle_id = ? AND role = ?", b.Id, BattleRoleChallenger).
		Group("user_id").
		Scan(&cs).Error; err != nil {
		return nil, err
	}
	challengerTotal := int64(0)
	for _, r := range cs {
		challengerTotal += r.Sum
	}

	// 结算规则（与文档一致，出池只涉及资金池本金，入场费已实时转给庄家不在结算单体现）：
	// - banker_wins：池内本金全给庄家
	// - banker_loses：挑战者按比例分得庄家本金 + 退还各自本金
	// - void：挑战者退还本金；庄家退还 90% 本金；10% burn
	if b.Result == BattleResultBankerWins {
		payout := b.BankerStakeTotal + challengerTotal
		if err := repositories.BattleSettlementRepository.CreateItem(tx, &models.BattleSettlementItem{
			SettlementId: st.Id,
			BattleId:     b.Id,
			UserId:       b.BankerUserId,
			PayoutAmount: payout,
			CreateTime:   now,
			UpdateTime:   now,
		}); err != nil {
			return nil, err
		}
		return st, nil
	}

	if b.Result == BattleResultBankerLoses {
		if challengerTotal <= 0 {
			// 没有挑战者，退还庄家本金
			if err := repositories.BattleSettlementRepository.CreateItem(tx, &models.BattleSettlementItem{
				SettlementId: st.Id,
				BattleId:     b.Id,
				UserId:       b.BankerUserId,
				PayoutAmount: b.BankerStakeTotal,
				CreateTime:   now,
				UpdateTime:   now,
			}); err != nil {
				return nil, err
			}
			return st, nil
		}
		for _, r := range cs {
			share := int64(math.Floor(float64(b.BankerStakeTotal) * (float64(r.Sum) / float64(challengerTotal))))
			payout := r.Sum + share
			if payout < 0 {
				payout = 0
			}
			if err := repositories.BattleSettlementRepository.CreateItem(tx, &models.BattleSettlementItem{
				SettlementId: st.Id,
				BattleId:     b.Id,
				UserId:       r.UserId,
				PayoutAmount: payout,
				CreateTime:   now,
				UpdateTime:   now,
			}); err != nil {
				return nil, err
			}
		}
		return st, nil
	}

	// void
	burn := int64(math.Floor(float64(b.BankerStakeTotal) * 0.10))
	bankerBack := b.BankerStakeTotal - burn
	if bankerBack < 0 {
		bankerBack = 0
	}
	if err := repositories.BattleSettlementRepository.CreateItem(tx, &models.BattleSettlementItem{
		SettlementId: st.Id,
		BattleId:     b.Id,
		UserId:       b.BankerUserId,
		PayoutAmount: bankerBack,
		CreateTime:   now,
		UpdateTime:   now,
	}); err != nil {
		return nil, err
	}
	for _, r := range cs {
		if r.Sum <= 0 {
			continue
		}
		if err := repositories.BattleSettlementRepository.CreateItem(tx, &models.BattleSettlementItem{
			SettlementId: st.Id,
			BattleId:     b.Id,
			UserId:       r.UserId,
			PayoutAmount: r.Sum,
			CreateTime:   now,
			UpdateTime:   now,
		}); err != nil {
			return nil, err
		}
	}
	// burn 发生在提取时从池里划转（为了幂等与可追溯），这里仅记录 battle 冗余统计
	b.BurnTotal += burn
	_ = repositories.BattleRepository.Update(tx, b)
	return st, nil
}

// Withdraw 一次性全提（幂等：重复请求不重复出池）。
func (s *battleService) Withdraw(userId int64, form WithdrawForm) (*models.BattleSettlementItem, error) {
	if userId <= 0 {
		return nil, errors.New("userId is required")
	}
	if form.BattleId <= 0 {
		return nil, errors.New("battleId is required")
	}
	form.RequestId = strings.TrimSpace(form.RequestId)
	if form.RequestId == "" {
		return nil, errors.New("requestId is required")
	}

	now := dates.NowTimestamp()
	var item *models.BattleSettlementItem

	err := sqls.DB().Transaction(func(tx *gorm.DB) error {
		b, err := repositories.BattleRepository.TakeForUpdate(tx, form.BattleId)
		if err != nil {
			return err
		}
		if b.Status != BattleStatusSettled {
			return errors.New("battle is not settled")
		}

		// 确保结算单存在
		st, err := s.GenerateSettlement(tx, b)
		if err != nil {
			return err
		}

		it := repositories.BattleSettlementRepository.TakeItemByBattleUser(tx, b.Id, userId)
		if it == nil {
			return errors.New("no payout for this user")
		}

		// 幂等：已提取直接返回
		if it.Withdrawn {
			item = it
			return nil
		}

		// 出池（payoutAmount）
		if it.PayoutAmount > 0 {
			if err := UserCoinService.PayFromPoolToUser(tx, userId, "BATTLE_WITHDRAW", it.Id, it.PayoutAmount, fmt.Sprintf("battle withdraw: battleId=%d", b.Id)); err != nil {
				return err
			}
		}

		it.Withdrawn = true
		it.WithdrawRequestId = form.RequestId
		it.WithdrawTime = now
		it.UpdateTime = now
		if err := tx.Save(it).Error; err != nil {
			return err
		}

		// void 场景下需要 burn（一次性：找 banker 的结算明细是否已提取会影响 burn 时机；这里用 st.Id 做幂等 ledger）
		if b.Result == BattleResultVoid {
			burn := int64(math.Floor(float64(b.BankerStakeTotal) * 0.10))
			if burn > 0 {
				// 用 settlement 作为幂等点，确保 burn 只执行一次
				exists := repositories.BattleLedgerRepository.TakeIdempotent(tx, b.Id, models.BattlePoolUserId, "burn", fmt.Sprintf("settlement-%d", st.Id))
				if exists == nil {
					if err := UserCoinService.BurnFromPool(tx, "BATTLE_BURN", st.Id, burn, fmt.Sprintf("battle burn: battleId=%d", b.Id)); err != nil {
						return err
					}
					_ = repositories.BattleLedgerRepository.Create(tx, &models.BattleLedger{
						BattleId:   b.Id,
						UserId:     models.BattlePoolUserId,
						Action:     "burn",
						RequestId:  fmt.Sprintf("settlement-%d", st.Id),
						Amount:     -burn,
						Remark:     "burn from pool",
						CreateTime: now,
					})
				}
			}
		}

		item = it
		_ = st
		return nil
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// CronTick 后台轮巡：封盘、到期 pending、庄家超时判负。
// 说明：一期先实现 open->sealed 与 sealed->pending 与 pending 超时；挑战者确认/争议仲裁后续再加。
func (s *battleService) CronTick() error {
	now := dates.NowTimestamp()
	return sqls.DB().Transaction(func(tx *gorm.DB) error {
		var battles []*models.Battle
		// 扫描可能需要状态迁移的 battle
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status IN ?", []string{BattleStatusOpen, BattleStatusSealed, BattleStatusPending, BattleStatusDisputed}).
			Find(&battles).Error; err != nil {
			return err
		}
		for _, b := range battles {
			changed, err := s.SealIfNeeded(tx, b, now)
			if err != nil {
				return err
			}
			if changed {
				// continue to next steps with updated status
			}
			_, err = s.MoveToPending(tx, b, now)
			if err != nil {
				return err
			}

			if b.Status == BattleStatusPending && b.PendingDeadline > 0 && now > b.PendingDeadline && b.Result == "" {
				// 超时未宣布：自动判庄家输，直接 settled 并生成结算单
				b.Result = BattleResultBankerLoses
				b.ResultBy = "timeout"
				b.ResultTime = now
				b.Status = BattleStatusSettled
				b.UpdateTime = now
				if err := repositories.BattleRepository.Update(tx, b); err != nil {
					return err
				}
				if _, err := s.GenerateSettlement(tx, b); err != nil {
					return err
				}
			}

			// 挑战者确认超时：默认确认；全部确认后 settled
			if b.Status == BattleStatusPending && b.Result != "" && b.ConfirmDeadline > 0 && now > b.ConfirmDeadline {
				// 找到所有挑战者 userId
				type row struct{ UserId int64 }
				var rows []row
				if err := tx.Model(&models.BattleBet{}).
					Select("DISTINCT user_id as user_id").
					Where("battle_id = ? AND role = ?", b.Id, BattleRoleChallenger).
					Scan(&rows).Error; err != nil {
					return err
				}
				for _, r := range rows {
					// 未动作的挑战者补一条 confirm（系统 requestId）
					if repositories.BattleChallengeActionRepository.TakeByUser(tx, b.Id, r.UserId) == nil {
						_ = repositories.BattleChallengeActionRepository.Create(tx, &models.BattleChallengeAction{
							BattleId:   b.Id,
							UserId:     r.UserId,
							Action:     "confirm",
							RequestId:  fmt.Sprintf("timeout-confirm-%d", b.Id),
							Remark:     "timeout default confirm",
							CreateTime: now,
						})
					}
				}
				b.Status = BattleStatusSettled
				b.ResultBy = "confirm_timeout"
				b.UpdateTime = now
				if err := repositories.BattleRepository.Update(tx, b); err != nil {
					return err
				}
				if _, err := s.GenerateSettlement(tx, b); err != nil {
					return err
				}
			}

			// disputed 管理员超时：默认 void
			if b.Status == BattleStatusDisputed && b.DisputeDeadline > 0 && now > b.DisputeDeadline {
				b.Result = BattleResultVoid
				b.ResultBy = "admin_timeout"
				b.ResultTime = now
				b.Status = BattleStatusSettled
				b.UpdateTime = now
				if err := repositories.BattleRepository.Update(tx, b); err != nil {
					return err
				}
				if _, err := s.GenerateSettlement(tx, b); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
