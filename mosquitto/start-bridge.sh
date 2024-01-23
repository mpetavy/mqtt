@echo off

call stop-bridge.bat
docker compose up -d --force-recreate
