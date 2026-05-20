#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  scripts/export_owner_users.sh [output_dir]

Exports owner-role users from a TurtlePoll/bbs-go PostgreSQL database.

Database:
  postgres://appuser:root@52.77.212.173:5432/turtlepoll?sslmode=disable

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

DB_URL="postgres://appuser:root@52.77.212.173:5432/turtlepoll?sslmode=disable"

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
  WITH owner_users AS (
    SELECT DISTINCT u.id
    FROM t_user u
    LEFT JOIN t_user_role ur ON ur.user_id = u.id
    LEFT JOIN t_role r ON r.id = ur.role_id
    WHERE r.code = 'owner'
       OR EXISTS (
         SELECT 1
         FROM regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code
         WHERE trim(role_code) = 'owner'
       )
  )
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
  JOIN owner_users ou ON ou.id = u.id
  ORDER BY u.id
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$USERS_CSV"

psql "$DB_URL" -v ON_ERROR_STOP=1 -q -c "COPY (
  WITH owner_users AS (
    SELECT DISTINCT u.id
    FROM t_user u
    LEFT JOIN t_user_role ur ON ur.user_id = u.id
    LEFT JOIN t_role r ON r.id = ur.role_id
    WHERE r.code = 'owner'
       OR EXISTS (
         SELECT 1
         FROM regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code
         WHERE trim(role_code) = 'owner'
       )
  ),
  existing_roles AS (
    SELECT DISTINCT
      ur.user_id,
      r.code AS role_code,
      ur.create_time
    FROM t_user_role ur
    JOIN t_role r ON r.id = ur.role_id
    JOIN owner_users ou ON ou.id = ur.user_id
  ),
  roles_from_user_cache AS (
    SELECT DISTINCT
      u.id AS user_id,
      trim(role_code) AS role_code,
      u.create_time
    FROM t_user u
    JOIN owner_users ou ON ou.id = u.id
    CROSS JOIN regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code
    WHERE trim(role_code) <> ''
  )
  SELECT
    user_id,
    role_code,
    MIN(create_time) AS create_time
  FROM (
    SELECT * FROM existing_roles
    UNION ALL
    SELECT * FROM roles_from_user_cache
  ) roles
  GROUP BY user_id, role_code
  ORDER BY user_id, role_code
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$USER_ROLES_CSV"

psql "$DB_URL" -v ON_ERROR_STOP=1 -q -c "COPY (
  WITH owner_users AS (
    SELECT DISTINCT u.id
    FROM t_user u
    LEFT JOIN t_user_role ur ON ur.user_id = u.id
    LEFT JOIN t_role r ON r.id = ur.role_id
    WHERE r.code = 'owner'
       OR EXISTS (
         SELECT 1
         FROM regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code
         WHERE trim(role_code) = 'owner'
       )
  )
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
  JOIN owner_users ou ON ou.id = tu.user_id
  ORDER BY tu.id
) TO STDOUT WITH (FORMAT csv, HEADER true, FORCE_QUOTE *);" >"$THIRD_USERS_CSV"

owner_count="$(psql "$DB_URL" -At -v ON_ERROR_STOP=1 -c "WITH owner_users AS (SELECT DISTINCT u.id FROM t_user u LEFT JOIN t_user_role ur ON ur.user_id = u.id LEFT JOIN t_role r ON r.id = ur.role_id WHERE r.code = 'owner' OR EXISTS (SELECT 1 FROM regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code WHERE trim(role_code) = 'owner')) SELECT count(*) FROM owner_users;")"
third_user_count="$(psql "$DB_URL" -At -v ON_ERROR_STOP=1 -c "WITH owner_users AS (SELECT DISTINCT u.id FROM t_user u LEFT JOIN t_user_role ur ON ur.user_id = u.id LEFT JOIN t_role r ON r.id = ur.role_id WHERE r.code = 'owner' OR EXISTS (SELECT 1 FROM regexp_split_to_table(COALESCE(u.roles, ''), ',') AS role_code WHERE trim(role_code) = 'owner')) SELECT count(*) FROM t_third_user tu JOIN owner_users ou ON ou.id = tu.user_id;")"

log "exported owner users: $owner_count"
log "exported third-party bindings: $third_user_count"
log "done"
