package ros

import (
	"io"
	"net"
	"testing"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

// `subscriber_test.go` uses `testMessageType` and `testMessage` defined in `subscription_test.go`.

func TestRemotePublisherConn_DoesConnect(t *testing.T) {
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

	startRemotePublisherConn(&log, pubURI, topic, msgType, nodeID, msgChan, quitChan, disconnectedChan)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	readAndVerifySubscriberHeader(t, conn, topic, msgType) // Test helper from subscription_test.go.

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

func TestRemotePublisherConn_ClosesFromSignal(t *testing.T) {
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

	startRemotePublisherConn(&log, pubURI, topic, msgType, nodeID, msgChan, quitChan, disconnectedChan)

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

func TestRemotePublisherConn_RemoteReceivesData(t *testing.T) {
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

	startRemotePublisherConn(&log, pubURI, topic, msgType, nodeID, msgChan, quitChan, disconnectedChan)

	conn := connectToSubscriber(t, l, topic, msgType)
	defer conn.Close()

	// Send something!
	sendMessageAndReceiveInChannel(t, conn, msgChan, []byte{0x12, 0x23})

	// Send another one!
	sendMessageAndReceiveInChannel(t, conn, msgChan, []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8})

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

func sendMessageAndReceiveInChannel(t *testing.T, conn net.Conn, msgChan chan messageEvent, buffer []byte) {
	if len(buffer) > 255 {
		t.Fatalf("sendMessageAndReceiveInChannel helper doesn't support more than 255 bytes!")
	}

	// Packet structure is [ LENGTH<uint32> | PAYLOAD<bytes[LENGTH]> ]
	length := uint8(len(buffer))
	n, err := conn.Write([]byte{length, 0x00, 0x00, 0x00})
	if n != 4 || err != nil { // Send length.
		t.Fatalf("Failed to write message size, n: %d : err: %s", n, err)
	}
	n, err = conn.Write(buffer) // Send payload.
	if n != len(buffer) || err != nil {
		t.Fatalf("Failed to write message payload, n: %d : err: %s", n, err)
	}

	select {
	case message := <-msgChan:

		if message.event.PublisherName != "testPublisher" {
			t.Fatalf("Published with the wrong publisher name: %s", message.event.PublisherName)
		}
		if len(message.bytes) != len(buffer) {
			t.Fatalf("Payload size is incorrect: %d, expected: %d", len(message.bytes), len(buffer))
		}
		for i := 1; i < len(buffer); i++ {
			if message.bytes[i] != buffer[i] {
				t.Fatalf("message.bytes[%d] = %x, expected %x", i, message.bytes[i], buffer[i])
			}
		}
		return
	case <-time.After(time.Duration(500) * time.Millisecond):
		t.Fatalf("Did not receive message from channel")
	}
}
