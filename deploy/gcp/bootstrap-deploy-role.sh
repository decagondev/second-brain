#!/usr/bin/env bash
# Creates the least-privilege deploy role the factory uses to stamp a Second Brain cell
# into this GCP project (sb-tenancy(7), Placement on GCP; SB-703). Run it as a project
# owner of the TENANT's project, once, before a hand-off or a tenant-account placement.
#
#   deploy/gcp/bootstrap-deploy-role.sh PROJECT FACTORY_PRINCIPAL
#
# FACTORY_PRINCIPAL is the factory's identity that may impersonate the deploy account,
# e.g. serviceAccount:sb-factory@FACTORY_PROJECT.iam.gserviceaccount.com. It changes no
# other IAM binding, and it prints the deploy account's email for the hand-off ticket.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 PROJECT FACTORY_PRINCIPAL" >&2
  exit 2
fi
project="$1"
factory="$2"
role="sbCellDeployer"
account="sb-cell-deployer"
email="${account}@${project}.iam.gserviceaccount.com"

# Exactly the permissions the gcp adapter uses (gcp.DeployPermissions; a test compares
# this list with the code). Keep one permission per line.
permissions=(
  cloudkms.cryptoKeyVersions.useToDecrypt
  cloudkms.cryptoKeyVersions.useToEncrypt
  cloudkms.cryptoKeys.create
  cloudkms.cryptoKeys.get
  cloudkms.keyRings.create
  cloudkms.keyRings.get
  cloudsql.databases.create
  cloudsql.databases.get
  cloudsql.instances.create
  cloudsql.instances.get
  pubsub.topics.create
  pubsub.topics.get
  secretmanager.secrets.create
  secretmanager.secrets.get
)
joined="$(IFS=,; echo "${permissions[*]}")"

gcloud services enable sqladmin.googleapis.com cloudkms.googleapis.com \
  secretmanager.googleapis.com pubsub.googleapis.com --project "$project"

if gcloud iam roles describe "$role" --project "$project" >/dev/null 2>&1; then
  gcloud iam roles update "$role" --project "$project" --permissions "$joined" --quiet
else
  gcloud iam roles create "$role" --project "$project" --title "Second Brain cell deployer" \
    --description "Stamps one Second Brain cell; no IAM changes, no secret values" --permissions "$joined"
fi

if ! gcloud iam service-accounts describe "$email" --project "$project" >/dev/null 2>&1; then
  gcloud iam service-accounts create "$account" --project "$project" --display-name "Second Brain cell deployer"
fi

gcloud projects add-iam-policy-binding "$project" --member "serviceAccount:${email}" \
  --role "projects/${project}/roles/${role}" --condition None --quiet >/dev/null

# Only the factory's principal may impersonate the deploy account. A hand-off removes this
# binding unless the hand-off record retains it (sb-factory(7)).
gcloud iam service-accounts add-iam-policy-binding "$email" --project "$project" \
  --member "$factory" --role roles/iam.serviceAccountTokenCreator --quiet >/dev/null

echo "$email"
