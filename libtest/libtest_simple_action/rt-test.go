package libtest_simple_action

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/edwinhayes/rosgo/ros"
)

var feedback []int32

// ActionClient

// actionClient implements a simple action client
// using callbacks
type ActionClient struct {
	node ros.Node
	name string
	ac   ros.SimpleActionClient
}

// New Action client instantiates a new simple action client
func newActionClient(node ros.Node, name string, actionType ros.ActionType) *ActionClient {

	fc := &ActionClient{
		node: node,
		ac:   ros.NewSimpleActionClient(node, name, actionType),
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
	log.Printf("DONE CALLBACK CALLED\n")
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
	as     ros.SimpleActionServer
	name   string
	fb     ros.Message
	result ros.Message
}

// newActionServer creates a new simple action server and starts it
func newActionServer(node ros.Node, name string, actionType ros.ActionType) {
	s := &ActionServer{
		node: node,
		name: name,
	}

	s.as = ros.NewSimpleActionServer(node, name,
		actionType, s.executeCallback, false)
	s.as.Start()
}

// executeCallback is the execution callback of the action server
func (s *ActionServer) executeCallback(goali interface{}, actionType interface{}) {

	fmt.Printf("Received in execute callback: %v\n", goali)
	goalmsg := goali.(*ros.DynamicMessage)
	fmt.Printf("goalmsg data in internal goal callback: %v\n", goalmsg)
	// To Do : Replace this with correct interface usage
	goalid := goalmsg.Data()["goal_id"].(ros.ActionGoalID)
	header := goalmsg.Data()["header"].(ros.ActionHeader)
	goalType := actionType.(ros.ActionType).GoalType()
	goal := goalType.NewGoalMessage()
	goal.SetGoal(goalmsg)
	goal.SetGoalId(goalid)
	goal.SetHeader(header)
	// instantiate action message types
	feedType := actionType.(ros.ActionType).FeedbackType()
	resultType := actionType.(ros.ActionType).ResultType()

	// setup sequences
	feed := feedType.NewFeedbackMessage().(*ros.DynamicActionFeedback)
	//feed := feedMsg.(*ros.DynamicMessage)
	seq := feed.Data()["sequence"].([]int32)
	seq = append(seq, 0)
	seq = append(seq, 1)
	success := true

	//goalData := goal.GetGoal().(*ros.DynamicMessage)

	// This method simply publishes feedback each second, incrementing count till goal acheived
	for i := 1; i < int(goalmsg.Data()["order"].(int32)); i++ {
		if s.as.IsPreemptRequested() {
			success = false
			if err := s.as.SetPreempted(nil, ""); err != nil {
				return
			}
			break
		}

		val := seq[i] + seq[i-1]
		seq = append(seq, val)
		feed.Data()["sequence"] = seq
		s.as.PublishFeedback(feed)
		time.Sleep(1000 * time.Millisecond)
	}

	// Once goal achieved, publish result
	if success {
		resultMsg := resultType.NewResultMessage().(*ros.DynamicActionResult)
		resultMsg.Data()["sequence"] = seq
		if err := s.as.SetSucceeded(resultMsg, "goal"); err != nil {
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
	actionType, err := ros.NewDynamicActionType("actionlib_tutorials/Fibonacci")
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
		//log.Println(feedback)

		_ = clientNode.SpinOnce()
	}
	quitThread <- struct{}{}
	// Our client ended because done was called
	t.Fatal("done")
}
