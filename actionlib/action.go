package actionlib

import (
	"github.com/edwinhayes/rosgo/ros"
)

type ActionType interface {
	MD5Sum() string
	Name() string
	GoalType() ActionGoalType
	FeedbackType() ActionFeedbackType
	ResultType() ActionResultType
	NewAction() Action
}

type Action interface {
	GetActionGoal() ActionGoal
	GetActionFeedback() ActionFeedback
	GetActionResult() ActionResult
}

type ActionGoalType interface {
	ros.MessageType
	NewGoalMessage() ActionGoal
}

type ActionGoalIDType interface {
	ros.MessageType
	NewGoalIDMessage() ActionGoalID
}

type ActionHeaderType interface {
	ros.MessageType
	NewHeaderMessage() ActionHeader
}

type ActionStatusType interface {
	ros.MessageType
	NewStatusMessage() ActionStatus
}

type ActionStatusArrayType interface {
	ros.MessageType
	NewStatusArrayMessage() ActionStatusArray
}

type ActionFeedbackType interface {
	ros.MessageType
	NewFeedbackMessage() ActionFeedback
}

type ActionResultType interface {
	ros.MessageType
	NewResultMessage() ActionResult
}

type ActionGoal interface {
	ros.Message
	GetHeader() ActionHeader
	GetGoalId() ActionGoalID
	GetGoal() ros.Message
	SetHeader(ActionHeader)
	SetGoalId(ActionGoalID)
	SetGoal(ros.Message)
}

type ActionGoalID interface {
	ros.Message
	GetID() string
	SetID(string)
	GetStamp() ros.Time
	SetStamp(ros.Time)
}

// * ActionFeedback interface
type ActionFeedback interface {
	ros.Message
	GetHeader() ActionHeader
	GetStatus() ActionStatus
	GetFeedback() ros.Message
	SetHeader(ActionHeader)
	SetStatus(ActionStatus)
	SetFeedback(ros.Message)
}

// * ActionResult interface
type ActionResult interface {
	ros.Message
	GetHeader() ActionHeader
	GetStatus() ActionStatus
	GetResult() ros.Message
	SetHeader(ActionHeader)
	SetStatus(ActionStatus)
	SetResult(ros.Message)
}

// *** Shared ActionHeader interface
type ActionHeader interface {
	ros.Message
	GetStamp() ros.Time
	SetStamp(ros.Time)
}

// *** Shared ActionStatus interface
type ActionStatus interface {
	ros.Message
	GetGoalID() ActionGoalID
	SetGoalID(ActionGoalID)
	GetStatus() uint8
	SetStatus(uint8)
	GetStatusText() string
	SetStatusText(string)
}

// *** Shared ActionStatusArray interface
type ActionStatusArray interface {
	ros.Message
	GetHeader() ActionHeader
	SetHeader(ActionHeader)
	GetStatusArray() []ActionStatus
	SetStatusArray([]ActionStatus)
}
