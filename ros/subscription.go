package ros

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
	"github.com/sirupsen/logrus"
)

// defaultSubscription connects to a publisher and runs a go routine to maintain its connection and packetize messages from the tcp stream. Messages are passed through the messageChan channel.
type defaultSubscription struct {
	pubURI                 string
	topic                  string
	msgType                MessageType
	nodeID                 string
	messageChan            chan messageEvent
	requestStopChan        chan struct{} // tell the subscription to disconnect
	remoteDisconnectedChan chan string   // tell the subscriber that the remote has disconnected
	event                  MessageEvent
	pool                   []byte
}

// newDefaultSubscription populates a subscription struct from the instantiation fields and fills in default data for the operational fields.
func newDefaultSubscription(
	pubURI string, topic string, msgType MessageType, nodeID string,
	messageChan chan messageEvent,
	requestStopChan chan struct{},
	remoteDisconnectedChan chan string) *defaultSubscription {

	return &defaultSubscription{
		pubURI:                 pubURI,
		topic:                  topic,
		msgType:                msgType,
		nodeID:                 nodeID,
		messageChan:            messageChan,
		requestStopChan:        requestStopChan,
		remoteDisconnectedChan: remoteDisconnectedChan,
		event:                  MessageEvent{"", time.Time{}, nil},
		pool:                   nil,
	}
}

// connectionFailureMode specifies a connection failure mode.
type connectionFailureMode int

const (
	publisherDisconnected connectionFailureMode = iota
	tcpOutOfSync
	readFailure
	stopRequested
)

// readResult determines the result of a subscription read operation.
type readResult int

const (
	readOk readResult = iota
	readFailed
	readTimeout
	remoteDisconnected
	readOutOfSync
)

// start spawns a go routine which connects a subscription to a publisher.
func (s *defaultSubscription) start(log *modular.ModuleLogger) {
	go s.run(log)
}

// run connects to a publisher and attempts to maintain a connection until either a stop is requested or the publisher disconnects.
func (s *defaultSubscription) run(log *modular.ModuleLogger) {
	logger := *log
	logger.WithFields(logrus.Fields{"topic": s.topic}).Debug("defaultSubscription.run() has started")

	defer func() {
		logger.WithFields(logrus.Fields{"topic": s.topic}).Debug("defaultSubscription.run() has exited")
	}()

	var conn net.Conn

	// The recovery loop: if a connection to the publisher fails or goes out of sync, this loop allows us to attempt to start again with a new subscription.
	for {
		// Establish a connection with our publisher.
		if s.connectToPublisher(&conn, log) == false {
			if conn != nil {
				conn.Close()
			}
			logger.WithFields(logrus.Fields{"topic": s.topic}).Info("could not connect to publisher, closing connection")
			return
		}

		// Reading from publisher, this will only return when our connection fails.
		connectionFailureMode := s.readFromPublisher(conn)

		// Under healthy conditions, we don't get here. Always close the connection, then handle the returned connection state.
		conn.Close()
		conn = nil

		switch connectionFailureMode {
		case tcpOutOfSync: // TCP out of sync; we will attempt to resync by closing the connection and trying again.
			logger.WithFields(logrus.Fields{"topic": s.topic}).Debug("connection closed - attempting to reconnect with publisher")
			continue
		case stopRequested: // A stop was externally requested - easy, just return!
			return
		case publisherDisconnected: // Publisher disconnected - not much we can do here, the subscription has ended.
			logger.WithFields(logrus.Fields{"topic": s.topic, "pubURI": s.pubURI}).Info("connection closed - publisher has disconnected")
			s.remoteDisconnectedChan <- s.pubURI
			return
		case readFailure: // Read failure; the reason is uncertain, maybe the bus is polluted? We give up.
			logger.WithFields(logrus.Fields{"topic": s.topic}).Error("connection closed - failed to read a message correctly")
			s.remoteDisconnectedChan <- s.pubURI
			return
		default: // Unknown failure - this is a bug, log the connectionFailureMode to help determine cause.
			logger.WithFields(logrus.Fields{"topic": s.topic, "failureMode": connectionFailureMode}).Error("connection closed - unknown failure mode")
			return
		}
	}
}

// connectToPublisher estabilishes a TCPROS connection with a publishing node by exchanging headers to ensure both nodes are using the same message type.
func (s *defaultSubscription) connectToPublisher(conn *net.Conn, log *modular.ModuleLogger) bool {
	var err error

	logger := *log

	// 1. Connnect to tcp.
	select {
	case <-time.After(time.Duration(3000) * time.Millisecond):
		logger.WithFields(logrus.Fields{"topic": s.topic, "pubURI": s.pubURI}).Error("failed to connect: timed out")
		return false
	default:
		*conn, err = net.Dial("tcp", s.pubURI)
		if err != nil {
			logger.WithFields(logrus.Fields{"topic": s.topic, "pubURI": s.pubURI, "error": err}).Error("failed to connect: connection error")
			return false
		}
	}

	// 2. Write connection header to the publisher.
	var subscriberHeaders []header
	subscriberHeaders = append(subscriberHeaders, header{"topic", s.topic})
	subscriberHeaders = append(subscriberHeaders, header{"md5sum", s.msgType.MD5Sum()})
	subscriberHeaders = append(subscriberHeaders, header{"type", s.msgType.Name()})
	subscriberHeaders = append(subscriberHeaders, header{"callerid", s.nodeID})

	logFields := make(logrus.Fields)
	for _, h := range subscriberHeaders {
		logFields[h.key] = h.value
	}
	logger.WithFields(logFields).Debug("writing TCPROS connection header")

	err = writeConnectionHeader(subscriberHeaders, *conn)
	if err != nil {
		logger.WithFields(logrus.Fields{"topic": s.topic, "error": err}).Error("failed to write connection header")
		return false
	}

	// 3. Read the publisher's reponse header.
	var resHeaders []header
	resHeaders, err = readConnectionHeader(*conn)
	if err != nil {
		logger.WithFields(logrus.Fields{"topic": s.topic, "error": err}).Error("failed to read response header")
		return false
	}
	logFields = logrus.Fields{"topic": s.topic}
	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
		logFields["pub["+h.key+"]"] = h.value
	}
	logger.WithFields(logFields).Debug("received TCPROS response header")

	// 4. Verify the publisher's response header.
	if resHeaderMap["type"] != s.msgType.Name() || resHeaderMap["md5sum"] != s.msgType.MD5Sum() {
		logFields = make(logrus.Fields)
		for key, value := range resHeaderMap {
			logFields["pub["+key+"]"] = value
		}
		for _, h := range subscriberHeaders {
			logFields["sub["+h.key+"]"] = h.value
		}
		logger.WithFields(logFields).Error("publisher provided incompatable message header")
		return false
	}

	// Some incomplete TCPROS implementations do not include topic name in response; set it if it is currently empty.
	if resHeaderMap["topic"] == "" {
		resHeaderMap["topic"] = s.topic
	}

	// Construct the event struct to be sent with each message.
	s.event = MessageEvent{
		PublisherName:    resHeaderMap["callerid"],
		ConnectionHeader: resHeaderMap,
	}
	return true
}

// readFromPublisher maintains a connection with a publisher. When a connection is stable, it will loop until either the publisher or subscriber disconnects.
func (s *defaultSubscription) readFromPublisher(conn net.Conn) connectionFailureMode {
	readingSize := true
	var msgSize int
	var buffer []byte
	var result readResult

	// Subscriber loop:
	// - Checks for external stop requests.
	// - Packages the tcp serial stream into messages and passes them through the message channel.
	for {
		select {
		case <-s.requestStopChan:
			return stopRequested
		default:
			conn.SetDeadline(time.Now().Add(1000 * time.Millisecond))
			if readingSize {
				msgSize, result = readSize(conn)

				if result == readOk {
					readingSize = false
					continue
				}

				if result == readTimeout {
					continue // Try again!
				}

			} else {
				buffer, result = s.readRawMessage(conn, msgSize)

				if result == readOk {
					s.event.ReceiptTime = time.Now()
					select {
					case s.messageChan <- messageEvent{bytes: buffer, event: s.event}:
					case <-time.After(time.Duration(30) * time.Millisecond):
						// Dropping message
					}
					readingSize = true
				}

				if result == readTimeout {
					// We just failed to read a message; it is likely that we are now out of sync.
					return tcpOutOfSync
				}
			}

			// Handle read result cases
			if result == readOutOfSync {
				return tcpOutOfSync
			}
			if result == readFailed {
				return readFailure
			}
			if result == remoteDisconnected {
				return publisherDisconnected
			}
		}
	}
}

// readSize reads the number of bytes to expect in the message payload. The structure of a ROS message is: [SIZE|PAYLOAD] where size is a uint32.
func readSize(r io.Reader) (int, readResult) {
	var msgSize uint32

	err := binary.Read(r, binary.LittleEndian, &msgSize)
	if err != nil {
		return 0, errorToReadResult(err)
	}
	// Check that our message size is in a range of possible sizes for a ros message.
	if msgSize < 256000000 {
		return int(msgSize), readOk
	}
	// A large number of bytes is an indication of a transport error - we assume we are out of sync.
	return 0, readOutOfSync
}

// readRawMessage reads ROS message bytes from the io.Reader.
func (s *defaultSubscription) readRawMessage(r io.Reader, size int) ([]byte, readResult) {
	// First, ensure our pool is large enough to receive the bytes. It is reallocated if it is too small.
	if len(s.pool) < size {
		s.pool = make([]byte, size)
	}
	buffer := s.pool[:size]

	// Read the full buffer; we expect this call to timeout if the read takes too long.
	_, err := io.ReadFull(r, buffer)
	if err != nil {
		return buffer, errorToReadResult(err)
	}

	return buffer, readOk
}

// errorToReadResult converts errors to readResult to be handled further up the callstack.
func errorToReadResult(err error) readResult {
	if err == io.EOF {
		return remoteDisconnected
	}
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return readTimeout
	}
	// Not sure what the cause was - it is just a generic readFailure.
	return readFailed
}
