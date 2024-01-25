#!/bin/sh

/bin/sh stop-bridge.sh
docker compose up -d --force-recreate --remove-orphans
