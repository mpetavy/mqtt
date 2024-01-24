docker compose logs  -f

mqtt -connect "tcp://localhost:1883" -clientid sender -topic mqtt -publish "Hello world!" -count 5

https://mosquitto.org/man/mosquitto-conf-5.html

1883 : subscribe from-broker/mqtt
1884 : subscribe from-bridge/mqtt
