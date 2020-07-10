package actionlib

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/edwinhayes/rosgo/ros"
)

type defaultActionServer struct {
	node             ros.Node
	autoStart        bool
	started          bool
	action           string
	actionType       ActionType
	actionResult     ros.MessageType
	actionResultType ros.MessageType
	actionFeedback   ros.MessageType
	actionGoal       ros.MessageType
	statusMutex      sync.RWMutex
	statusFrequency  ros.Rate
	statusTimer      *time.Ticker
	handlers         map[string]*serverGoalHandler
	handlersTimeout  ros.Duration
	handlersMutex    sync.Mutex
	goalCallback     interface{}
	cancelCallback   interface{}
	lastCancel       ros.Time
	pubQueueSize     int
	subQueueSize     int
	goalSub          ros.Subscriber
	cancelSub        ros.Subscriber
	resultPub        ros.Publisher
	feedbackPub      ros.Publisher
	statusPub        ros.Publisher
	statusPubChan    chan struct{}
	goalIDGen        *goalIDGenerator
	shutdownChan     chan struct{}
}

func newDefaultActionServer(node ros.Node, action string, actType ActionType, goalCb interface{}, cancelCb interface{}, start bool) *defaultActionServer {
	return &defaultActionServer{
		node:            node,
		autoStart:       start,
		started:         false,
		action:          action,
		actionType:      actType,
		actionResult:    actType.ResultType(),
		actionFeedback:  actType.FeedbackType(),
		actionGoal:      actType.GoalType(),
		handlersTimeout: ros.NewDuration(60, 0),
		goalCallback:    goalCb,
		cancelCallback:  cancelCb,
		lastCancel:      ros.Now(),
	}
}

func (as *defaultActionServer) init() {
	as.statusPubChan = make(chan struct{}, 10)
	as.shutdownChan = make(chan struct{}, 10)

	// setup goal id generator and goal handlers
	as.goalIDGen = newGoalIDGenerator(as.node.Name())
	as.handlers = map[string]*serverGoalHandler{}

	// setup action result type so that we can create default result messages
	res := as.actionResult.NewMessage().(ActionResult).GetResult()
	as.actionResultType = res.Type()

	// get frequency from ros params
	as.statusFrequency = ros.NewRate(5.0)

	// get queue sizes from ros params
	// queue sizes not implemented by ros.Node yet
	as.pubQueueSize = 50
	as.subQueueSize = 50

	// Create goal subscription
	as.goalSub, _ = as.node.NewSubscriber(fmt.Sprintf("%s/goal", as.action), as.actionType.GoalType(), as.internalGoalCallback)
	// Create a cancel subscription
	cancelMsgType, _ := ros.NewDynamicMessageType("actionlib_msgs/GoalID")
	as.cancelSub, _ = as.node.NewSubscriber(fmt.Sprintf("%s/cancel", as.action), cancelMsgType, as.internalCancelCallback)
	// Create result publisher
	as.resultPub, _ = as.node.NewPublisher(fmt.Sprintf("%s/result", as.action), as.actionType.ResultType())
	// Create feedback publisher
	as.feedbackPub, _ = as.node.NewPublisher(fmt.Sprintf("%s/feedback", as.action), as.actionType.FeedbackType())
	// Create Status publisher
	statusMsgType, _ := ros.NewDynamicMessageType("actionlib_msgs/GoalStatusArray")
	as.statusPub, _ = as.node.NewPublisher(fmt.Sprintf("%s/status", as.action), statusMsgType)
}

func (as *defaultActionServer) Start() {
	logger := *as.node.Logger()
	defer func() {
		logger.Debug("defaultActionServer.start exit")
		as.started = false
	}()

	// initialize subscribers and publishers
	as.init()

	// start status publish ticker that notifies at 5hz
	as.statusTimer = time.NewTicker(time.Second / 5.0)
	defer as.statusTimer.Stop()

	as.started = true

	for {
		select {
		case <-as.shutdownChan:
			return

		case <-as.statusTimer.C:
			as.PublishStatus()

		case <-as.statusPubChan:
			arr := as.getStatus()
			as.statusPub.Publish(arr)
		}
	}
}

// PublishResult publishes action result message
func (as *defaultActionServer) PublishResult(status *ros.DynamicMessage, result ros.Message) {
	msg := as.actionResult.NewMessage().(ActionResult)
	// Create a header message with time stamp
	headerMsgType, _ := ros.NewDynamicMessageType("std_msgs/Header")
	headerMsg := headerMsgType.NewMessage().(*ros.DynamicMessage)
	headerMsg.Data()["stamp"] = ros.Now()
	msg.SetHeader(headerMsg)
	msg.SetStatus(status)
	msg.SetResult(result)
	as.resultPub.Publish(msg)
}

// PublishFeedback publishes action feedback messages
func (as *defaultActionServer) PublishFeedback(status *ros.DynamicMessage, feedback ros.Message) {
	msg := as.actionFeedback.NewMessage().(ActionFeedback)
	// Create a header message with time stamp
	headerMsgType, _ := ros.NewDynamicMessageType("std_msgs/Header")
	headerMsg := headerMsgType.NewMessage().(*ros.DynamicMessage)
	headerMsg.Data()["stamp"] = ros.Now()
	msg.SetHeader(headerMsg)
	msg.SetStatus(status)
	msg.SetFeedback(feedback)
	as.feedbackPub.Publish(msg)
}

func (as *defaultActionServer) getStatus() *ros.DynamicMessage {
	as.handlersMutex.Lock()
	defer as.handlersMutex.Unlock()
	var statusList []*ros.DynamicMessage

	if as.node.OK() {
		for id, gh := range as.handlers {
			handlerTime := gh.GetHandlerDestructionTime()
			destroyTime := handlerTime.Add(as.handlersTimeout)

			if !handlerTime.IsZero() && destroyTime.Cmp(ros.Now()) <= 0 {
				delete(as.handlers, id)
				continue
			}

			statusList = append(statusList, gh.GetGoalStatus())
		}
	}
	// Create a goal status array message
	statusArrayType, _ := ros.NewDynamicMessageType("actionlib_msgs/GoalStatusArray")
	statusArrayMsg := statusArrayType.NewMessage().(*ros.DynamicMessage)

	// Create a header message with time stamp
	headerMsgType, _ := ros.NewDynamicMessageType("std_msgs/Header")
	headerMsg := headerMsgType.NewMessage().(*ros.DynamicMessage)
	headerMsg.Data()["stamp"] = ros.Now()
	statusArrayMsg.Data()["header"] = headerMsg

	// Add status list
	statusArrayMsg.Data()["status_list"] = statusList
	return statusArrayMsg
}

func (as *defaultActionServer) PublishStatus() {
	as.statusPubChan <- struct{}{}
}

// internalCancelCallback recieves cancel message from client
func (as *defaultActionServer) internalCancelCallback(goalID *ros.DynamicMessage, event ros.MessageEvent) {
	as.handlersMutex.Lock()
	defer as.handlersMutex.Unlock()

	goalFound := false
	logger := *as.node.Logger()
	logger.Debug("Action server has received a new cancel request")

	for id, gh := range as.handlers {
		cancelAll := (goalID.Data()["id"].(string) == "" && goalID.Data()["stamp"].(*ros.Time).IsZero())
		cancelCurrent := (goalID.Data()["id"].(string) == id)

		st := gh.GetGoalStatus()
		cancelBeforeStamp := (!goalID.Data()["stamp"].(*ros.Time).IsZero() && st.Data()["goal_id"].(*ros.DynamicMessage).Data()["stamp"].(*ros.Time).Cmp(goalID.Data()["stamp"].(ros.Time)) <= 0)

		if cancelAll || cancelCurrent || cancelBeforeStamp {
			if goalID.Data()["id"].(string) == st.Data()["goal_id"].(*ros.DynamicMessage).Data()["id"].(string) {
				goalFound = true
			}

			if gh.SetCancelRequested() {
				args := []reflect.Value{reflect.ValueOf(goalID)}
				fun := reflect.ValueOf(as.cancelCallback)
				numArgsNeeded := fun.Type().NumIn()

				if numArgsNeeded <= 1 {
					fun.Call(args[0:numArgsNeeded])
				}
			}
		}
	}

	if goalID.Data()["id"].(string) != "" && !goalFound {
		gh := newServerGoalHandlerWithGoalId(as, goalID)
		as.handlers[goalID.Data()["id"].(string)] = gh
		gh.SetHandlerDestructionTime(ros.Now())
	}

	if goalID.Data()["stamp"].(*ros.Time).Cmp(as.lastCancel) > 0 {
		as.lastCancel = goalID.Data()["stamp"].(ros.Time)
	}
}

// internalGoalCallback recieves the goals from client and checks if
// the goalID already exists in the status list. If not, it will call
// server's goalCallback with goal that was recieved from the client.
func (as *defaultActionServer) internalGoalCallback(goal ActionGoal, event ros.MessageEvent) {
	as.handlersMutex.Lock()
	defer as.handlersMutex.Unlock()

	logger := *as.node.Logger()
	goalID := goal.GetGoalId()

	for id, gh := range as.handlers {
		if goalID.Data()["id"].(string) == id {
			st := gh.GetGoalStatus()
			logger.Debugf("Goal %s was already in the status list with status %+v", goalID.Data()["id"].(string), st.Data()["status"].(uint8))
			if st.Data()["status"].(uint8) == uint8(7) {
				st.Data()["status"] = uint8(8)
				result := as.actionResultType.NewMessage()
				as.PublishResult(st, result)
			}

			gh.SetHandlerDestructionTime(ros.Now())
			return
		}
	}

	id := goalID.Data()["id"].(string)
	if len(id) == 0 {
		id = as.goalIDGen.generateID()
		// Create goal id message with id and time stamp
		goalMsgType, _ := ros.NewDynamicMessageType("actionlib_msgs/GoalID")
		goalMsg := goalMsgType.NewMessage().(*ros.DynamicMessage)
		goalMsg.Data()["id"] = id
		goalMsg.Data()["stamp"] = goalID.Data()["stamp"].(string)
		// Set goal id
		goal.SetGoalId(goalMsg)
	}

	gh := newServerGoalHandlerWithGoal(as, goal)
	as.handlers[id] = gh
	if !goalID.Data()["stamp"].(*ros.Time).IsZero() && goalID.Data()["stamp"].(*ros.Time).Cmp(as.lastCancel) <= 0 {
		gh.SetCancelled(nil, "timestamp older than last goal cancel")
		return
	}

	args := []reflect.Value{reflect.ValueOf(goal), reflect.ValueOf(event)}
	fun := reflect.ValueOf(as.goalCallback)
	numArgsNeeded := fun.Type().NumIn()

	if numArgsNeeded <= 1 {
		fun.Call(args[0:numArgsNeeded])
	}
}

func (as *defaultActionServer) getHandler(id string) *serverGoalHandler {
	handler := as.handlers[id]
	return handler
}

// RegisterGoalCallback replaces existing goal callback function with newly
// provided goal callback function.
func (as *defaultActionServer) RegisterGoalCallback(goalCb interface{}) {
	as.goalCallback = goalCb
}

func (as *defaultActionServer) RegisterCancelCallback(cancelCb interface{}) {
	as.cancelCallback = cancelCb
}

func (as *defaultActionServer) Shutdown() {
	as.shutdownChan <- struct{}{}
}
