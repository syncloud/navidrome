#!/bin/bash -e

export SSL_CERT_FILE=/var/snap/platform/current/syncloud.ca.crt

/bin/rm -f ${SNAP_DATA}/backend.sock

exec ${SNAP}/backend/backend
