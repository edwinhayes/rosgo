package ros

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

type startRemotePublisher func(*modular.ModuleLogger,
	string, string, string,
	string, string,
	chan messageEvent,
	chan struct{},
	chan string, MessageType)

func BenchmarkRemotePublisherConn_Throughput1Kb(b *testing.B) {

	l, conn, msgChan, disconnectedChan := setupRemotePublisherConnBenchmark(b, startRemotePublisherConn)
	defer l.Close()
	defer conn.Close()

	buffer := make([]byte, 1000) // 1 kB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	teardownRemotePublisherConnBenchmark(b, l, conn, disconnectedChan)
}

func BenchmarkRemotePublisherConn_NewThroughput1Kb(b *testing.B) {
	l, conn, msgChan, disconnectedChan := setupRemotePublisherConnBenchmark(b, newStartRemotePublisherConn)
	defer l.Close()
	defer conn.Close()

	buffer := make([]byte, 1000) // 1 kB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	teardownRemotePublisherConnBenchmark(b, l, conn, disconnectedChan)
}

func BenchmarkRemotePublisherConn_Throughput1Mb(b *testing.B) {

	l, conn, msgChan, disconnectedChan := setupRemotePublisherConnBenchmark(b, startRemotePublisherConn)
	defer l.Close()
	defer conn.Close()

	buffer := make([]byte, 1000000) // 1 MB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	teardownRemotePublisherConnBenchmark(b, l, conn, disconnectedChan)
}

func BenchmarkRemotePublisherConn_NewThroughput1Mb(b *testing.B) {
	l, conn, msgChan, disconnectedChan := setupRemotePublisherConnBenchmark(b, newStartRemotePublisherConn)
	defer l.Close()
	defer conn.Close()

	buffer := make([]byte, 1000000) // 1 MB of data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendMessageAndReceiveInChannelWithB(b, conn, msgChan, buffer)
	}

	teardownRemotePublisherConnBenchmark(b, l, conn, disconnectedChan)
}

// Benchmark helpers

//
// Setup, establishes all init values and kicks off the start function
//
func setupRemotePublisherConnBenchmark(b *testing.B, start startRemotePublisher) (net.Listener, net.Conn, chan messageEvent, chan string) {
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

	go start(
		&log,
		l.Addr().String(),
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
	return l, conn, msgChan, disconnectedChan
}

//
// Teardown, take down TCP connections and ensures the remotePublisherConn disconnects as expected
//
func teardownRemotePublisherConnBenchmark(b *testing.B, l net.Listener, conn net.Conn, disconnectedChan chan string) {
	conn.Close()
	l.Close()
	select {
	case <-disconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		b.Fatalf("Took too long for client to disconnect from publisher")
	}
}

//
// Connects the test "publisher" to the subscriber, exectutes a header exchange
//
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

//
// Sends a message to the subscriber with a set number of bytes
//
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
