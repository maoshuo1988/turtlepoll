# 清表
``` bash
PGPASSWORD=root psql -h localhost -U appuser -d turtlepoll -Atc "SELECT 'DROP TABLE IF EXISTS \"' || tablename || '\" CASCADE;' FROM pg_tables WHERE schemaname = 'public';" | PGPASSWORD=root psql -h localhost -U appuser -d turtlepoll
```

# 构建
``` bash
make buildlinux
```

# 部署
## 下载
``` bash
scp -i /opt/pem/bbs.pem ubuntu@<SERVER_IP>:/srv/project/turtlepoll/bbs-go.yaml ./
```
## 上传
``` bash
scp -i /opt/pem/bbs.pem bbs-go-linux ubuntu@<SERVER_IP>:/tmp/
```
## 移动文件
``` bash
sudo mv /tmp/bbs-go-linux /srv/project/turtlepoll/
```

# 同步赛程
``` bash
curl -X POST "http://localhost:8082/api/football/sync_worldcup" --cookie "token=<YOUR_TOKEN>"
```

``` bash
psql "postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable" -c "SELECT * FROM t_user_token;"
psql "postgres://appuser:root@127.0.0.1:5432/turtlepoll?sslmode=disable" -c "SELECT * FROM t_predict_market"
```
# 铸币
```bash
curl -X POST "http://localhost:8082/api/admin/coin/mint" \
  -H "Authorization: Bearer cc1396f0a58c412eaef71306259adc7c" \
  -d "userId=1" \
  -d "amount=80000" \
  -d "remark=self mint"
```

# 查询余额
```bash
curl -X GET "http://localhost:8082/api/coin/me" \
  -H "Authorization: Bearer cc1396f0a58c412eaef71306259adc7c"
```

# 上传图片

接口：`POST /api/upload`（multipart 表单字段名固定为 `image`，认证通常使用 cookie `bbsgo_token`）

```bash
curl -X POST "http://localhost:8082/api/upload" \
  -b "bbsgo_token=<YOUR_TOKEN>" \
  -F "image=@/absolute/path/to/image.png;type=image/png"
```