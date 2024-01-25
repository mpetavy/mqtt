docker compose logs  -f

mqtt -connect "tcp://localhost:1883" -clientid sender -topic mqtt -publish "Hello world!" -count 5

https://mosquitto.org/man/mosquitto-conf-5.html

on 1883 : subscribe from-broker/mqtt
on 1884 : subscribe from-bridge/mqtt

docker compose stop broker
docker compose start broker

docker compose up -d --force-recreate --scale bridge=3

mosquitto_pub -h localhost -p 1884 -t mqtt -m Hallo
mosquitto_sub -h localhost -t from-bridge/mqtt
