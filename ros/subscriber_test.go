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
	topic := "/test/topic"
	msgType := testMessageType{}

	l, conn, _, _, _, disconnectedChan := setupRemotePublisherConnTest(t)
	defer l.Close()
	defer conn.Close()

	readAndVerifySubscriberHeader(t, conn, topic, msgType) // Test helper from subscription_test.go.

	replyHeader := []header{
		{"topic", topic},
		{"md5sum", msgType.MD5Sum()},
		{"type", msgType.Name()},
		{"callerid", "testPublisher"},
	}

	err := writeConnectionHeader(replyHeader, conn)
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

	l, conn, _, _, quitChan, _ := setupRemotePublisherConnTest(t)
	defer l.Close()

	connectToSubscriber(t, conn)

	// Signal to close.
	quitChan <- struct{}{}

	// Check that buffer closed.
	buffer := make([]byte, 1)
	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
	_, err := conn.Read(buffer)

	if err != io.EOF {
		t.Fatalf("Expected subscriber to close connection")
	}

	conn.Close()
	l.Close()
}

func TestRemotePublisherConn_RemoteReceivesData(t *testing.T) {

	l, conn, msgChan, _, _, disconnectedChan := setupRemotePublisherConnTest(t)
	defer l.Close()
	defer conn.Close()

	connectToSubscriber(t, conn)

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

//
// Setup, establishes all init values and kicks off the start function
//
func setupRemotePublisherConnTest(t *testing.T) (net.Listener, net.Conn, chan messageEvent, chan bool,
	chan struct{}, chan string) {
	logger := modular.NewRootLogger(logrus.New())
	topic := "/test/topic"
	nodeID := "testNode"
	msgChan := make(chan messageEvent, 1)
	enableChan := make(chan bool, 1)
	quitChan := make(chan struct{}, 1)
	disconnectedChan := make(chan string, 1)
	msgType := testMessageType{}

	log := logger.GetModuleLogger()
	log.SetLevel(logrus.InfoLevel)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	pubURI := l.Addr().String()

	startRemotePublisherConn(&log, pubURI, topic, msgType, nodeID, msgChan, enableChan, quitChan, disconnectedChan)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	return l, conn, msgChan, enableChan, quitChan, disconnectedChan
}

func connectToSubscriber(t *testing.T, conn net.Conn) {
	msgType := testMessageType{}
	topic := "/test/topic"

	_, err := readConnectionHeader(conn)

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
	case <-time.After(time.Duration(10) * time.Millisecond):
		t.Fatalf("Did not receive message from channel")
	}
}
