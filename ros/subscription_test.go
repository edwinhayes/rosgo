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

//
// Basic integration stuff - TODO: figure out how to tidy this up...
//
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

	resHeaders, err := readConnectionHeader(conn)

	if err != nil {
		t.Fatal("Failed to read header:", err)
	}

	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
	}

	if resHeaderMap["md5sum"] != subscription.msgType.MD5Sum() {
		t.Fatalf("Incorrect MD5 sum %s", resHeaderMap["md5sum"])
	}

	if resHeaderMap["topic"] != subscription.topic {
		t.Fatalf("Incorrect topic: %s", subscription.topic)
	}

	if resHeaderMap["type"] != subscription.msgType.Name() {
		t.Fatalf("Incorrect type: %s", resHeaderMap["type"])
	}

	if resHeaderMap["callerid"] != "testNode" {
		t.Fatalf("Incorrect caller ID: %s", resHeaderMap["testNode"])
	}

	replyHeader := []header{
		{"topic", subscription.topic},
		{"md5sum", subscription.msgType.MD5Sum()},
		{"type", subscription.msgType.Name()},
		{"callerid", "testPublisher"},
	}

	err = writeConnectionHeader(replyHeader, conn)
	if err != nil {
		t.Fatalf("Failed to write header: %s", replyHeader)
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
