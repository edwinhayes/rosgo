package libtest_simple_action

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/edwinhayes/rosgo/actionlib"

	"github.com/edwinhayes/rosgo/ros"
)

var feedback []int32

// ActionClient

// actionClient implements a simple action client
// using callbacks
type ActionClient struct {
	node ros.Node
	name string
	ac   actionlib.SimpleActionClient
}

// New Action client instantiates a new simple action client
func newActionClient(node ros.Node, name string, actionType actionlib.ActionType) *ActionClient {

	fc := &ActionClient{
		node: node,
		ac:   actionlib.NewSimpleActionClient(node, name, actionType),
	}

	fc.ac.WaitForServer(ros.NewDuration(0, 0))
	return fc
}

// Action Client active subscriber callback
func (fc *ActionClient) activeCb() {
}

// ActionClient feedback subscriber callback
func (fc *ActionClient) feedbackCb(fb *ros.DynamicMessage) {
	feedback = fb.Data()["sequence"].([]int32)
}

// ActionClient done subscriber callback
func (fc *ActionClient) doneCb(state uint8, result *ros.DynamicMessage) {
	fc.node.Shutdown()
}

// ActionClient send goal function used to send a goal to simple action server
func (fc *ActionClient) sendGoal(goal *ros.DynamicMessage) {
	fc.ac.SendGoal(goal, fc.doneCb, fc.activeCb, fc.feedbackCb)
}

// ActionServer

// ActionServer implements a simple action server
// using the execute callback
type ActionServer struct {
	node   ros.Node
	as     actionlib.SimpleActionServer
	name   string
	fb     ros.Message
	result ros.Message
}

// newActionServer creates a new simple action server and starts it
func newActionServer(node ros.Node, name string, actionType actionlib.ActionType) {
	s := &ActionServer{
		node: node,
		name: name,
	}

	s.as = actionlib.NewSimpleActionServer(node, name,
		actionType, s.executeCallback, false)
	s.as.Start()
}

// executeCallback is the execution callback of the action server
func (s *ActionServer) executeCallback(goal *ros.DynamicMessage) {

	// instantiate action message types
	feed := s.fb.(*ros.DynamicMessage)
	result := s.result.(*ros.DynamicMessage)

	// setup sequences
	seq := feed.Data()["sequence"].([]int32)
	seq = append(seq, 0)
	seq = append(seq, 1)
	success := true

	// This method simply publishes feedback each second, incrementing count till goal acheived
	for i := 1; i < int(goal.Data()["order"].(int32)); i++ {
		if s.as.IsPreemptRequested() {
			success = false
			if err := s.as.SetPreempted(nil, ""); err != nil {
				return
			}
			break
		}

		val := seq[i] + seq[i-1]
		seq = append(seq, val)

		s.as.PublishFeedback(feed)
		time.Sleep(1000 * time.Millisecond)
	}

	// Once goal achieved, publish result
	if success {
		result.Data()["sequence"] = seq
		if err := s.as.SetSucceeded(result, "goal"); err != nil {
			return
		}
	}
}

// Spin the server node in a separate thread
func spinServer(node ros.Node, quit <-chan struct{}) {

	//Initialize server
	defer node.Shutdown()
	for {
		select {
		case <-quit:
			node.Shutdown()
			return
		default:
			node.SpinOnce()
		}
	}
}

func RTTest(t *testing.T) {

	// Create a client node
	clientNode, err := ros.NewNode("test_fibonacci_client", os.Args)
	if err != nil {
		t.Fatalf("could not create client node: %s", err)
	}
	defer clientNode.Shutdown()

	// Create a server node
	serverNode, err := ros.NewNode("test_fibonacci_server", os.Args)
	if err != nil {
		t.Fatalf("could not create server node: %s", err)
	}
	defer serverNode.Shutdown()

	// Create a dynamic action type
	actionType, err := actionlib.NewDynamicActionType("actionlib_tutorials/Fibonacci")
	if err != nil {
		t.Fatalf("could not create action type: %s", err)
	}

	// Create a new action server
	newActionServer(serverNode, "fibonacci", actionType)

	// Spin server in another thread
	quitThread := make(chan struct{})
	go spinServer(serverNode, quitThread)

	// Create a goal message for the client
	goalMsg := actionType.GoalType().NewMessage().(*ros.DynamicMessage)
	goalMsg.Data()["order"] = int32(10)

	// Create a client and send the goal to the server
	fc := newActionClient(clientNode, "fibonacci", actionType)
	fc.sendGoal(goalMsg)

	// Spin the client node
	for clientNode.OK() {

		// Check our feedback
		log.Println(feedback)

		_ = clientNode.SpinOnce()
	}
	quitThread <- struct{}{}
	// Our client ended because done was called
	t.Fatal("done")
}
