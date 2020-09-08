package ros

func NewActionClient(node Node, action string, actionType ActionType) (ActionClient, error) {
	return newDefaultActionClient(node, action, actionType)
}

func NewActionServer(node Node, action string, actionType ActionType, goalCb, cancelCb interface{}, autoStart bool) ActionServer {
	return newDefaultActionServer(node, action, actionType, goalCb, cancelCb, autoStart)
}

func NewSimpleActionClient(node Node, action string, actionType ActionType) (SimpleActionClient, error) {
	return newSimpleActionClient(node, action, actionType)
}

func NewSimpleActionServer(node Node, action string, actionType ActionType, executeCb interface{}, autoStart bool) SimpleActionServer {
	return newSimpleActionServer(node, action, actionType, executeCb, autoStart)
}

func NewServerGoalHandlerWithGoal(as ActionServer, goal ActionGoal) ServerGoalHandler {
	return newServerGoalHandlerWithGoal(as, goal)
}

func NewServerGoalHandlerWithGoalId(as ActionServer, goalID ActionGoalID) ServerGoalHandler {
	return newServerGoalHandlerWithGoalId(as, goalID)
}

type ActionClient interface {
	WaitForServer(timeout Duration) bool
	SendGoal(goal Message, transitionCallback interface{}, feedbackCallback interface{}, goalID string) (ClientGoalHandler, error)
	CancelAllGoals()
	CancelAllGoalsBeforeTime(stamp Time)
}

type ActionServer interface {
	Start()
	Shutdown()
	PublishResult(status ActionStatus, result Message)
	PublishFeedback(status ActionStatus, feedback Message)
	PublishStatus()
	RegisterGoalCallback(interface{})
	RegisterCancelCallback(interface{})
}

type SimpleActionClient interface {
	SendGoal(goal Message, doneCb, activeCb, feedbackCb interface{}, goalID string) error
	SendGoalAndWait(goal Message, executeTimeout, preeptTimeout Duration) (uint8, error)
	WaitForServer(timeout Duration) bool
	WaitForResult(timeout Duration) bool
	GetResult() (Message, error)
	GetState() (uint8, error)
	GetGoalStatusText() (string, error)
	CancelAllGoals()
	CancelAllGoalsBeforeTime(stamp Time)
	CancelGoal() error
	ShutdownClient(bool, bool, bool)
	StopTrackingGoal()
}

type SimpleActionServer interface {
	Start()
	IsNewGoalAvailable() bool
	IsPreemptRequested() bool
	IsActive() bool
	SetSucceeded(result Message, text string) error
	SetAborted(result Message, text string) error
	SetPreempted(result Message, text string) error
	AcceptNewGoal() (Message, error)
	PublishFeedback(feedback Message)
	GetDefaultResult() Message
	RegisterGoalCallback(callback interface{}) error
	RegisterPreemptCallback(callback interface{})
}

type ClientGoalHandler interface {
	IsExpired() bool
	GetCommState() (CommState, error)
	GetGoalStatus() (uint8, error)
	GetGoalStatusText() (string, error)
	GetTerminalState() (uint8, error)
	GetResult() (Message, error)
	Resend() error
	Cancel() error
}

type ServerGoalHandler interface {
	SetAccepted(string) error
	SetCancelled(Message, string) error
	SetRejected(Message, string) error
	SetAborted(Message, string) error
	SetSucceeded(Message, string) error
	SetCancelRequested() bool
	PublishFeedback(Message)
	GetGoal() Message
	GetGoalId() ActionGoalID
	GetGoalStatus() ActionStatus
	Equal(ServerGoalHandler) bool
	NotEqual(ServerGoalHandler) bool
	Hash() uint32
	GetHandlerDestructionTime() Time
	SetHandlerDestructionTime(Time)
}
