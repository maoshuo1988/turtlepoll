#!/usr/bin/env bash
set -euo pipefail

# 用法：
#   ADMIN_TOKEN=xxx ./scripts/backfill_pet_default_abilities.sh
#   ADMIN_TOKEN=xxx DRY_RUN=true PET_KEYS='["fortune"]' ./scripts/backfill_pet_default_abilities.sh

BASE_URL="${BASE_URL:-http://127.0.0.1:8082}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
DRY_RUN="${DRY_RUN:-false}"
PET_KEYS="${PET_KEYS:-[\"fortune\",\"lava\",\"lightning\",\"shell\"]}"

if [[ -z "$ADMIN_TOKEN" ]]; then
  echo "ADMIN_TOKEN is required"
  echo "example: ADMIN_TOKEN=xxx ./scripts/backfill_pet_default_abilities.sh"
  exit 1
fi

payload=$(cat <<JSON
{
  "petKeys": ${PET_KEYS},
  "dryRun": ${DRY_RUN}
}
JSON
)

echo "POST ${BASE_URL}/api/admin/pet/backfill-default-abilities"
echo "payload: ${payload}"

curl -sS -X POST "${BASE_URL}/api/admin/pet/backfill-default-abilities" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ADMIN_TOKEN}" \
  -d "${payload}"

echo
