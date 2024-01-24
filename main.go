package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
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

	CMD_EXIT        = "exit"
	CMD_INFO        = "info"
	CMD_INFOS       = "infos"
	CMD_QOS         = "qos"
	CMD_FILE        = "file"
	CMD_TIMEOUT     = "timeout"
	CMD_CLIENTID    = "clientid"
	CMD_TOPIC       = "topic"
	CMD_INSPECT     = "inspect"
	CMD_CONNECT     = "connect"
	CMD_SUBSCRIBE   = "subscribe"
	CMD_UNSUBSCRIBE = "unsubscribe"
	CMD_DISCONNECT  = "disconnect"
	CMD_PUBLISH     = "publish"
)

type Conn struct {
	Broker        string   `json:"Broker"`
	ClientId      string   `json:"Clientid"`
	Topic         string   `json:"Topic"`
	Subscriptions []string `json:"Subscriptions"`
	Qos           int      `json:"Qos"`
	Timeout       int      `json:"Timeout"`

	client mqtt.Client
}

var (
	host      = flag.String(CMD_CONNECT, "", "host IP of MQTT broker")
	username  = flag.String("username", "", "Username")
	password  = flag.String("password", "", "Password")
	clientId  = flag.String(CMD_CLIENTID, uuid.New().String(), "Client ID")
	topic     = flag.String(CMD_TOPIC, "", "Topic")
	subscribe = flag.String(CMD_SUBSCRIBE, "", "Subscriptions")
	timeout   = flag.Int(CMD_TIMEOUT, 3000, "timeout")
	qos       = flag.Int(CMD_QOS, QOS_AT_MOST_ONCE, "timeout")
	publish   = flag.String(CMD_PUBLISH, "", "Payload")
	file      = flag.String(CMD_FILE, "", "File to execute")
	retained  = flag.Bool("retained", false, "Retained flag")
	count     = flag.Int("count", 1, "count")

	conns       = make(map[string]*Conn)
	currentConn *Conn
	inPrompt    bool
	connected   chan struct{}
	lastMsg     mqtt.Message
)

//go:embed go.mod
var resources embed.FS

func init() {
	common.Init("", "", "", "", "mqtt", "", "", "", &resources, nil, nil, run, 0)
}

var unsubscribedMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	prompt("-- [MQTT Event] Unsubscribed message received. Topic: %s | %s", msg.Topic(), msg.Payload())

	lastMsg = msg
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	prompt("-- [MQTT Event] Connected!")

	close(connected)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	prompt("-- [MQTT Event] Connection lost!")
}

var reconnectHandler mqtt.ReconnectHandler = func(client mqtt.Client, clientOptions *mqtt.ClientOptions) {
	prompt("-- [MQTT Event] Reconnected!")
}

func receive(client mqtt.Client, msg mqtt.Message) {
	prompt("-- [MQTT Event] Received [%s]: %s", msg.Topic(), string(msg.Payload()))

	lastMsg = msg
}

func inspect() error {
	if lastMsg == nil {
		return nil
	}

	ba, err := json.MarshalIndent(lastMsg, "", "    ")
	if common.Error(err) {
		return err
	}

	prompt("%s", string(ba))

	return nil
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
		return err
	}

	conn.ClientId = clientId

	return nil
}

func (conn *Conn) setTopic(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	conn.Topic = topic

	return nil
}

func (conn *Conn) setQos(q string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	v, err := strconv.Atoi(q)
	if common.Error(err) {
		return err
	}

	conn.Qos = v

	return nil
}

func (conn *Conn) setTimeout(q string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	v, err := strconv.Atoi(q)
	if common.Error(err) {
		return err
	}

	conn.Timeout = v

	return nil
}

func connect(broker string, username string, password string) error {
	conn, ok := conns[broker]
	if !ok {
		conn = &Conn{
			Broker:        broker,
			ClientId:      *clientId,
			Topic:         *topic,
			Subscriptions: nil,
			Qos:           *qos,
			Timeout:       *timeout,
			client:        nil,
		}
	}

	u, err := url.Parse(broker)
	if common.Error(err) {
		return err
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
	opts.SetDefaultPublishHandler(unsubscribedMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.OnReconnecting = reconnectHandler

	conn.client = mqtt.NewClient(opts)

	connected = make(chan struct{})

	token := conn.client.Connect()
	err = waitOnToken(*timeout, token)
	if common.Error(err) {
		return err
	}

	<-connected

	conns[broker] = conn

	currentConn = conn

	return nil
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
		return err
	}

	ba, _ := json.MarshalIndent(conn, "", "    ")

	prompt(string(ba))

	return nil
}

func (conn *Conn) infos() error {
	ba, _ := json.MarshalIndent(conns, "", "    ")

	prompt(string(ba))

	return nil
}

func (conn *Conn) subscribe(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	if slices.Contains(conn.Subscriptions, topic) {
		common.Warn("already subscribed: %s", topic)

		return nil

	}

	token := conn.client.Subscribe(topic, byte(*qos), receive)

	err = waitOnToken(conn.Timeout, token)
	if common.Error(err) {
		return err
	}

	conn.Subscriptions = append(conn.Subscriptions, topic)

	return nil
}

func (conn *Conn) unsubscribe(topic string) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	p := slices.Index(conn.Subscriptions, topic)

	if p == -1 {
		common.Warn("Not subscribed to: %s", topic)

		return nil
	}

	token := conn.client.Unsubscribe(topic)

	err = waitOnToken(conn.Timeout, token)
	if common.Error(err) {
		return err
	}

	conn.Subscriptions = slices.Delete(conn.Subscriptions, p, p+1)

	return nil
}

func (conn *Conn) publish(text string, count int, retained bool) error {
	err := isConnected(conn)
	if common.Error(err) {
		return err
	}

	if conn.Topic == "" {
		return fmt.Errorf("undefined topic")
	}

	for i := 0; i < count; i++ {
		msg := text
		if count > 1 {
			msg = fmt.Sprintf("%s [%d]", text, i)
		}

		token := conn.client.Publish(conn.Topic, byte(conn.Qos), retained, msg)

		err := waitOnToken(conn.Timeout, token)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func executeFile(filename string) error {
	ba, err := os.ReadFile(filename)
	if common.Error(err) {
		return err
	}

	err = executeScript(ba)
	if common.Error(err) {
		return err
	}

	return err
}

func executeScript(script []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(script))
	for scanner.Scan() {
		line := scanner.Text()

		prompt("%s", line)

		err := executeLine(line)
		if common.Error(err) {
			return err
		}
	}

	return nil
}

func promptNewLine() {
	if inPrompt {
		fmt.Println()

		inPrompt = false
	}
}

func prompt(format string, args ...any) {
	promptNewLine()

	if format != "" {
		fmt.Printf(format, args...)
		fmt.Println()
	} else {
		fmt.Printf("> ")

		inPrompt = true
	}
}

func executeLine(cmdline string) error {
	cmds := common.SplitCmdline(strings.TrimSpace(cmdline))

	if len(cmds) == 0 {
		return nil
	}

	switch cmds[0] {
	case CMD_EXIT:
		return &common.ErrExit{}
	case CMD_INFO:
		common.Error(currentConn.info())
	case CMD_INFOS:
		common.Error(currentConn.infos())
	case CMD_QOS:
		args := defaults(cmds, strconv.Itoa(*qos))
		common.Error(currentConn.setQos(args[0]))
	case CMD_TIMEOUT:
		args := defaults(cmds, strconv.Itoa(*timeout))
		common.Error(currentConn.setTimeout(args[0]))
	case CMD_CLIENTID:
		args := defaults(cmds, "")
		common.Error(currentConn.setClientId(args[0]))
	case CMD_TOPIC:
		args := defaults(cmds, "")
		common.Error(currentConn.setTopic(args[0]))
	case CMD_INSPECT:
		common.Error(inspect())
	case CMD_CONNECT:
		args := defaults(cmds, "localhost", "", "")
		common.Error(connect(args[0], args[1], args[2]))
	case CMD_FILE:
		args := defaults(cmds, "")
		common.Error(executeFile(args[0]))
	case CMD_SUBSCRIBE:
		args := defaults(cmds, "")
		common.Error(currentConn.subscribe(args[0]))
	case CMD_UNSUBSCRIBE:
		args := defaults(cmds, "")
		common.Error(currentConn.unsubscribe(args[0]))
	case CMD_DISCONNECT:
		currentConn.disconnect()
	case CMD_PUBLISH:
		var args []string

		switch len(cmds) {
		case 2:
			args = defaults(cmds, cmds[1], "1", "false")
		case 3:
			args = defaults(cmds, cmds[1], cmds[2], "false")
		case 4:
			args = defaults(cmds, cmds[1], cmds[2], cmds[3])
		default:
			return fmt.Errorf("invalid command: %s", cmds[0])
		}

		c, err := strconv.Atoi(args[1])
		if common.Error(err) {
			return err
		}

		b := common.ToBool(args[2])

		common.Error(currentConn.publish(args[0], c, b))
	default:
		common.Error(fmt.Errorf("unknown command: %s", cmds[0]))
	}

	return nil
}

func run() error {
	if *host != "" {
		sb := bytes.Buffer{}

		sb.WriteString(fmt.Sprintf("%s %s", CMD_CONNECT, *host))
		if *username != "" {
			sb.WriteString(fmt.Sprintf("%s %s", *username, *password))
		}
		sb.WriteString("\n")

		if *clientId != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", CMD_CLIENTID, *clientId))
		}

		if *topic != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", CMD_TOPIC, *topic))
		}

		if *subscribe != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n", CMD_SUBSCRIBE, *subscribe))
		}

		if *qos != 0 {
			sb.WriteString(fmt.Sprintf("%s %d\n", CMD_QOS, *qos))
		}

		if *timeout != 0 {
			sb.WriteString(fmt.Sprintf("%s %d\n", CMD_TIMEOUT, *timeout))
		}

		if *publish != "" {
			sb.WriteString(fmt.Sprintf("%s '%s' %d %v\n", CMD_PUBLISH, *publish, *count, *retained))
		}

		common.Error(executeScript(sb.Bytes()))
	}

	if *file != "" {
		err := executeFile(*file)
		if common.Error(err) {
			return err
		}
	}

	defer func() {
		for _, conn := range conns {
			conn.disconnect()
		}
	}()

	for {
		prompt("")

		reader := bufio.NewReader(os.Stdin)
		cmdline, err := reader.ReadString('\n')
		common.DebugError(err)

		inPrompt = false

		err = executeLine(cmdline)
		if common.IsErrExit(err) {
			break
		}
		common.Error(err)
	}

	return nil
}

func main() {
	common.Run(nil)
}
