package ros

type ActionType interface {
	MD5Sum() string
	Name() string
	GoalType() ActionGoalType
	FeedbackType() ActionFeedbackType
	ResultType() ActionResultType
	NewAction() Action
}

type ActionGoalType interface {
	MessageType
	NewGoalMessage() ActionGoal
}

type ActionFeedbackType interface {
	MessageType
	NewFeedbackMessage() ActionFeedback
}

type ActionResultType interface {
	MessageType
	NewResultMessage() ActionResult
}

type ActionGoalIDType interface {
	MessageType
	NewGoalIDMessage() ActionGoalID
}

type ActionStatusType interface {
	MessageType
	NewStatusMessage() ActionStatus
}

type ActionStatusArrayType interface {
	MessageType
	NewStatusArrayMessage() ActionStatusArray
}

type Action interface {
	GetActionGoal() ActionGoal
	GetActionFeedback() ActionFeedback
	GetActionResult() ActionResult
}

type ActionGoal interface {
	Message
	GetHeader() (Message, error)
	GetGoalId() (ActionGoalID, error)
	GetGoal() (Message, error)
	SetHeader(Message)
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
	GetHeader() Message
	GetStatus() ActionStatus
	GetFeedback() Message
	SetHeader(Message)
	SetStatus(ActionStatus)
	SetFeedback(Message)
}

// * ActionResult interface
type ActionResult interface {
	Message
	GetHeader() Message
	GetStatus() ActionStatus
	GetResult() Message
	SetHeader(Message)
	SetStatus(ActionStatus)
	SetResult(Message)
}

// *** Shared ActionHeader interface
type ActionHeader interface {
	Message
	GetStamp() Time
	SetStamp(Time)
	GetFrameID() string
	SetFrameID(string)
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
	GetHeader() Message
	SetHeader(Message)
	GetStatusArray() []ActionStatus
	SetStatusArray([]ActionStatus)
}
