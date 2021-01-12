package ros

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
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

// connectionFailureMode indicates a connection failure mode
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
	logger.Debug(s.topic, " : defaultSubscription.start()")

	defer func() {
		logger.Debug(s.topic, " : defaultSubscription.start() exit")
	}()

	var conn net.Conn

	// The recovery loop
	// If a connection to the publisher fails or goes out of sync, this loop allows us to
	// attempt to start again with a new subscription.
	for {
		// Connect
		if s.connectToPublisher(&conn, log) == false {
			if conn != nil {
				conn.Close()
			}
			logger.Info(s.topic, " : Could not connect to publisher, closing connection")
			return
		}
		defer conn.Close() // Make sure we close this

		// Reading from publisher
		connectionFailureMode := s.readFromPublisher(conn)

		// Under healthy conditions, we don't get here
		// handle the returned connection state

		// TCP out of sync; we will attempt to resync by closing the connection and trying again
		if connectionFailureMode == tcpOutOfSync {
			conn.Close()
			logger.Debug(s.topic, " : Connection closed, reconnecting with publisher")
		}

		// A stop was externally requested - easy one!
		if connectionFailureMode == stopRequested {
			return
		}

		// Publisher disconnected - not much we can do here, the subscription has ended
		if connectionFailureMode == publisherDisconnected {
			logger.Infof("Publisher %s on topic %s disconnected", s.pubURI, s.topic)
			s.remoteDisconnectedChan <- s.pubURI
			return
		}

		// read failure; the reason is uncertain, so we will give up
		if connectionFailureMode == readFailure {
			logger.Error(s.topic, " : Failed to read a message correctly")
			s.remoteDisconnectedChan <- s.pubURI
			return
		}
	}
}

// ConnectToPublisher Estabilishes a TCPROS connection with a publishing node
// Connects via TCP and then exchanges headers to ensure
// both nodes are using the same message type
func (s *defaultSubscription) connectToPublisher(conn *net.Conn, log *modular.ModuleLogger) bool {
	var err error

	logger := *log

	// 1. Connnect to tcp
	select {
	case <-time.After(time.Duration(3000) * time.Millisecond):
		logger.Error(s.topic, " : Failed to connect to ", s.pubURI, "timed out")
		return false
	default:
		*conn, err = net.Dial("tcp", s.pubURI)
		if err != nil {
			logger.Error(s.topic, " : Failed to connect to ", s.pubURI, "- error: ", err)
			return false
		}
	}

	// 2. Write connection header
	var headers []header
	headers = append(headers, header{"topic", s.topic})
	headers = append(headers, header{"md5sum", s.msgType.MD5Sum()})
	headers = append(headers, header{"type", s.msgType.Name()})
	headers = append(headers, header{"callerid", s.nodeID})
	logger.Debug(s.topic, " : TCPROS Connection Header")
	for _, h := range headers {
		logger.Debugf("          `%s` = `%s`", h.key, h.value)
	}
	err = writeConnectionHeader(headers, *conn)
	if err != nil {
		logger.Error(s.topic, " : Failed to write connection header.")
		return false
	}

	// 3. Read reponse header
	var resHeaders []header
	resHeaders, err = readConnectionHeader(*conn)
	if err != nil {
		logger.Error(s.topic, " : Failed to read response header.")
		return false
	}
	logger.Debug(s.topic, " : TCPROS Response Header:")
	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
		logger.Debugf("          `%s` = `%s`", h.key, h.value)
	}

	// 4. Verify response header
	if resHeaderMap["type"] != s.msgType.Name() || resHeaderMap["md5sum"] != s.msgType.MD5Sum() {
		logger.Error("Incompatible message type for ", s.topic, ": ", resHeaderMap["type"], ":", s.msgType.Name(), " ", resHeaderMap["md5sum"], ":", s.msgType.MD5Sum())
		return false
	}

	// Some incomplete TCPROS implementations do not include topic name in response
	if resHeaderMap["topic"] == "" {
		resHeaderMap["topic"] = s.topic
	}

	s.event = MessageEvent{ // Event struct to be sent with each message.
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
					continue // try again!
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

// readRawMessage reads ROS message bytes from the io.Reader
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

//
// Convert errors to readResult to be handled up the callstack
//
func errorToReadResult(err error) readResult {
	if err == io.EOF {
		return remoteDisconnected
	}
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		return readTimeout
	}
	// Not sure what the cause was - return failure at this point
	return readFailed
}
