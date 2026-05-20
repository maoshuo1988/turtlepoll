#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
Usage:
  scripts/import_owner_users.sh [export_dir]

Imports owner-role users exported by scripts/export_owner_users.sh.
By default, export_dir is the directory where this script is located.

Database:
  postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable

Behavior:
  - Upserts users by their original id.
  - Rebuilds role links for imported users based on role code in the target DB.
  - Recomputes t_user.roles from target role codes after import.
  - Upserts third-party login bindings by open_id + third_type.
  - Login tokens are not imported.

Important:
  Stop or restart the backend around import if it may have cached these users.
EOF
}

log() {
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] [import_owner_users] $*" >&2
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXPORT_DIR="${1:-$SCRIPT_DIR}"

for file in metadata.txt users.csv user_roles.csv third_users.csv; do
  if [ ! -f "$EXPORT_DIR/$file" ]; then
    echo "missing export file: $EXPORT_DIR/$file" >&2
    exit 1
  fi
done

DB_URL="postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable"

log "importing owner users from $EXPORT_DIR"

psql "$DB_URL" -v ON_ERROR_STOP=1 \
  -v users_csv="$EXPORT_DIR/users.csv" \
  -v user_roles_csv="$EXPORT_DIR/user_roles.csv" \
  -v third_users_csv="$EXPORT_DIR/third_users.csv" <<'SQL'
BEGIN;

CREATE TEMP TABLE tmp_owner_users (
  id bigint,
  type integer,
  phone text,
  username text,
  email text,
  email_verified boolean,
  nickname text,
  avatar text,
  gender text,
  birthday timestamptz,
  background_image text,
  password text,
  home_page text,
  description text,
  score integer,
  exp integer,
  level integer,
  status integer,
  topic_count integer,
  comment_count integer,
  follow_count integer,
  fans_count integer,
  roles text,
  forbidden_end_time bigint,
  create_time bigint,
  update_time bigint
) ON COMMIT DROP;

CREATE TEMP TABLE tmp_owner_user_roles (
  user_id bigint,
  role_code text,
  create_time bigint
) ON COMMIT DROP;

CREATE TEMP TABLE tmp_owner_third_users (
  id bigint,
  user_id bigint,
  open_id text,
  third_type text,
  nickname text,
  avatar text,
  extra_data text,
  create_time bigint,
  update_time bigint
) ON COMMIT DROP;

\copy tmp_owner_users FROM :'users_csv' WITH (FORMAT csv, HEADER true)
\copy tmp_owner_user_roles FROM :'user_roles_csv' WITH (FORMAT csv, HEADER true)
\copy tmp_owner_third_users FROM :'third_users_csv' WITH (FORMAT csv, HEADER true)

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM t_role WHERE code = 'owner') THEN
    RAISE EXCEPTION 'target database is missing role code owner; run installation/migrations first';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM tmp_owner_user_roles tur
    LEFT JOIN t_role r ON r.code = tur.role_code
    WHERE r.id IS NULL
  ) THEN
    RAISE EXCEPTION 'target database is missing one or more role codes from export: %',
      (
        SELECT string_agg(DISTINCT tur.role_code, ', ')
        FROM tmp_owner_user_roles tur
        LEFT JOIN t_role r ON r.code = tur.role_code
        WHERE r.id IS NULL
      );
  END IF;
END $$;

INSERT INTO t_user (
  id,
  type,
  phone,
  username,
  email,
  email_verified,
  nickname,
  avatar,
  gender,
  birthday,
  background_image,
  password,
  home_page,
  description,
  score,
  exp,
  level,
  status,
  topic_count,
  comment_count,
  follow_count,
  fans_count,
  roles,
  forbidden_end_time,
  create_time,
  update_time
)
SELECT
  id,
  type,
  NULLIF(phone, ''),
  NULLIF(username, ''),
  NULLIF(email, ''),
  email_verified,
  nickname,
  avatar,
  gender,
  birthday,
  background_image,
  password,
  home_page,
  description,
  score,
  exp,
  level,
  status,
  topic_count,
  comment_count,
  follow_count,
  fans_count,
  roles,
  forbidden_end_time,
  create_time,
  update_time
FROM tmp_owner_users
ON CONFLICT (id) DO UPDATE SET
  type = EXCLUDED.type,
  phone = EXCLUDED.phone,
  username = EXCLUDED.username,
  email = EXCLUDED.email,
  email_verified = EXCLUDED.email_verified,
  nickname = EXCLUDED.nickname,
  avatar = EXCLUDED.avatar,
  gender = EXCLUDED.gender,
  birthday = EXCLUDED.birthday,
  background_image = EXCLUDED.background_image,
  password = EXCLUDED.password,
  home_page = EXCLUDED.home_page,
  description = EXCLUDED.description,
  score = EXCLUDED.score,
  exp = EXCLUDED.exp,
  level = EXCLUDED.level,
  status = EXCLUDED.status,
  topic_count = EXCLUDED.topic_count,
  comment_count = EXCLUDED.comment_count,
  follow_count = EXCLUDED.follow_count,
  fans_count = EXCLUDED.fans_count,
  roles = EXCLUDED.roles,
  forbidden_end_time = EXCLUDED.forbidden_end_time,
  create_time = EXCLUDED.create_time,
  update_time = EXCLUDED.update_time;

DELETE FROM t_user_role ur
USING tmp_owner_users u
WHERE ur.user_id = u.id;

INSERT INTO t_user_role (user_id, role_id, create_time)
SELECT DISTINCT
  tur.user_id,
  r.id,
  COALESCE(tur.create_time, floor(extract(epoch FROM now()))::bigint)
FROM tmp_owner_user_roles tur
JOIN t_role r ON r.code = tur.role_code
ON CONFLICT (user_id, role_id) DO UPDATE SET
  create_time = EXCLUDED.create_time;

UPDATE t_user u
SET roles = role_codes.roles
FROM (
  SELECT
    ur.user_id,
    string_agg(r.code, ',' ORDER BY r.sort_no ASC, r.id DESC) AS roles
  FROM t_user_role ur
  JOIN t_role r ON r.id = ur.role_id
  JOIN tmp_owner_users ou ON ou.id = ur.user_id
  GROUP BY ur.user_id
) role_codes
WHERE u.id = role_codes.user_id;

INSERT INTO t_third_user (
  id,
  user_id,
  open_id,
  third_type,
  nickname,
  avatar,
  extra_data,
  create_time,
  update_time
)
SELECT
  id,
  user_id,
  open_id,
  third_type,
  nickname,
  avatar,
  extra_data,
  create_time,
  update_time
FROM tmp_owner_third_users
ON CONFLICT (open_id, third_type) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  nickname = EXCLUDED.nickname,
  avatar = EXCLUDED.avatar,
  extra_data = EXCLUDED.extra_data,
  create_time = EXCLUDED.create_time,
  update_time = EXCLUDED.update_time;

SELECT setval(pg_get_serial_sequence('t_user', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM t_user), 1), true);
SELECT setval(pg_get_serial_sequence('t_user_role', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM t_user_role), 1), true);
SELECT setval(pg_get_serial_sequence('t_third_user', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM t_third_user), 1), true);

SELECT
  count(DISTINCT u.id) AS imported_owner_users
FROM t_user u
JOIN t_user_role ur ON ur.user_id = u.id
JOIN t_role r ON r.id = ur.role_id
JOIN tmp_owner_users tu ON tu.id = u.id
WHERE r.code = 'owner';

COMMIT;
SQL

log "done"
