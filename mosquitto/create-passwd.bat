@echo off

mosquitto_passwd -c bridge\bridge.passwd bridgeuser

mosquitto_passwd -c broker\broker.passwd brokeruser
