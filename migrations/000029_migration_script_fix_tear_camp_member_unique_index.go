package migrations

import "github.com/mlogclub/simple/sqls"

// migrate_fix_tear_camp_member_unique_index
// 修复 t_tear_camp_member 唯一索引维度，加入 topic_id，避免同一用户跨市场冲突。
func migrate_fix_tear_camp_member_unique_index() error {
	stmts := []string{
		"DROP INDEX IF EXISTS idx_tear_camp_event_user",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_tear_camp_event_user ON t_tear_camp_member(event_type, topic_id, round_id, user_id)",
	}
	for _, stmt := range stmts {
		if err := sqls.DB().Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
