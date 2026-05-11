#!/bin/zsh

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <source_job_id> [cloud_resource_id]" >&2
  exit 1
fi

job_id="$1"
resource_id="${2:-}"

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mysql_query() {
  local sql="$1"
  docker compose exec -T mysql /bin/sh -lc "mysql -N -B -ubackend -pbackend123 backend_center -e \"$sql\""
}

print_query_or_note() {
  local sql="$1"
  local empty_message="$2"
  local output
  output="$(mysql_query "$sql")"
  if [[ -z "${output//[$'\t\r\n ']}" ]]; then
    echo "$empty_message"
    return
  fi
  printf '%s\n' "$output"
}

column_exists() {
  local table_name="$1"
  local column_name="$2"
  local count
  count="$(mysql_query "select count(*) from information_schema.columns where table_schema = 'backend_center' and table_name = '${table_name}' and column_name = '${column_name}';" | tr -d '[:space:]')"
  [[ "$count" != "0" ]]
}

print_section() {
  local title="$1"
  echo
  echo "== ${title} =="
}

print_section "runtime"
echo "worker location: host process or docker compose worker container"
echo "evidence source: mysql container backend_center database"

print_section "source job"
print_query_or_note \
  "select id, name, provider, status, action, resource_sync_status, created_at, ended_at from deployment_jobs where id = ${job_id};" \
  "no deployment job found for id=${job_id}"

resource_columns="id, provider, category, resource_type, resource_name, cloud_id, lifecycle_state, source_job_id, region"
if column_exists "cloud_resources" "cluster_enrollment_status"; then
  resource_columns="${resource_columns}, cluster_enrollment_status, cluster_enrollment_worker, cluster_enrollment_error, cluster_enrolled_at"
fi

print_section "cloud resources"
if [[ -n "$resource_id" ]]; then
  print_query_or_note \
    "select ${resource_columns} from cloud_resources where id = ${resource_id};" \
    "no cloud resource found for id=${resource_id}"
else
  print_query_or_note \
    "select ${resource_columns} from cloud_resources where source_job_id = ${job_id} order by id desc;" \
    "no cloud resources found for source_job_id=${job_id}"
fi

if column_exists "cloud_resources" "cluster_enrollment_status"; then
  print_section "cluster-enrollment claim/result proxy"
  if [[ -n "$resource_id" ]]; then
    print_query_or_note \
      "select id as resource_id, provider, region, resource_type, resource_name, cloud_id, cluster_enrollment_status, cluster_enrollment_worker, cluster_enrollment_error, cluster_enrolled_at from cloud_resources where id = ${resource_id};" \
      "no cluster enrollment proxy record found for resource_id=${resource_id}"
  else
    print_query_or_note \
      "select id as resource_id, provider, region, resource_type, resource_name, cloud_id, cluster_enrollment_status, cluster_enrollment_worker, cluster_enrollment_error, cluster_enrolled_at from cloud_resources where source_job_id = ${job_id} and (resource_type like '%eks%' or resource_name like '%eks%' or resource_type like '%cluster%' or resource_name like '%cluster%') order by id desc;" \
      "no cluster enrollment proxy rows found for source_job_id=${job_id}"
  fi
else
  echo "cloud_resources table in current runtime is missing cluster enrollment columns."
  echo "this runtime cannot produce a full phase4 claim/result evidence chain until backend migrations are applied."
fi

print_section "final managed cluster record"
if column_exists "clusters" "source_resource_id"; then
  if [[ -n "$resource_id" ]]; then
    print_query_or_note \
      "select id, name, code, environment, provider, status, api_endpoint, source_resource_id, created_at from clusters where source_resource_id in (select cloud_id from cloud_resources where id = ${resource_id});" \
      "no managed cluster record linked to resource_id=${resource_id}"
  else
    print_query_or_note \
      "select id, name, code, environment, provider, status, api_endpoint, source_resource_id, created_at from clusters where source_resource_id in (select cloud_id from cloud_resources where source_job_id = ${job_id});" \
      "no managed cluster record linked to source_job_id=${job_id}"
  fi
else
  echo "clusters table in current runtime is missing source_resource_id."
  echo "this runtime is still on an older schema and cannot prove the final managed record linkage for phase4."
fi
