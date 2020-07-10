package actionlib

import (
	"github.com/edwinhayes/rosgo/ros"
)

type ActionType interface {
	MD5Sum() string
	Name() string
	GoalType() ros.MessageType
	FeedbackType() ros.MessageType
	ResultType() ros.MessageType
	NewAction() Action
}

type Action interface {
	GetActionGoal() ActionGoal
	GetActionFeedback() ActionFeedback
	GetActionResult() ActionResult
}

type ActionGoal interface {
	ros.Message
	GetHeader() *ros.DynamicMessage
	GetGoalId() *ros.DynamicMessage
	GetGoal() ros.Message
	SetHeader(*ros.DynamicMessage)
	SetGoalId(*ros.DynamicMessage)
	SetGoal(ros.Message)
}

type ActionFeedback interface {
	ros.Message
	GetHeader() *ros.DynamicMessage
	GetStatus() *ros.DynamicMessage
	GetFeedback() ros.Message
	SetHeader(*ros.DynamicMessage)
	SetStatus(*ros.DynamicMessage)
	SetFeedback(ros.Message)
}

type ActionResult interface {
	ros.Message
	GetHeader() *ros.DynamicMessage
	GetStatus() *ros.DynamicMessage
	GetResult() ros.Message
	SetHeader(*ros.DynamicMessage)
	SetStatus(*ros.DynamicMessage)
	SetResult(ros.Message)
}
