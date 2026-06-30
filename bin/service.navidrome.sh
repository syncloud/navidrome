#!/bin/bash -e

/bin/rm -f ${SNAP_DATA}/navidrome.sock

export ND_ADDRESS=unix:${SNAP_DATA}/navidrome.sock
export ND_UNIXSOCKETPERM=0660
export ND_DATAFOLDER=${SNAP_DATA}/data
export ND_CACHEFOLDER=${SNAP_DATA}/cache
export ND_MUSICFOLDER=/data/navidrome
export ND_EXTAUTH_USERHEADER=Remote-User
export ND_EXTAUTH_TRUSTEDSOURCES=@
export ND_LOGLEVEL=info
export ND_ENABLEINSIGHTSCOLLECTOR=false
export ND_SCANSCHEDULE=1h

exec ${SNAP}/navidrome/navidrome
