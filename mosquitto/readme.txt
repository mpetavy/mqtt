docker compose logs  -f

mqtt -connect "tcp://localhost:1883" -clientid sender -topic mqtt -publish "Hello world!" -count 5

https://mosquitto.org/man/mosquitto-conf-5.html

1883 : subscribe from-broker/mqtt
1884 : subscribe from-bridge/mqtt


mosquitto_pub -h localhost -p 1884 -m Hallo -t mqtt
mosquitto_sub -h localhost -t from-broker/mqtt