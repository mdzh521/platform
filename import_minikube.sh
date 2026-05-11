#!/bin/zsh
set -euo pipefail

ROOT=/Users/alex/ops/platform-center
COMPOSE_CMD=(/usr/local/bin/docker compose -f "$ROOT/ops/compose/docker-compose.yml")

login() {
  /usr/bin/jq -nc '{username:"admin",password:"admin123456"}' \
    | "${COMPOSE_CMD[@]}" exec -T backend /bin/sh -lc \
      'cat >/tmp/login.json && wget -qO- --header="Content-Type: application/json" --post-file=/tmp/login.json http://127.0.0.1:8080/api/v1/auth/login/local'
}

api_post_json() {
  local token="$1"
  local path="$2"
  local body_file="$3"

  /bin/cat "$body_file" \
    | "${COMPOSE_CMD[@]}" exec -T backend /bin/sh -lc \
      "cat >/tmp/request.json && wget -qO- --header=\"Content-Type: application/json\" --header=\"Authorization: Bearer $token\" --post-file=/tmp/request.json http://127.0.0.1:8080$path"
}

api_post_empty() {
  local token="$1"
  local path="$2"

  "${COMPOSE_CMD[@]}" exec -T backend /bin/sh -lc \
    "wget -qO- --header=\"Authorization: Bearer $token\" --post-data=\"\" http://127.0.0.1:8080$path"
}

api_get() {
  local token="$1"
  local path="$2"

  "${COMPOSE_CMD[@]}" exec -T backend /bin/sh -lc \
    "wget -qO- --header=\"Authorization: Bearer $token\" http://127.0.0.1:8080$path"
}

api_delete() {
  local token="$1"
  local path="$2"

  "${COMPOSE_CMD[@]}" exec -T backend /bin/sh -lc \
    "wget -qO- --method=DELETE --header=\"Authorization: Bearer $token\" http://127.0.0.1:8080$path"
}

token="$(login | /usr/bin/jq -r .data.token)"

"${COMPOSE_CMD[@]}" exec -T mysql mysql -ubackend -pbackend123 -D backend_center -e \
  "delete from workloads where cluster_id in (1,2); delete from namespaces where cluster_id in (1,2); delete from clusters where id in (1,2);"

/usr/local/bin/kubectl config view --raw --flatten --minify \
  | /usr/bin/sed 's#https://127.0.0.1:#https://host.docker.internal:#' \
  | /usr/bin/sed '/server:/a\
    insecure-skip-tls-verify: true' \
  > /tmp/platform-center-minikube-kubeconfig.yaml

/usr/bin/jq -nc --rawfile kc /tmp/platform-center-minikube-kubeconfig.yaml '{
  name: "Local Minikube",
  code: "minikube-local",
  environment: "dev",
  provider: "minikube",
  api_endpoint: "https://host.docker.internal:54338",
  auth_type: "kubeconfig",
  credential: $kc,
  description: "Local minikube cluster from host"
}' > /tmp/platform-center-minikube-cluster.json

create_response="$(api_post_json "$token" "/api/v1/k8s/clusters" /tmp/platform-center-minikube-cluster.json)"
echo "$create_response"

cluster_id="$(echo "$create_response" | /usr/bin/jq -r .data.id)"

api_post_empty "$token" "/api/v1/k8s/clusters/$cluster_id/test"
api_post_empty "$token" "/api/v1/k8s/clusters/$cluster_id/sync-namespaces"
api_post_empty "$token" "/api/v1/k8s/clusters/$cluster_id/sync-workloads"
api_get "$token" "/api/v1/k8s/clusters"
