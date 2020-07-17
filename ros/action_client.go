package ros

import (
	"fmt"
	"sync"

	modular "github.com/edwinhayes/logrus-modular"
)

type defaultActionClient struct {
	started          bool
	node             Node
	action           string
	actionType       ActionType
	actionResult     MessageType
	actionResultType MessageType
	actionFeedback   MessageType
	actionGoal       MessageType
	goalPub          Publisher
	cancelPub        Publisher
	resultSub        Subscriber
	feedbackSub      Subscriber
	statusSub        Subscriber
	logger           *modular.ModuleLogger
	handlers         []*clientGoalHandler
	handlersMutex    sync.RWMutex
	goalIDGen        *goalIDGenerator
	statusReceived   bool
	callerID         string
}

func newDefaultActionClient(node Node, action string, actType ActionType) *defaultActionClient {
	ac := &defaultActionClient{
		node:           node,
		action:         action,
		actionType:     actType,
		actionResult:   actType.ResultType(),
		actionFeedback: actType.FeedbackType(),
		actionGoal:     actType.GoalType(),
		logger:         node.Logger(),
		statusReceived: false,
		goalIDGen:      newGoalIDGenerator(node.Name()),
	}

	// Create goal publisher
	ac.goalPub, _ = node.NewPublisher(fmt.Sprintf("%s/goal", action), actType.GoalType())
	// Create cancel publisher
	ac.cancelPub, _ = node.NewPublisher(fmt.Sprintf("%s/cancel", action), NewActionGoalIDType())
	// Create result subscriber
	ac.resultSub, _ = node.NewSubscriber(fmt.Sprintf("%s/result", action), actType.ResultType(), ac.internalResultCallback)
	// Create feedback subscriber
	ac.feedbackSub, _ = node.NewSubscriber(fmt.Sprintf("%s/feedback", action), actType.FeedbackType(), ac.internalFeedbackCallback)
	// Create status subscriber
	ac.statusSub, _ = node.NewSubscriber(fmt.Sprintf("%s/status", action), NewActionStatusArrayType(), ac.internalStatusCallback)

	return ac
}

func (ac *defaultActionClient) SendGoal(goal Message, transitionCb, feedbackCb interface{}) ClientGoalHandler {
	logger := *ac.logger
	if !ac.started {
		logger.Error("[ActionClient] Trying to send a goal on an inactive ActionClient")
	}

	ag := ac.actionType.GoalType().NewGoalMessage().(*DynamicActionGoal)
	// make a goalId message with timestamp and generated id
	//goalid := ag.GetGoalId()
	goalid := NewActionGoalIDType().NewGoalIDMessage()
	goalid.SetStamp(Now())
	goalid.SetID(ac.goalIDGen.generateID())
	// make a header with timestamp

	ag.SetGoal(goal)
	ag.SetGoalId(goalid)
	ag.SetHeader(NewActionHeader())

	ac.PublishActionGoal(ag)

	handler := newClientGoalHandler(ac, ag, transitionCb, feedbackCb)

	ac.handlersMutex.Lock()
	ac.handlers = append(ac.handlers, handler)
	ac.handlersMutex.Unlock()

	return handler
}

func (ac *defaultActionClient) CancelAllGoals() {
	logger := *ac.logger
	if !ac.started {
		logger.Error("[ActionClient] Trying to cancel goals on an inactive ActionClient")
		return
	}
	// Create a goal id message
	goalid := NewActionGoalIDType().NewGoalIDMessage()
	ac.cancelPub.Publish(goalid)
}

func (ac *defaultActionClient) CancelAllGoalsBeforeTime(stamp Time) {
	logger := *ac.logger
	if !ac.started {
		logger.Error("[ActionClient] Trying to cancel goals on an inactive ActionClient")
		return
	}
	// Create a goal id message using timestamp
	goalid := NewActionGoalIDType().NewGoalIDMessage()
	goalid.SetStamp(stamp)
	ac.cancelPub.Publish(goalid)
}

func (ac *defaultActionClient) Shutdown() {
	ac.handlersMutex.Lock()
	defer ac.handlersMutex.Unlock()

	ac.started = false
	for _, h := range ac.handlers {
		h.Shutdown(false)
	}

	ac.handlers = nil
	ac.node.Shutdown()
}

func (ac *defaultActionClient) PublishActionGoal(ag ActionGoal) {
	if ac.started {
		ac.goalPub.Publish(ag)
	}
}

func (ac *defaultActionClient) PublishCancel(cancel *DynamicMessage) {
	if ac.started {
		ac.cancelPub.Publish(cancel)
	}
}

func (ac *defaultActionClient) WaitForServer(timeout Duration) bool {
	logger := *ac.logger
	started := false
	logger.Info("[ActionClient] Waiting action server to start")
	rate := CycleTime(NewDuration(0, 10000000))
	waitStart := Now()

LOOP:
	for !started {
		gSubs := ac.goalPub.GetNumSubscribers()
		cSubs := ac.cancelPub.GetNumSubscribers()
		fPubs := ac.feedbackSub.GetNumPublishers()
		rPubs := ac.resultSub.GetNumPublishers()
		sPubs := ac.statusSub.GetNumPublishers()
		started = (gSubs > 0 && cSubs > 0 && fPubs > 0 && rPubs > 0 && sPubs > 0)

		now := Now()
		diff := now.Diff(waitStart)
		if !timeout.IsZero() && diff.Cmp(timeout) >= 0 {
			break LOOP
		}

		rate.Sleep()
	}

	if started {
		ac.started = started
	}

	return started
}

func (ac *defaultActionClient) DeleteGoalHandler(gh *clientGoalHandler) {
	ac.handlersMutex.Lock()
	defer ac.handlersMutex.Unlock()

	for i, h := range ac.handlers {
		if h == gh {
			ac.handlers[i] = ac.handlers[len(ac.handlers)-1]
			ac.handlers[len(ac.handlers)-1] = nil
			ac.handlers = ac.handlers[:len(ac.handlers)-1]
		}
	}
}

// Internal Result Callback for Result subscriber
func (ac *defaultActionClient) internalResultCallback(result interface{}, event MessageEvent) {
	logger := *ac.logger
	ac.handlersMutex.RLock()
	defer ac.handlersMutex.RUnlock()

	// Interface to ActionResult
	results := ac.actionType.ResultType().(*DynamicActionResultType).NewResultMessageFromInterface(result)

	for _, h := range ac.handlers {
		if err := h.updateResult(results); err != nil {
			logger.Error(err)
		}
	}
}

// Itnernal Feedback Callback for Feedback subscriber
func (ac *defaultActionClient) internalFeedbackCallback(feedback interface{}, event MessageEvent) {
	ac.handlersMutex.RLock()
	defer ac.handlersMutex.RUnlock()

	// Interface to ActionFeedback
	feed := ac.actionType.FeedbackType().(*DynamicActionFeedbackType).NewFeedbackMessageFromInterface(feedback)

	for _, h := range ac.handlers {
		h.updateFeedback(feed)
	}
}

// Internal Status Callback for status subscriber
func (ac *defaultActionClient) internalStatusCallback(statusArr interface{}, event MessageEvent) {
	logger := *ac.logger
	ac.handlersMutex.RLock()
	defer ac.handlersMutex.RUnlock()

	if !ac.statusReceived {
		ac.statusReceived = true
		logger.Debug("Recieved first status message from action server ")
	} else if ac.callerID != event.PublisherName {
		logger.Debug("Previously received status from %s, now from %s. Did the action server change", ac.callerID, event.PublisherName)
	}

	// Interface to status array conversion
	statusArray := NewActionStatusArrayType().(*DynamicActionStatusArrayType).NewStatusArrayFromInterface(statusArr)

	ac.callerID = event.PublisherName
	for _, h := range ac.handlers {
		if err := h.updateStatus(statusArray); err != nil {
			logger.Error(err)
		}
	}
}
