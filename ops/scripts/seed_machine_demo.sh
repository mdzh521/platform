#!/bin/zsh
set -euo pipefail

TOKEN=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login/local \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123456"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

AUTH="Authorization: Bearer $TOKEN"

create_group() {
  local name="$1"
  local code="$2"
  local policy="$3"
  local desc="$4"
  local existing
  local payload

  existing=$(curl -s http://127.0.0.1:8080/api/v1/machines/groups -H "$AUTH" | grep -c "\"name\":\"$name\"" || true)
  if [ "$existing" = "0" ]; then
    payload=$(printf '{"name":"%s","code":"%s","default_login_policy":"%s","description":"%s"}' "$name" "$code" "$policy" "$desc")
    curl -s -X POST http://127.0.0.1:8080/api/v1/machines/groups \
      -H "$AUTH" \
      -H "Content-Type: application/json" \
      -d "$payload" >/dev/null
  fi
}

create_asset() {
  local name="$1"
  local address="$2"
  local group="$3"
  local policy="$4"
  local port="$5"
  local account="$6"
  local status="$7"
  local tags="$8"
  local desc="$9"
  local existing
  local payload

  existing=$(curl -s http://127.0.0.1:8080/api/v1/machines/assets -H "$AUTH" | grep -c "\"name\":\"$name\"" || true)
  if [ "$existing" = "0" ]; then
    payload=$(printf '{"name":"%s","address":"%s","platform":"linux","protocol":"ssh","group_name":"%s","login_policy":"%s","port":%s,"account":"%s","status":"%s","tags":"%s","description":"%s"}' \
      "$name" "$address" "$group" "$policy" "$port" "$account" "$status" "$tags" "$desc")
    curl -s -X POST http://127.0.0.1:8080/api/v1/machines/assets \
      -H "$AUTH" \
      -H "Content-Type: application/json" \
      -d "$payload" >/dev/null
  fi
}

create_group "生产环境" "prod" "managed_first" "生产机器，优先使用托管账号登录"
create_group "测试环境" "staging" "manual_only" "测试和联调用机器"
create_group "数据库" "database" "managed_only" "数据库和关键中间件"
create_group "网络设备" "network" "manual_only" "网络与边界设备"

create_asset "prod-app-01" "10.10.1.21" "生产环境" "managed_first" "22" "deploy" "online" "prod,app,web" "生产应用节点，承载核心服务"
create_asset "prod-app-02" "10.10.1.22" "生产环境" "managed_first" "22" "deploy" "warning" "prod,app,api" "生产应用节点，近期有连接告警"
create_asset "staging-api-01" "10.20.3.15" "测试环境" "manual_only" "22" "tester" "online" "staging,api" "测试环境 API 节点"
create_asset "staging-job-01" "10.20.3.41" "测试环境" "manual_only" "22" "runner" "offline" "staging,batch" "测试批处理节点，当前离线"
create_asset "mysql-core-01" "10.30.0.12" "数据库" "managed_only" "22" "dbadmin" "online" "db,mysql,core" "核心 MySQL 主库"
create_asset "redis-cache-01" "10.30.0.26" "数据库" "managed_only" "22" "cacheops" "warning" "redis,cache" "缓存节点，需关注"
create_asset "fw-gateway-01" "10.40.0.1" "网络设备" "manual_only" "22" "netops" "online" "network,firewall" "边界网关设备"

echo "GROUPS"
curl -s http://127.0.0.1:8080/api/v1/machines/groups -H "$AUTH"
echo
echo "ASSETS"
curl -s http://127.0.0.1:8080/api/v1/machines/assets -H "$AUTH"
