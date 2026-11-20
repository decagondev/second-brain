#!/usr/bin/env bash
# Puts one cell's project inside its own VPC Service Controls perimeter (sb-runtime(7), Data,
# exactly; SB-902). Run it as an Access Context Manager admin of the tenant's organisation,
# once per cell project, after bootstrap-deploy-role.sh.
#
#   deploy/gcp/bootstrap-perimeter.sh ACCESS_POLICY PROJECT_NUMBER TENANT
#
# Only the cell's own service accounts may call the restricted services from outside the
# perimeter; everything else, including a wrongly granted identity from another cell, is refused.
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 ACCESS_POLICY PROJECT_NUMBER TENANT" >&2
  exit 2
fi
policy="$1"
number="$2"
tenant="$3"
name="sb_cell_${tenant//-/_}"

# The services the perimeter covers; the GCP fake enforces the same list (gcptest.restricted).
services=(
  sqladmin.googleapis.com
  cloudkms.googleapis.com
  secretmanager.googleapis.com
  pubsub.googleapis.com
  storage.googleapis.com
)
joined="$(IFS=,; echo "${services[*]}")"

gcloud access-context-manager perimeters create "$name" --policy "$policy" \
  --title "Second Brain cell ${tenant}" --resources "projects/${number}" \
  --restricted-services "$joined" --perimeter-type regular

echo "perimeter ${name} covers projects/${number}"
