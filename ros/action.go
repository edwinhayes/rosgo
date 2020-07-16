package ros

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
	MessageType
	NewGoalMessage() ActionGoal
}

type ActionGoalIDType interface {
	MessageType
	NewGoalIDMessage() ActionGoalID
}

type ActionHeaderType interface {
	MessageType
	NewHeaderMessage() ActionHeader
}

type ActionStatusType interface {
	MessageType
	NewStatusMessage() ActionStatus
}

type ActionStatusArrayType interface {
	MessageType
	NewStatusArrayMessage() ActionStatusArray
}

type ActionFeedbackType interface {
	MessageType
	NewFeedbackMessage() ActionFeedback
}

type ActionResultType interface {
	MessageType
	NewResultMessage() ActionResult
}

type ActionGoal interface {
	Message
	GetHeader() ActionHeader
	GetGoalId() ActionGoalID
	GetGoal() Message
	SetHeader(ActionHeader)
	SetGoalId(ActionGoalID)
	SetGoal(Message)
}

type ActionGoalID interface {
	Message
	GetID() string
	SetID(string)
	GetStamp() Time
	SetStamp(Time)
}

// * ActionFeedback interface
type ActionFeedback interface {
	Message
	GetHeader() ActionHeader
	GetStatus() ActionStatus
	GetFeedback() Message
	SetHeader(ActionHeader)
	SetStatus(ActionStatus)
	SetFeedback(Message)
}

// * ActionResult interface
type ActionResult interface {
	Message
	GetHeader() ActionHeader
	GetStatus() ActionStatus
	GetResult() Message
	SetHeader(ActionHeader)
	SetStatus(ActionStatus)
	SetResult(Message)
}

// *** Shared ActionHeader interface
type ActionHeader interface {
	Message
	GetStamp() Time
	SetStamp(Time)
}

// *** Shared ActionStatus interface
type ActionStatus interface {
	Message
	GetGoalID() ActionGoalID
	SetGoalID(ActionGoalID)
	GetStatus() uint8
	SetStatus(uint8)
	GetStatusText() string
	SetStatusText(string)
}

// *** Shared ActionStatusArray interface
type ActionStatusArray interface {
	Message
	GetHeader() ActionHeader
	SetHeader(ActionHeader)
	GetStatusArray() []ActionStatus
	SetStatusArray([]ActionStatus)
}
