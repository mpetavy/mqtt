package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mpetavy/common"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	QOS_AT_MOST_ONCE  = 0
	QOS_AT_LEAST_ONCE = 1
	QOS_EXACTLY_ONCE  = 2
)

type Conn struct {
	Broker        string   `json:"Broker"`
	Username      string   `json:"Username"`
	Password      string   `json:"Password"`
	ClientId      string   `json:"Clientid"`
	Topic         string   `json:"Topic"`
	Subscriptions []string `json:"Subscriptions"`
	Qos           int      `json:"Qos"`

	client mqtt.Client
}

var (
	host          = flag.String("url", "", "host IP of MQTT broker")
	username      = flag.String("username", "", "Username")
	password      = flag.String("password", "", "Password")
	clientId      = flag.String("clientid", "", "Client ID")
	topic         = flag.String("topic", "", "Topic")
	subscriptions = flag.String("subscriptions", "", "Subscriptions")
	timeout       = flag.Int("timeout", 0, "timeout")
	qos           = flag.Int("qos", QOS_AT_MOST_ONCE, "timeout")
	text          = flag.String("text", "", "Payload")
	retained      = flag.Bool("retained", false, "Retained flag")
	count         = flag.Int("count", 1, "count")

	conns       = make(map[string]*Conn)
	currentConn *Conn
)

//go:embed go.mod
var resources embed.FS

func init() {
	common.Init("", "", "", "", "mqtt", "", "", "", &resources, nil, nil, run, 0)
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Topic: %s | %s\n", msg.Topic(), msg.Payload())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	common.Info("Connected!")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	common.Warn("Connection lost!")
}

var reconnectHandler mqtt.ReconnectHandler = func(client mqtt.Client, clientOptions *mqtt.ClientOptions) {
	common.Warn("Reconnected!")
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

func receive(client mqtt.Client, message mqtt.Message) {
	common.Info("Received [%s]: %s", message.Topic(), string(message.Payload()))
	common.Debug("Internals: %+v", message)
}

func defaults(cmds []string, args ...string) []string {
	var list []string

	for i, arg := range args {
		if i+1 < len(cmds) {
			arg = cmds[i+1]
		}

		list = append(list, arg)
	}

	return list
}

func (conn *Conn) setClientId(clientId string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	conn.ClientId = clientId

	return nil
}

func (conn *Conn) setTopic(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	conn.Topic = topic

	return nil
}

func (conn *Conn) setQos(q string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	v, err := strconv.Atoi(q)
	if common.Error(err) {
		return err
	}

	conn.Qos = v

	return nil
}

func connect(broker string, username string, password string) (*Conn, error) {
	conn, ok := conns[broker]
	if !ok {
		conn = &Conn{
			Broker:        broker,
			Username:      "",
			Password:      "",
			ClientId:      "",
			Subscriptions: nil,
			Qos:           QOS_AT_MOST_ONCE,
			client:        nil,
		}
	}

	u, err := url.Parse(broker)
	if common.Error(err) {
		return nil, err
	}

	if u.Port() == "" {
		broker = fmt.Sprintf("%s:1883", broker)
	}

	if !strings.Contains(broker, "tcp://") && !strings.Contains(broker, "ssl://") && !strings.Contains(broker, "ws://") {
		broker = fmt.Sprintf("tcp://%s", broker)
	}

	conn.Broker = broker

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(*clientId)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.OnReconnecting = reconnectHandler

	conn.client = mqtt.NewClient(opts)

	token := conn.client.Connect()
	err = waitOnToken(*timeout, token)
	if common.Error(err) {
		return nil, err
	}

	conns[broker] = conn

	return conn, nil
}

func isConnected(conn *Conn) error {
	if conn == nil || conn.client == nil {
		return fmt.Errorf("not connected")
	}

	return nil
}

func (conn *Conn) disconnect() error {
	if conn == nil || conn.client == nil {
		return nil
	}

	conn.client.Disconnect(1000)
	conn.client = nil

	return nil
}

func (conn *Conn) String() string {
	ba, _ := json.MarshalIndent(conn, "", "    ")

	return string(ba)
}

func (conn *Conn) info() error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	ba, _ := json.MarshalIndent(conn, "", "    ")

	common.Info(string(ba))

	return nil
}

func (conn *Conn) subscribe(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	if slices.Contains(conn.Subscriptions, topic) {
		common.Warn("already subscribed: %s", topic)

		return nil

	}

	token := conn.client.Subscribe(topic, byte(*qos), receive)

	err = waitOnToken(*timeout, token)
	if common.Error(err) {
		return err
	}

	conn.Subscriptions = append(conn.Subscriptions, topic)

	return nil
}

func (conn *Conn) unsubscribe(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	p := slices.Index(conn.Subscriptions, topic)

	if p == -1 {
		common.Warn("Not subscribed to: %s", topic)

		return nil
	}

	token := conn.client.Unsubscribe(topic)

	err = waitOnToken(*timeout, token)
	if common.Error(err) {
		return err
	}

	conn.Subscriptions = slices.Delete(conn.Subscriptions, p, p+1)

	return nil
}

func (conn *Conn) publish(text string, count int, retained bool) error {
	err := isConnected(conn)
	if common.Error(err) {
		return nil
	}

	for i := 0; i < count; i++ {
		token := conn.client.Publish(conn.Topic, byte(conn.Qos), retained, text)

		err := waitOnToken(*timeout, token)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func run() error {
	if *host != "" {
		var err error

		currentConn, err = connect(*host, *username, *password)
		if !common.Error(err) {
			if *subscriptions != "" {
				for _, subscription := range common.Split(*subscriptions, ";") {
					common.Error(currentConn.subscribe(subscription))
				}
			}

			if *clientId != "" {
				common.Error(currentConn.setClientId(*clientId))
			}

			if *topic != "" {
				common.Error(currentConn.setTopic(*topic))
			}

			if *qos != 0 {
				common.Error(currentConn.setQos(strconv.Itoa(*qos)))
			}

			if *text != "" {
				currentConn.publish(*text, *count, *retained)
			}
		}
	}

	defer func() {
		for _, conn := range conns {
			conn.disconnect()
		}
	}()

	for {
		fmt.Printf("> ")

		reader := bufio.NewReader(os.Stdin)
		cmdline, err := reader.ReadString('\n')
		common.DebugError(err)

		cmds := common.SplitCmdline(strings.TrimSpace(cmdline))

		if len(cmds) == 0 {
			continue
		}

		switch cmds[0] {
		case "exit":
			break
		case "info":
			common.Error(currentConn.info())
		case "qos":
			args := defaults(cmds, strconv.Itoa(QOS_AT_MOST_ONCE))
			common.Error(currentConn.setQos(args[0]))
		case "clientid":
			args := defaults(cmds, "")
			common.Error(currentConn.setClientId(args[0]))
		case "topic":
			args := defaults(cmds, "")
			common.Error(currentConn.setClientId(args[0]))
		case "connect":
			args := defaults(cmds, "localhost", "", "")
			connect(args[0], args[1], args[2])
		case "subscribe":
			args := defaults(cmds, "")
			common.Error(currentConn.subscribe(args[0]))
		case "unsubscribe":
			args := defaults(cmds, "")
			common.Error(currentConn.unsubscribe(args[0]))
		case "disconnect":
			currentConn.disconnect()
		case "publish":
			var args []string

			switch len(cmds) {
			case 2:
				args = defaults(cmds, cmds[1], "1", "false")
			case 3:
				args = defaults(cmds, cmds[1], cmds[2], "false")
			case 4:
				args = defaults(cmds, cmds[1], cmds[2], cmds[3])
			default:
				common.Error(fmt.Errorf("invalid command: %s", cmds[0]))

				continue
			}

			c, err := strconv.Atoi(args[1])
			if common.Error(err) {
				continue
			}

			b := common.ToBool(args[2])

			currentConn.publish(args[0], c, b)
		default:
			common.Error(fmt.Errorf("unknown command: %s", cmds[0]))
		}
	}

	return nil
}

func main() {
	common.Run([]string{"url", "clientid", "topic"})
}
