#!/bin/zsh
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <cluster-id> [addon-key]" >&2
  exit 1
fi

CLUSTER_ID="$1"
ADDON_KEY="${2:-aws_load_balancer_controller}"

echo
echo "== cluster =="
docker compose exec -T mysql mysql -ubackend -pbackend123 backend_center -e "
select id,name,code,provider,environment,source_resource_id,status,updated_at
from clusters
where id=${CLUSTER_ID}\G"

echo
echo "== addon execution =="
docker compose exec -T mysql mysql -ubackend -pbackend123 backend_center -e "
select id,cluster_id,source_job_id,resource_id,provider,region,addon_key,addon_type,install_mode,release_name,namespace,status,worker_name,error_message,started_at,ended_at,last_success_at,updated_at
from cluster_addon_executions
where cluster_id=${CLUSTER_ID} and addon_key='${ADDON_KEY}'
order by id desc\G"

echo
echo "== addon contract =="
docker compose exec -T mysql mysql -ubackend -pbackend123 backend_center -N -B -e "
select contract_json
from cluster_addon_executions
where cluster_id=${CLUSTER_ID} and addon_key='${ADDON_KEY}'
order by id desc
limit 1"

echo
echo "== addon result =="
docker compose exec -T mysql mysql -ubackend -pbackend123 backend_center -N -B -e "
select result_json
from cluster_addon_executions
where cluster_id=${CLUSTER_ID} and addon_key='${ADDON_KEY}'
order by id desc
limit 1"
