#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

PROJECT="${1:-desktop}"
NAME=navidrome
export PLAYWRIGHT_DOMAIN="${PLAYWRIGHT_DOMAIN:-bookworm.com}"
export PLAYWRIGHT_USER="${PLAYWRIGHT_USER:-user}"
export PLAYWRIGHT_PASSWORD="${PLAYWRIGHT_PASSWORD:-Password1}"

DOMAIN="$PLAYWRIGHT_DOMAIN"
APP_DOMAIN="${NAME}.${DOMAIN}"
SSH_PASS="${PLAYWRIGHT_SSH_PASSWORD:-$PLAYWRIGHT_PASSWORD}"
export PLAYWRIGHT_ARTIFACT_DIR="$( cd "${DIR}/../.." && pwd )/artifact"
mkdir -p "${PLAYWRIGHT_ARTIFACT_DIR}"

getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | tee -a /etc/hosts
cat /etc/hosts

apt-get update -qq
apt-get install -y -qq sshpass openssh-client

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=15"

# Navidrome has no upload UI: music is added by placing files in the library folder.
# Put a sample track there (the real way users add music) and force a rescan.
sshpass -p "$SSH_PASS" ssh $SSH_OPTS root@${APP_DOMAIN} "mkdir -p /data/navidrome/sample"
sshpass -p "$SSH_PASS" scp $SSH_OPTS "${DIR}/fixtures/sample.mp3" "root@${APP_DOMAIN}:/data/navidrome/sample/song.mp3"
sshpass -p "$SSH_PASS" ssh $SSH_OPTS root@${APP_DOMAIN} "chown -R navidrome /data/navidrome && snap restart navidrome.navidrome"
sshpass -p "$SSH_PASS" ssh $SSH_OPTS root@${APP_DOMAIN} "for i in \$(seq 1 60); do test -S /var/snap/navidrome/current/navidrome.sock && exit 0; sleep 2; done; exit 1"

cd ${DIR}
npm install --no-audit --no-fund
npx playwright test --project="${PROJECT}"
