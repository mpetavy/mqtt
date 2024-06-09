#!/bin/sh

mosquitto_passwd -c -b bridge/bridge.passwd bridgeuser bridgepwd
chmod 0700 /home/ransom/go/src/mqtt/mosquitto/bridge/bridge.passwd

mosquitto_passwd -c -b broker/broker.passwd brokeruser brokerpwd
chmod 0700 /home/ransom/go/src/mqtt/mosquitto/broker/broker.passwd