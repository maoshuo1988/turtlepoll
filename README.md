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
scp -i /opt/pem/bbs.pem ubuntu@52.77.212.173:/srv/project/turtlepoll/bbs-go.yaml ./
```
## 上传
``` bash
scp -i /opt/pem/bbs.pem bbs-go-linux ubuntu@<SERVER_IP>:/tmp/
scp -i /opt/pem/bbs.pem bbs-go-linux ubuntu@52.77.212.173:/tmp/
```
## 移动文件
``` bash
sudo mv /tmp/bbs-go-linux /srv/project/turtlepoll/
cd /srv/project/turtlepoll
sudo chmod +x /srv/project/turtlepoll/bbs-go-linux
sudo chown root:root /srv/project/turtlepoll/bbs-go-linux

sudo bash -c 'mv /tmp/bbs-go-linux /srv/project/turtlepoll/ \
  && cd /srv/project/turtlepoll \
  && chmod +x bbs-go-linux \
  && chown root:root bbs-go-linux'
```

# 同步赛程
``` bash
curl -X POST "http://localhost:8082/api/football/sync_worldcup" --cookie "token=<YOUR_TOKEN>"
```

# 同步 Polymarket（只读）

说明：
- 只同步配置指定范围（tags / marketSlugs）
- 不同步价格盘口
- 同步市场状态与最终结算结果（resolved outcome）

```bash
curl -X POST "http://localhost:8082/api/football/sync_polymarket" --cookie "token=<YOUR_TOKEN>"
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


# 赌局更新
``` bash
curl -X POST -m 15 -sS -w "\nHTTP:%{http_code} time:%{time_total}\n" \
  "http://localhost:8082/api/admin/battle/cron_tick" \
  -b "bbsgo_token=34b04557201242e8bdf7a7615f2cc6e0" \
  -H "Accept: application/json"
```

``` bash
ps aux | grep go | grep -v grep
```

```
sudo BBSGO_ENV=prod nohup ./bbs-go-linux
BBSGO_ENV=prod go run ./main.go -c ./bbs-go.yaml
```