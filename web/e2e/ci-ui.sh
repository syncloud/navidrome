#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

PROJECT="${1:-desktop}"
NAME=navidrome
export PLAYWRIGHT_DOMAIN="${PLAYWRIGHT_DOMAIN:-bookworm.com}"
export PLAYWRIGHT_USER="${PLAYWRIGHT_USER:-user}"
export PLAYWRIGHT_PASSWORD="${PLAYWRIGHT_PASSWORD:-Password1}"

DOMAIN="$PLAYWRIGHT_DOMAIN"
APP_DOMAIN="${NAME}.${DOMAIN}"
getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | tee -a /etc/hosts
cat /etc/hosts

cd ${DIR}
npm install --no-audit --no-fund
npx playwright test --project="${PROJECT}"
