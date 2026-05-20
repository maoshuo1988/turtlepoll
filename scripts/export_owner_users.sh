#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  scripts/export_owner_users.sh [output_dir]

Exports owner-role users from a TurtlePoll/bbs-go PostgreSQL database.

Database:
  postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable

Output:
  output_dir/users.csv
  output_dir/user_roles.csv
  output_dir/third_users.csv
  output_dir/metadata.txt

Notes:
  - Login tokens are intentionally not exported.
  - The target environment should already be installed/migrated before import.
EOF
}

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [export_owner_users] $*" >&2
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

require_cmd psql

DB_URL="postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable"

OUTPUT_DIR="${1:-owner-users-export-$(date '+%Y%m%d-%H%M%S')}"
mkdir -p "$OUTPUT_DIR"

USERS_CSV="$OUTPUT_DIR/users.csv"
USER_ROLES_CSV="$OUTPUT_DIR/user_roles.csv"
THIRD_USERS_CSV="$OUTPUT_DIR/third_users.csv"

log "exporting owner users to $OUTPUT_DIR"

cat >"$OUTPUT_DIR/metadata.txt" <<EOF
exported_at=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
format=turtlepoll-owner-users-v1
tables=t_user,t_user_role,t_third_user
EOF

psql "$DB_URL" -v ON_ERROR_STOP=1 -q -c "COPY (
  SELECT DISTINCT
    u.id,
    u.type,
    u.phone,
    u.username,
    u.email,
    u.email_verified,
    u.nickname,
    u.avatar,
    u.gender,
    u.birthday,
    u.background_image,
    u.password,
    u.home_page,
    u.description,
    u.score,
    u.exp,
    u.level,
    u.status,
    u.topic_count,
    u.comment_count,
    u.follow_count,
    u.fans_count,
    u.roles,
    u.forbidden_end_time,
    u.create_time,
    u.update_time
  FROM t_user u
  JOIN t_user_role ur ON ur.user_id = u.id
  JOIN t_role r ON r.id = ur.role_id
  WHERE r.code = 'owner'
  ORDER BY u.id
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$USERS_CSV"

psql "$DB_URL" -v ON_ERROR_STOP=1 -q -c "COPY (
  SELECT DISTINCT
    ur.user_id,
    r.code AS role_code,
    ur.create_time
  FROM t_user_role ur
  JOIN t_role r ON r.id = ur.role_id
  JOIN t_user_role owner_ur ON owner_ur.user_id = ur.user_id
  JOIN t_role owner_role ON owner_role.id = owner_ur.role_id
  WHERE owner_role.code = 'owner'
  ORDER BY ur.user_id, r.code
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$USER_ROLES_CSV"

psql "$DB_URL" -v ON_ERROR_STOP=1 -q -c "COPY (
  SELECT
    tu.id,
    tu.user_id,
    tu.open_id,
    tu.third_type,
    tu.nickname,
    tu.avatar,
    tu.extra_data,
    tu.create_time,
    tu.update_time
  FROM t_third_user tu
  JOIN t_user_role owner_ur ON owner_ur.user_id = tu.user_id
  JOIN t_role owner_role ON owner_role.id = owner_ur.role_id
  WHERE owner_role.code = 'owner'
  ORDER BY tu.id
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$THIRD_USERS_CSV"

owner_count="$(psql "$DB_URL" -At -v ON_ERROR_STOP=1 -c "SELECT count(DISTINCT u.id) FROM t_user u JOIN t_user_role ur ON ur.user_id = u.id JOIN t_role r ON r.id = ur.role_id WHERE r.code = 'owner';")"
third_user_count="$(psql "$DB_URL" -At -v ON_ERROR_STOP=1 -c "SELECT count(*) FROM t_third_user tu JOIN t_user_role ur ON ur.user_id = tu.user_id JOIN t_role r ON r.id = ur.role_id WHERE r.code = 'owner';")"

log "exported owner users: $owner_count"
log "exported third-party bindings: $third_user_count"
log "done"
