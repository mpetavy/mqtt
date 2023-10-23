package main

import (
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mpetavy/common"
	"net/url"
	"time"
)

var (
	serverurl = flag.String("url", "", "URL for sending messages")
	clientId  = flag.String("clientid", "", "Client ID")
	topic     = flag.String("topic", "", "Topic")
	username  = flag.String("username", "", "Username")
	password  = flag.String("password", "", "Password")
	timeout   = flag.Int("timeout", 0, "timeout")
	qos       = flag.Int("qos", QOS_AT_MOST_ONCE, "timeout")
	text      = flag.String("text", "", "Payload")
	retained  = flag.Bool("retained", false, "Retained flag")
	count     = flag.Int("count", 1, "count")
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
	common.Info("Connected to %s", *serverurl)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	common.Warn("Connected lost to %s", *serverurl)
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

	_, err := url.Parse(*serverurl)
	if common.Error(err) {
		return err
	}

	isReceiver := *text == ""

	if isReceiver {
		common.Info("Receiver")
	} else {
		common.Info("Sender")

		if *text == "" {
			return fmt.Errorf("flag 'text' not defined")
		}
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(*serverurl)
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

	err = waitOnToken(*timeout, token)
	if common.Error(err) {
		return err
	}

	defer func() {
		time.Sleep(time.Second)
		client.Disconnect(1000)
	}()

	if isReceiver {
		token := client.Subscribe(*topic, byte(*qos), func(client mqtt.Client, message mqtt.Message) {
			common.Info("---------------------------------------------------------------------")
			common.Info("Message payload: %s", string(message.Payload()))
			common.Debug("Message internals: %+v", message)
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
			t = fmt.Sprintf("%s (count: #%d)", *text, i)
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
	common.Run([]string{"url", "clientid", "topic"})
}
