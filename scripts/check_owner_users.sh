psql "postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable" <<'SQL'
SELECT
  u.id,
  u.username,
  u.email,
  u.nickname,
  u.status,
  u.roles
FROM t_users u
JOIN t_user_roles ur ON ur.user_id = u.id
JOIN t_roles r ON r.id = ur.role_id
WHERE r.code = 'owner'
ORDER BY u.id;

SELECT
  u.id AS user_id,
  u.username,
  string_agg(r.code, ',' ORDER BY r.sort_no ASC, r.id DESC) AS role_codes
FROM t_users u
JOIN t_user_roles ur ON ur.user_id = u.id
JOIN t_roles r ON r.id = ur.role_id
WHERE u.id IN (
  SELECT ur2.user_id
  FROM t_user_roles ur2
  JOIN t_roles r2 ON r2.id = ur2.role_id
  WHERE r2.code = 'owner'
)
GROUP BY u.id, u.username
ORDER BY u.id;
SQL