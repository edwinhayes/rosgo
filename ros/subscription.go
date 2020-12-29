package ros

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	modular "github.com/edwinhayes/logrus-modular"
)

//
// The subscription object runs in own go routine (start()).
// Basically handles the subscription logic and provides raw byte messages
// through the `messageChan` channel
//
type defaultSubscription struct {
	logger                 *modular.ModuleLogger
	pubURI                 string
	topic                  string
	md5sum                 string
	msgType                string
	nodeID                 string
	messageChan            chan messageEvent
	requestStopChan        chan struct{} // tell the subscription to disconnect
	remoteDisconnectedChan chan string   // tell the subscriber that the remote has disconnected
	msgTypeProper          MessageType
	event                  MessageEvent
	pool                   []byte
}

func newDefaultSubscription(logger *modular.ModuleLogger,
	pubURI string, topic string, md5sum string,
	msgType string, nodeID string,
	messageChan chan messageEvent,
	requestStopChan chan struct{},
	remoteDisconnectedChan chan string, msgTypeProper MessageType) *defaultSubscription {

	return &defaultSubscription{
		logger:                 logger,
		pubURI:                 pubURI,
		topic:                  topic,
		md5sum:                 md5sum,
		msgType:                msgType,
		nodeID:                 nodeID,
		messageChan:            messageChan,
		requestStopChan:        requestStopChan,
		remoteDisconnectedChan: remoteDisconnectedChan,
		msgTypeProper:          msgTypeProper,
	}
}

type connectionState int

const (
	publisherDisconnected connectionState = iota
	tcpOutOfSync
	connectionFailure
	stopRequested
)

type readResult int

const (
	readOk readResult = iota
	readFailed
	readTimeout
	remoteDisconnected
	readOutOfSync
)

//
// Start executing a subscription object, this is blocking and is expected to
// be run as a go routine
//
func (s *defaultSubscription) start() {
	logger := *s.logger
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
		if s.connectToPublisher(&conn) == false {
			if conn != nil {
				conn.Close()
			}
			logger.Info(s.topic, " : Could not connect to publisher, closing connection")
			return
		}
		defer conn.Close() // Make sure we close this

		// Reading from publisher
		connectionState := s.readFromPublisher(conn)

		// Under healthy conditions, we don't get here
		// handle the returned connection state

		// TCP out of sync; we will attempt to resync by closing the connection and trying again
		if connectionState == tcpOutOfSync {
			conn.Close()
			logger.Debug(s.topic, " : Connection closed, reconnecting with publisher")
		}

		// A stop was externally requested - easy one!
		if connectionState == stopRequested {
			return
		}

		// Publisher disconnected - not much we can do here, the subscription has ended
		if connectionState == publisherDisconnected {
			logger.Infof("Publisher %s on topic %s disconnected", s.pubURI, s.topic)
			s.remoteDisconnectedChan <- s.pubURI
			return
		}

		// Connection Failure is caused by read failures; the reason is uncertain, so we will give up
		if connectionState == connectionFailure {
			logger.Error(s.topic, " : Failed to read a message size")
			s.remoteDisconnectedChan <- s.pubURI
			return
		}
	}

}

//
// Estabilishes a TCPROS connection with a publishing node
// Connects via TCP and then exchanges headers to ensure
// both nodes are using the same message type
//
func (s *defaultSubscription) connectToPublisher(conn *net.Conn) bool {
	var err error

	logger := *s.logger

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
	headers = append(headers, header{"md5sum", s.md5sum})
	headers = append(headers, header{"type", s.msgType})
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
	if resHeaderMap["type"] != s.msgType || resHeaderMap["md5sum"] != s.md5sum {
		logger.Error("Incompatible message type for ", s.topic, ": ", resHeaderMap["type"], ":", s.msgType, " ", resHeaderMap["md5sum"], ":", s.md5sum)
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

//
// This function will maintain a connection with a publisher, and should normally be
// expected to loop until either the publisher or subscriber disconnects
//
func (s *defaultSubscription) readFromPublisher(conn net.Conn) connectionState {
	readingSize := true
	var msgSize int
	var buffer []byte
	var result readResult
	//
	// Subscriber loop
	// - Checks for external events
	// - Breaks the tcp serial stream into messages passed through the message channel
	//
	for {
		select {
		case <-s.requestStopChan:
			return stopRequested
		default:
			conn.SetDeadline(time.Now().Add(1000 * time.Millisecond))
			if readingSize {
				msgSize, result = readSize(conn)

				if result == readOk {
					// TODO - what if msgSize == 0?
					readingSize = false
					continue
				}

				if result == readTimeout {
					// TODO: This is pretty shaky... what if we only got a portion of the size bytes?
					//       I think we can do better
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
					// We just failed to read a message; it is likely that we are now out of sync
					return tcpOutOfSync
				}
			}

			// Handle read result cases
			if result == readOutOfSync {
				return tcpOutOfSync
			}
			if result == readFailed {
				return connectionFailure
			}
			if result == remoteDisconnected {
				return publisherDisconnected
			}
		}
	}
}

//
// Read the number of bytes to expect in a ROS message payload
//
// The structure of a ROS message is: [SIZE|PAYLOAD]
// This function reads the 4-byte size of a message
//
func readSize(r io.Reader) (int, readResult) {
	var msgSize uint32

	err := binary.Read(r, binary.LittleEndian, &msgSize)
	if err != nil {
		return 0, errorToReadResult(err)
	}
	// Check reasonable buffer size
	if msgSize < 256000000 {
		return int(msgSize), readOk
	} else {
		// We assume that this many bytes means we are out of sync
		return 0, readOutOfSync
	}
}

//
// Read the raw ROS message payload
//
func (s *defaultSubscription) readRawMessage(r io.Reader, size int) ([]byte, readResult) {
	// first assign a buffer from our pool
	// The pool is reallocated if it is too small
	if len(s.pool) < size {
		s.pool = make([]byte, size)
	}
	buffer := s.pool[:size]

	// Read the full buffer; we expect this call to timeout if it takes too long
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
