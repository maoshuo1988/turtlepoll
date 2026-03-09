清表
``` bash
PGPASSWORD=root psql -h localhost -U appuser -d turtlepoll -Atc "SELECT 'DROP TABLE IF EXISTS \"' || tablename || '\" CASCADE;' FROM pg_tables WHERE schemaname = 'public';" | PGPASSWORD=root psql -h localhost -U appuser -d turtlepoll
```

构建
``` bash
make build
```