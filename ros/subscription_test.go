package ros

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

// `subscription_test.go` uses `testMessageType` and `testMessage` defined in `subscriber_test.go`.
// Integration of subscriptions is tested in the RemotePublisherConn tests in `subscriber_test.go`.

// Helper structs

// testReader provides the io.Reader interface.
type testReader struct {
	buffer []byte
	n      int
	err    error
}

func (r *testReader) Read(buf []byte) (n int, err error) {
	_ = copy(buf, r.buffer)
	n = r.n
	err = r.err
	return
}

// testLogger provides the modular.ModuleLogger interface

type testLogger struct {
}

var _ io.Reader = &testReader{} // verify that testReader satisfies the reader interface

func getTestSubscription(pubURI string) *defaultSubscription {

	topic := "/test/topic"
	nodeID := "testNode"
	messageChan := make(chan messageEvent)
	requestStopChan := make(chan struct{})
	remoteDisconnectedChan := make(chan string)
	msgType := testMessageType{}

	return newDefaultSubscription(
		pubURI, topic, msgType, nodeID,
		messageChan,
		requestStopChan,
		remoteDisconnectedChan)
}

//
// Read Size tests
//

func TestSubscription_ReadSize(t *testing.T) {
	type testCase struct {
		buffer   []byte
		expected int
	}

	testCases := []testCase{
		{[]byte{0x00, 0x00, 0x00, 0x00}, 0},
		{[]byte{0x01, 0x00, 0x00, 0x00}, 1},
		{[]byte{0x0F, 0x00, 0x00, 0x00}, 15},
		{[]byte{0x00, 0x01, 0x00, 0x00}, 256},
		{[]byte{0xa1, 0x86, 0x01, 0x00}, 100001},
	}

	for _, tc := range testCases {
		reader := testReader{tc.buffer, 4, nil}
		n, res := readSize(&reader)
		if res != readOk {
			t.Fatalf("Expected read result %d, but got %d", readOk, res)
		}
		if n != tc.expected {
			t.Fatalf("ReadSize failed, expected %d, got %d", tc.expected, n)
		}

	}
}

// Error cases
func TestSubscription_ReadSize_TooLarge(t *testing.T) {
	reader := testReader{[]byte{0x00, 0x00, 0x00, 0x80}, 4, nil}
	_, res := readSize(&reader)
	if res != readOutOfSync {
		t.Fatalf("Expected read result %d, but got %d", readOutOfSync, res)
	}
}

func TestSubscription_ReadSize_disconnected(t *testing.T) {
	reader := testReader{[]byte{}, 0, io.EOF}
	_, res := readSize(&reader)
	if res != remoteDisconnected {
		t.Fatalf("Expected read result %d, but got %d", remoteDisconnected, res)
	}
}

func TestSubscription_ReadSize_otherError(t *testing.T) {
	reader := testReader{[]byte{}, 0, errors.New("MysteryError")}
	_, res := readSize(&reader)
	if res != readFailed {
		t.Fatalf("Expected read result %d, but got %d", readFailed, res)
	}
}

//
// Read Raw Data tests
//

// Checks pool buffer resizing logic
func TestSubscription_ReadRawData_PoolBuffer(t *testing.T) {
	subscription := getTestSubscription("testUri")
	if len(subscription.pool) != 0 {
		t.Fatalf("Expected pool size of 0, but got %d", len(subscription.pool))
	}

	reader := testReader{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 4, nil}

	// Test 1, read 4 bytes, pool size goes to 4 bytes
	reader.n = 4
	_, _ = subscription.readRawMessage(&reader, 4)
	if len(subscription.pool) != 4 {
		t.Fatalf("Expected pool size of 4, but got %d", len(subscription.pool))
	}

	// Test 2, read 2 bytes, pool size stays at 4 bytes
	reader.n = 2
	_, _ = subscription.readRawMessage(&reader, 2)
	if len(subscription.pool) != 4 {
		t.Fatalf("Expected pool size of 4, but got %d", len(subscription.pool))
	}

	// Test 3, read 10 bytes, pool size goes to 10 bytes
	reader.n = 10
	_, _ = subscription.readRawMessage(&reader, 10)
	if len(subscription.pool) != 10 {
		t.Fatalf("Expected pool size of 10, but got %d", len(subscription.pool))
	}
}

// checks basic buffer reading works correctly
func TestSubscription_ReadRawData_ReadData(t *testing.T) {
	subscription := getTestSubscription("testUri")

	reader := testReader{[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 10, nil}

	buf, res := subscription.readRawMessage(&reader, 4)
	if res != readOk {
		t.Fatalf("Expected read result %d, but got %d", readOk, res)
	}

	for i := 0; i < len(buf); i++ {
		if buf[i] != reader.buffer[i] {
			t.Fatalf("Expected read buf[%d] = %x, but got %x", i, reader.buffer[i], buf[i])
		}
	}
}

// checks handling disconnections
func TestSubscription_ReadRawData_disconnected(t *testing.T) {
	subscription := getTestSubscription("testUri")

	reader := testReader{[]byte{}, 0, io.EOF}

	_, res := subscription.readRawMessage(&reader, 4)
	if res != remoteDisconnected {
		t.Fatalf("Expected read result %d, but got %d", remoteDisconnected, res)
	}
}

// Create a new subscription and pass headers correctly.
func TestSubscription_NewSubscription(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	pubURI := l.Addr().String()

	subscription := getTestSubscription(pubURI)

	logger := modular.NewRootLogger(logrus.New())
	log := logger.GetModuleLogger()

	subscription.start(&log)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	readAndVerifySubscriberHeader(t, conn, subscription.topic, subscription.msgType)

	replyHeader := []header{
		{"topic", subscription.topic},
		{"md5sum", subscription.msgType.MD5Sum()},
		{"type", subscription.msgType.Name()},
		{"callerid", "testPublisher"},
	}

	writeAndVerifyPublisherHeader(t, conn, subscription, replyHeader)

	conn.Close()
	l.Close()
	select {
	case <-subscription.remoteDisconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Took too long for client to disconnect from publisher")
	}
}

// Create a new subscription and pass headers correctly.
func TestSubscription_NewSubscription_NoTopicInHeader(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	pubURI := l.Addr().String()

	subscription := getTestSubscription(pubURI)

	logger := modular.NewRootLogger(logrus.New())
	log := logger.GetModuleLogger()

	subscription.start(&log)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	readAndVerifySubscriberHeader(t, conn, subscription.topic, subscription.msgType)

	replyHeader := []header{
		{"md5sum", subscription.msgType.MD5Sum()},
		{"type", subscription.msgType.Name()},
		{"callerid", "testPublisher"},
	}

	writeAndVerifyPublisherHeader(t, conn, subscription, replyHeader)

	// Expect that we store the topic anyway!
	if result, ok := subscription.event.ConnectionHeader["topic"]; ok {
		if subscription.topic != result {
			t.Fatalf("expected header[topic] = %s, but got %s", subscription.topic, result)
		}
	} else {
		t.Fatalf("subscription did not store header data for topic")
	}

	conn.Close()
	l.Close()
	select {
	case <-subscription.remoteDisconnectedChan:
		return
	case <-time.After(time.Duration(100) * time.Millisecond):
		t.Fatalf("Took too long for client to disconnect from publisher")
	}
}

// Subscription closes when it receives invalid response header.
func TestSubscription_NewSubscription_InvalidResponseHeader(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	pubURI := l.Addr().String()

	subscription := getTestSubscription(pubURI)

	logger := modular.NewRootLogger(logrus.New())
	log := logger.GetModuleLogger()

	subscription.start(&log)

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	readAndVerifySubscriberHeader(t, conn, subscription.topic, subscription.msgType)

	invalidMD5 := "00112233445566778899aabbccddeeff"
	replyHeader := []header{
		{"topic", subscription.topic},
		{"md5sum", invalidMD5},
		{"type", subscription.msgType.Name()},
		{"callerid", "testPublisher"},
	}

	if err := writeConnectionHeader(replyHeader, conn); err != nil {
		t.Fatalf("failed to write header: %s", replyHeader)
	}

	// Wait for the subscription to receive the data.
	<-time.After(time.Millisecond)

	// Expect the Subscription has closed the channel.
	dummySlice := make([]byte, 1)
	if _, err := conn.Read(dummySlice); err != io.EOF {
		t.Fatalf("expected subscription to close connection when receiving invalid header")

	}

	conn.Close()
	l.Close()
}

// Private Helper functions/

//
func readAndVerifySubscriberHeader(t *testing.T, conn net.Conn, topic string, msgType MessageType) {
	resHeaders, err := readConnectionHeader(conn)

	if err != nil {
		t.Fatal("Failed to read header:", err)
	}

	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
	}

	if resHeaderMap["md5sum"] != msgType.MD5Sum() {
		t.Fatalf("incorrect MD5 sum %s", resHeaderMap["md5sum"])
	}

	if resHeaderMap["topic"] != topic {
		t.Fatalf("incorrect topic: %s", topic)
	}

	if resHeaderMap["type"] != msgType.Name() {
		t.Fatalf("incorrect type: %s", resHeaderMap["type"])
	}

	if resHeaderMap["callerid"] != "testNode" {
		t.Fatalf("incorrect caller ID: %s", resHeaderMap["testNode"])
	}
}

func writeAndVerifyPublisherHeader(t *testing.T, conn net.Conn, subscription *defaultSubscription, replyHeader []header) {

	if err := writeConnectionHeader(replyHeader, conn); err != nil {
		t.Fatalf("Failed to write header: %s", replyHeader)
	}

	// wait for the subscription to receive the data
	<-time.After(time.Millisecond)

	for _, expected := range replyHeader {
		if result, ok := subscription.event.ConnectionHeader[expected.key]; ok {
			if expected.value != result {
				t.Fatalf("expected header[%s] = %s, but got %s", expected.key, expected.value, result)
			}
		} else {
			t.Fatalf("subscription did not store header data for %s", expected.key)
		}
	}
}
