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

说明：
- 同步会为每条赛程创建/更新对应的预测市场（`PredictMarket`）。
- 只有当赛程 `home_team` 和 `away_team` 都有值时，市场状态才会设置为 `OPEN`；否则为 `CLOSE`。
- 每次同步都会根据主队/客队刷新市场 `title`（如：`Home vs Away`）。

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

``` bash
curl 'http://turtle.cloud-ip.cc:8082/api/config/configs' \
  -H 'Accept: */*' \
  -H 'Accept-Language: zh-CN,zh;q=0.9' \
  -b 'bbsgo_token=2b08b8f830034454b8d9b62226e2d103' \
  -H 'Proxy-Connection: keep-alive' \
  -H 'Referer: http://turtle.cloud-ip.cc:8082/' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36' \
  --insecure
```

``` bash
curl 'https://turtle.cloud-ip.cc/api/config/configs' \
  -H 'Accept: */*' \
  -H 'Accept-Language: zh-CN,zh;q=0.9' \
  -b 'bbsgo_token=2b08b8f830034454b8d9b62226e2d103' \
  -H 'Proxy-Connection: keep-alive' \
  -H 'Referer: https://turtle.cloud-ip.cc/' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36' \
  -H 'Origin: https://main.d2vufo32ngwxrk.amplifyapp.com' \
  --insecure
```


