package ros

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

func BenchmarkRemotePublisherConn_Throughput(b *testing.B) {
	logger := modular.NewRootLogger(logrus.New())
	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent)
	quitChan := make(chan struct{})
	disconnectedChan := make(chan string)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()
	log.SetLevel(logrus.InfoLevel)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatal(err)
	}
	defer l.Close()

	pubURI := l.Addr().String()

	go startRemotePublisherConn(
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

	conn := connectToSubscriberWithB(b, l, topic, msgType)
	defer conn.Close()

	buffer := make([]byte, 1000000) // 1 MB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	conn.Close()
	l.Close()
	select {
	case <-disconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		b.Fatalf("Took too long for client to disconnect from publisher")
	}
}

func BenchmarkRemotePublisherConn_NewThroughput(b *testing.B) {
	logger := modular.NewRootLogger(logrus.New())
	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent)
	quitChan := make(chan struct{})
	disconnectedChan := make(chan string)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()
	log.SetLevel(logrus.InfoLevel)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatal(err)
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

	conn := connectToSubscriberWithB(b, l, topic, msgType)
	defer conn.Close()

	buffer := make([]byte, 1000000) // 1 MB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	conn.Close()
	l.Close()
	select {
	case <-disconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		b.Fatalf("Took too long for client to disconnect from publisher")
	}
}

func connectToSubscriberWithB(t *testing.B, l net.Listener, topic string, msgType testMessageType) net.Conn {
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

func sendMessageAndReceiveInChannelWithB(t *testing.B, conn net.Conn, msgChan chan messageEvent, buffer []byte) {

	err := binary.Write(conn, binary.LittleEndian, uint32(len(buffer)))
	if err != nil {
		t.Fatalf("Failed to write message size, err: %s", err)
	}
	n, err := conn.Write(buffer) // payload
	if n != len(buffer) || err != nil {
		t.Fatalf("Failed to write message payload, n: %d : err: %s", n, err)
	}

	select {
	case <-msgChan:
		// Assume the message is fine - we have unit tests for that!
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Did not receive message from channel")
	}
}
