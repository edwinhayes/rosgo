package ros

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

//
// Set up testMessage fakes
//
type testMessageType struct{}
type testMessage struct{}

var _ MessageType = testMessageType{}
var _ Message = testMessage{}

func (t testMessageType) Text() string {
	return "test_message_type"
}

func (t testMessageType) MD5Sum() string {
	return "fakeMD5"
}

func (t testMessageType) Name() string {
	return "test_message"
}

func (t testMessageType) NewMessage() Message {
	return &testMessage{}
}

func (t testMessage) Type() MessageType {
	return &testMessageType{}
}

func (t testMessage) Serialize(buf *bytes.Buffer) error {
	return nil
}

func (t testMessage) Deserialize(buf *bytes.Reader) error {
	return nil
}

func TestRemoteConnects(t *testing.T) {
	logger := modular.NewRootLogger(logrus.New())

	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent)
	quitChan := make(chan struct{})
	disconnectedChan := make(chan string)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	pubURI := l.Addr().String()

	go newStartRemotePublisherConn(
		&log,
		pubURI,
		topic,
		msgType.MD5Sum(),
		msgType.Name(),
		nodeID,
		msgChan,
		quitChan,
		disconnectedChan,
		msgType,
	)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	resHeaders, err := readConnectionHeader(conn)

	if err != nil {
		t.Fatal("Failed to read header:", err)
	}

	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
	}

	if resHeaderMap["md5sum"] != msgType.MD5Sum() {
		t.Fatalf("Incorrect MD5 sum %s", resHeaderMap["md5sum"])
	}

	if resHeaderMap["topic"] != topic {
		t.Fatalf("Incorrect topic: %s", topic)
	}

	if resHeaderMap["type"] != msgType.Name() {
		t.Fatalf("Incorrect type: %s", resHeaderMap["type"])
	}

	if resHeaderMap["callerid"] != "testNode" {
		t.Fatalf("Incorrect caller ID: %s", resHeaderMap["testNode"])
	}

	replyHeader := []header{
		{"topic", topic},
		{"md5sum", msgType.MD5Sum()},
		{"type", msgType.Name()},
		{"callerid", "testPublisher"},
	}

	err = writeConnectionHeader(replyHeader, conn)
	if err != nil {
		t.Fatalf("Failed to write header: %s", replyHeader)
	}

	conn.Close()
	l.Close()
	select {
	case <-disconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Took too long for client to disconnect from publisher")
	}
}

func TestClosesFromSignal(t *testing.T) {
	logger := modular.NewRootLogger(logrus.New())

	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent)
	quitChan := make(chan struct{})
	disconnectedChan := make(chan string)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	pubURI := l.Addr().String()

	go newStartRemotePublisherConn(
		&log,
		pubURI,
		topic,
		msgType.MD5Sum(),
		msgType.Name(),
		nodeID,
		msgChan,
		quitChan,
		disconnectedChan,
		msgType,
	)

	conn := connectToSubscriber(t, l, topic, msgType)
	defer conn.Close()

	// Signal to close
	quitChan <- struct{}{}

	// Check that buffer closed
	buffer := make([]byte, 1)
	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_, err = conn.Read(buffer)

	if err != io.EOF {
		t.Fatalf("Expected subscriber to close connection")
	}

	conn.Close()
	l.Close()
}

func TestRemoteReceivesData(t *testing.T) {
	logger := modular.NewRootLogger(logrus.New())

	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent)
	quitChan := make(chan struct{})
	disconnectedChan := make(chan string)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	pubURI := l.Addr().String()

	go newStartRemotePublisherConn(
		&log,
		pubURI,
		topic,
		msgType.MD5Sum(),
		msgType.Name(),
		nodeID,
		msgChan,
		quitChan,
		disconnectedChan,
		msgType,
	)

	conn := connectToSubscriber(t, l, topic, msgType)
	defer conn.Close()

	// Size = 2 bytes
	n, err := conn.Write([]byte{0x02, 0x00, 0x00, 0x00})
	if n != 4 || err != nil {
		t.Fatalf("Failed to write message size, n: %d : err: %s", n, err)
	}
	n, err = conn.Write([]byte{0x34, 0x12}) // payload
	if n != 2 || err != nil {
		t.Fatalf("Failed to write message payload, n: %d : err: %s", n, err)
	}

	select {
	case message := <-msgChan:

		if message.event.PublisherName != "testPublisher" {
			t.Fatalf("Published with the wrong publisher name: %s", message.event.PublisherName)
		}
		if len(message.bytes) != 2 {
			t.Fatalf("Payload size is incorrect: %d", len(message.bytes))
		}
		if message.bytes[0] != 0x34 || message.bytes[1] != 0x12 {
			t.Fatalf("Published the wrong payload: %x:02 %x:02", message.bytes[0], message.bytes[1])
		}
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Took too long for client to disconnect from publisher")
	}

	conn.Close()
	l.Close()
	select {
	case channelName := <-disconnectedChan:
		t.Log(channelName)
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Took too long for client to disconnect from publisher")
	}
}

func connectToSubscriber(t *testing.T, l net.Listener, topic string, msgType testMessageType) net.Conn {
	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	_, err = readConnectionHeader(conn)

	if err != nil {
		t.Fatal("Failed to read header:", err)
	}

	replyHeader := []header{
		{"topic", topic},
		{"md5sum", msgType.MD5Sum()},
		{"type", msgType.Name()},
		{"callerid", "testPublisher"},
	}

	err = writeConnectionHeader(replyHeader, conn)
	if err != nil {
		t.Fatalf("Failed to write header: %s", replyHeader)
	}

	return conn
}
