package main

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	//url      = "tcp://broker.hivemq.com:1883"
	url      = "tcp://localhost:1883"
	topic    = "mqtt"
	clientid = "mqtt"
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Topic: %s | %s\n", msg.Topic(), msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %+v", err)
}

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID(clientid)
	// opts.SetUsername("admin")
	// opts.SetPassword("instar")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	sub(client)

	publish(client)

	client.Disconnect(250)
}

func publish(client mqtt.Client) {
	for i := 0; i < 5; i++ {
		value := fmt.Sprintf("ping #%d", i)
		token := client.Publish(topic, 0, false, value)
		token.Wait()
		time.Sleep(time.Second)
	}
}

func sub(client mqtt.Client) {
	// Subscribe to the LWT connection status
	token := client.Subscribe(topic, 1, func(client mqtt.Client, message mqtt.Message) {
		fmt.Printf("%+v\n", string(message.Payload()))
	})
	token.Wait()
	fmt.Println("Subscribed to LWT", topic)
}
