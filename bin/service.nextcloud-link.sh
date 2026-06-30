#!/bin/bash

while true; do
    ${SNAP}/bin/nextcloud-link.sh || true
    sleep 300
done
