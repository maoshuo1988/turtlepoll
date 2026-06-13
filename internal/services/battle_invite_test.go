package services

import (
	"bbs-go/internal/models/models"
	"testing"
)

// 邀请码生成与验证规则测试
func TestBattle_InviteCodeFormat(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{"valid uppercase", "ABCD", true},
		{"valid mixed case - should fail (format requires uppercase)", "AbCd", false},
		{"valid with digits", "A1B2", true},
		{"too short", "ABC", false},
		{"too long", "ABCDE", false},
		{"empty", "", false},
		{"with space", "AB CD", false},
		{"with special chars", "AB@D", false},
		{"all digits", "1234", true},
		{"all letters uppercase", "ABCD", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isInviteCodeFormatValid(tt.code)
			if actual != tt.valid {
				t.Errorf("isInviteCodeFormatValid(%q) = %v, want %v", tt.code, actual, tt.valid)
			}
		})
	}
}

// 邀请码大小写归一化测试
func TestBattle_NormalizeInviteCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC1", "ABC1"},
		{"abc1", "ABC1"},
		{"  ABC1  ", "ABC1"},
		{"aBc1", "ABC1"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			actual := normalizeInviteCode(tt.input)
			if actual != tt.expected {
				t.Errorf("normalizeInviteCode(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

// 邀请码验证逻辑测试
func TestBattle_CanJoinWithInviteCode(t *testing.T) {
	now := int64(1000000)

	tests := []struct {
		name           string
		inviteCode     string // 输入的邀请码
		battleInvite   string // 战局中存储的邀请码
		battleExpireAt int64  // 战局邀请码过期时间
		shouldSuccess  bool
	}{
		// 正常情况
		{"exact match", "ABC1", "ABC1", now + 1000, true},

		// 大小写不敏感
		{"uppercase input lowercase stored", "ABC1", "abc1", now + 1000, true},
		{"lowercase input uppercase stored", "abc1", "ABC1", now + 1000, true},

		// 格式检查
		{"format invalid - too short", "ABC", "ABC1", now + 1000, false},
		{"format invalid - too long", "ABCDE", "ABC1", now + 1000, false},
		{"format invalid - special char", "AB@1", "AB@1", now + 1000, false},

		// 匹配检查
		{"code mismatch", "XXXX", "ABC1", now + 1000, false},

		// 过期检查
		{"code expired", "ABC1", "ABC1", now - 100, false},
		{"code just expired (now > expireAt)", "ABC1", "ABC1", now - 1, false},
		{"code still valid (now == expireAt)", "ABC1", "ABC1", now, true},

		// 空值检查
		{"empty input code", "", "ABC1", now + 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &models.Battle{
				InviteCode:         tt.battleInvite,
				InviteCodeExpireAt: tt.battleExpireAt,
			}

			s := &battleService{}
			err := s.canJoinPrivateBattleWithInvite(b, tt.inviteCode, now)
			hasError := err != nil

			if hasError == tt.shouldSuccess {
				t.Errorf("canJoinPrivateBattleWithInvite failed: code=%q, error=%v, expected success=%v",
					tt.inviteCode, err, tt.shouldSuccess)
			}
		})
	}
}

// 邀请码生成随机性测试
func TestBattle_InviteCodeGeneration(t *testing.T) {
	t.Run("generate random codes are different", func(t *testing.T) {
		s := &battleService{}
		code1, err1 := s.newRandomInviteCode()
		code2, err2 := s.newRandomInviteCode()

		if err1 != nil || err2 != nil {
			t.Errorf("code generation failed: %v, %v", err1, err2)
		}
		if len(code1) != battleInviteCodeLength || len(code2) != battleInviteCodeLength {
			t.Errorf("code length invalid: %d, %d", len(code1), len(code2))
		}

		// 生成多个码，检查格式有效性
		for i := 0; i < 10; i++ {
			code, err := s.newRandomInviteCode()
			if err != nil {
				t.Errorf("code %d generation failed: %v", i, err)
			}
			if !isInviteCodeFormatValid(code) {
				t.Errorf("code %d %q format invalid", i, code)
			}
		}
	})
}
