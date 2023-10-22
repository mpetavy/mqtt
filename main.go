package main

import (
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mpetavy/common"
	"time"
)

var (
	client   = flag.String("c", "", "URL for sending messages")
	server   = flag.String("s", "", "URL for receiving messages")
	clientId = flag.String("clientid", "", "Client ID")
	topic    = flag.String("topic", common.Title(), "Topic")
	username = flag.String("username", "", "Username")
	password = flag.String("password", "", "Password")
	timeout  = flag.Int("timeout", 0, "timeout")
	qos      = flag.Int("qos", QOS_AT_MOST_ONCE, "timeout")
	text     = flag.String("text", "Hello world!", "Payload")
	retained = flag.Bool("retained", false, "Retained flag")
	count    = flag.Int("count", 1, "count")

	url string
)

const (
	QOS_AT_MOST_ONCE  = 0
	QOS_AT_LEAST_ONCE = 1
	QOS_EXACTLY_ONCE  = 2
)

func init() {
	common.Init("mqtt", "", "", "", "2023", "mqtt", "mpetavy", fmt.Sprintf("https://github.com/mpetavy/%s", common.Title()), common.APACHE, nil, nil, nil, run, 0)
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Topic: %s | %s\n", msg.Topic(), msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	common.Info("Connected to %s", url)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	common.Warn("Connected lost to %s", url)
}

func waitOnToken(timeout int, token mqtt.Token) error {
	var b bool

	if timeout > 0 {
		b = token.WaitTimeout(common.MillisecondToDuration(timeout))
	} else {
		b = token.Wait()
	}

	var err error

	if !b {
		err = token.Error()

		if err == nil {
			err = fmt.Errorf("token timeout")
		}
	}

	if common.Error(err) {
		return err
	}

	return nil
}

func run() error {
	flag.Visit(func(f *flag.Flag) {
		common.Info("Flag %s: \t%s", f.Name, f.Value)
	})

	isServer := *server != ""

	if isServer {
		url = *server
	} else {
		url = *client
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID(*clientId)
	if *username != "" {
		opts.SetUsername(*username)
		opts.SetPassword(*password)
	}
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)

	token := client.Connect()

	err := waitOnToken(*timeout, token)
	if common.Error(err) {
		return err
	}

	defer func() {
		time.Sleep(time.Second)
		client.Disconnect(1000)
	}()

	if isServer {
		token := client.Subscribe(*topic, byte(*qos), func(client mqtt.Client, message mqtt.Message) {
			common.Info("----------------------")
			common.Info("Message received: %+v", message)
			common.Info("Message payload: %s", string(message.Payload()))
		})

		err := waitOnToken(*timeout, token)
		if common.Error(err) {
			return err
		}

		<-common.AppLifecycle().NewChannel()

		return nil
	}

	for i := 0; i < *count; i++ {
		t := *text
		if *count > 1 {
			t = fmt.Sprintf(" %s: %s #%d", time.Now().Format(time.DateTime), *text, i)
		}

		common.Info("Send text: %s", t)

		token = client.Publish(*topic, byte(*qos), *retained, t)

		err = waitOnToken(*timeout, token)
		if common.Error(err) {
			return err
		}

		if *count > 1 {
			time.Sleep(time.Second)
		}
	}

	return nil
}

func main() {
	common.Run([]string{"c|s", "clientid"})
}
