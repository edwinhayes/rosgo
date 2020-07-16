package ros

// IMPORT REQUIRED PACKAGES.

import (
	"strings"

	"github.com/edwinhayes/rosgo/libgengo"
	"github.com/pkg/errors"
)

// DEFINE PUBLIC STRUCTURES.

// DynamicActionType abstracts the schema of a ROS Action whose schema is only known at runtime.
// DynamicActionType implements the rosgo actionlib.ActionType interface, allowing it to be used throughout rosgo in the same manner as action schemas generated
// at compiletime by gengo.
type DynamicActionType struct {
	name         string
	md5sum       string
	text         string
	goalType     ActionGoalType
	feedbackType ActionFeedbackType
	resultType   ActionResultType
}

// DynamicAction abstracts an instance of a ROS Action whose type is only known at runtime.  The schema of the action is denoted by the referenced DynamicActionType, while the
// Goal, Feedback and Result references the rosgo Messages it implements.  DynamicAction implements the rosgo actionlib.Action interface, allowing
// it to be used throughout rosgo in the same manner as action types generated at compiletime by gengo.
type DynamicAction struct {
	dynamicType *DynamicActionType
	Goal        ActionGoal
	Feedback    ActionFeedback
	Result      ActionResult
}

// DynamicAction* interfaces.
type DynamicActionGoal struct{ DynamicMessage }
type DynamicActionFeedback struct{ DynamicMessage }
type DynamicActionResult struct{ DynamicMessage }

// DynamicActionType* type interfaces
type DynamicActionGoalType struct{ DynamicMessageType }
type DynamicActionFeedbackType struct{ DynamicMessageType }
type DynamicActionResultType struct{ DynamicMessageType }

// DynamicAction** shared interfaces.
type DynamicActionGoalID struct{ DynamicMessage }
type DynamicActionHeader struct{ DynamicMessage }
type DynamicActionStatus struct{ DynamicMessage }
type DynamicActionStatusArray struct{ DynamicMessage }

// DynamicActionType** shared type interfaces
type DynamicActionGoalIDType struct{ DynamicMessageType }
type DynamicActionHeaderType struct{ DynamicMessageType }
type DynamicActionStatusType struct{ DynamicMessageType }
type DynamicActionStatusArrayType struct{ DynamicMessageType }

// DEFINE PRIVATE STRUCTURES.

// DEFINE PUBLIC GLOBALS.

// DEFINE PRIVATE GLOBALS.

// NewDynamicActionType generates a DynamicActionType corresponding to the specified typeName from the available ROS action definitions; typeName should be a fully-qualified
// ROS action type name.  The first time the function is run, a message/service/action 'context' is created by searching through the available ROS definitions, then the ROS action to
// be used for the definition is looked up by name.  On subsequent calls, the ROS action type is looked up directly from the existing context.
func NewDynamicActionType(typeName string) (*DynamicActionType, error) {
	return newDynamicActionTypeNested(typeName, "")
}

// newDynamicActionTypeNested generates a DynamicActionType from the available ROS definitions.  The first time the function is run, a message/service/action 'context' is created by
// searching through the available ROS definitions, then the ROS action type to use for the defintion is looked up by name.  On subsequent calls, the ROS action type
// is looked up directly from the existing context.  This 'nested' version of the function is able to be called recursively, where packageName should be the typeName of the
// parent ROS action; this is used internally for handling complex ROS action.
func newDynamicActionTypeNested(typeName string, packageName string) (*DynamicActionType, error) {
	// Create an empty action type.
	m := new(DynamicActionType)

	// Create context for our ROS install.
	c, err := libgengo.NewPkgContext(strings.Split(GetRuntimePackagePath(), ":"))
	if err != nil {
		return nil, err
	}
	context = c

	// We need to try to look up the full name, in case we've just been given a short name.
	fullname := typeName
	_, ok := context.GetActions()[fullname]
	if !ok {
		// Messages in the same package are allowed to use relative names, so try prefixing the package.
		if packageName != "" {
			fullname = packageName + "/" + fullname
		}
	}

	// Load context for the target message.
	spec, err := context.LoadAction(fullname)
	if err != nil {
		return nil, err
	}

	// Now we know all about the service!
	m.name = spec.ShortName
	m.md5sum = spec.MD5Sum
	m.text = spec.Text

	// Create Dynamic Goal Type
	goal, err := NewDynamicActionGoalType(spec.Goal.FullName)
	if err != nil {
		return nil, err
	}
	m.goalType = goal

	// Create Dynamic Feedback Type
	feedback := DynamicActionFeedbackType{}
	feedbackType, err := NewDynamicMessageTypeLiteral(spec.Feedback.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	feedback.DynamicMessageType = feedbackType
	m.feedbackType = &feedback

	// Create Dynamic Result Type
	result, err := NewDynamicActionResultType(spec.Result.FullName)
	if err != nil {
		return nil, err
	}
	m.resultType = result
	// We've successfully made a new service type matching the requested ROS type.
	return m, nil

}

// DEFINE PUBLIC RECEIVER FUNCTIONS.

//	DynamicActionType

// Name returns the full ROS name of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) Name() string { return t.name }

// MD5Sum returns the ROS compatible MD5 sum of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) MD5Sum() string { return t.md5sum }

// Text returns the full ROS text of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) Text() string { return t.text }

// GoalType returns the full ROS MessageType of the action goalType; required for actionlib.ActionType.
func (t *DynamicActionType) GoalType() ActionGoalType { return t.goalType }

// FeedbackType returns the full ROS MessageType of the action feedbackType; required for actionlib.ActionType.
func (t *DynamicActionType) FeedbackType() ActionFeedbackType { return t.feedbackType }

// ResultType returns the full ROS MessageType of the action resultType; required for actionlib.ActionType.
func (t *DynamicActionType) ResultType() ActionResultType { return t.resultType }

// NewAction creates a new DynamicAction instantiating the action type; required for actionlib.ActionType.
func (t *DynamicActionType) NewAction() Action {
	// Don't instantiate actions for incomplete types.
	if t == nil {
		return nil
	}
	// But otherwise, make a new one.
	a := new(DynamicAction)
	a.dynamicType = t
	a.Goal = t.GoalType().NewGoalMessage().(*DynamicActionGoal)
	a.Feedback = t.FeedbackType().NewFeedbackMessage().(*DynamicActionFeedback)
	a.Result = t.ResultType().NewResultMessage().(*DynamicActionResult)
	return a
}

//

// GetActionGoal returns the actionlib.ActionGoal of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionGoal() ActionGoal { return a.Goal }

// GetActionFeedback returns the actionlib.ActionFeedback of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionFeedback() ActionFeedback { return a.Feedback }

// GetActionResult returns the actionlib.ActionResult of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionResult() ActionResult { return a.Result }

// DynamicAction* Type instantiation Functions
// NewDynamicActionGoalType creates a new ActionGoalType interface
func NewDynamicActionGoalType(name string) (ActionGoalType, error) {
	goal := DynamicActionGoalType{}
	goalType, err := NewDynamicMessageTypeLiteral(name)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	// A goal type requires the fields GoalID and Header
	header, err := NewDynamicHeaderType()
	if err != nil {
		return nil, err
	}
	goalid, err := NewDynamicGoalIDType()
	if err != nil {
		return nil, err
	}
	goalType.Data()["goal_id"] = goalid.NewGoalIDMessage()
	goalType.Data()["header"] = header.NewHeaderMessage()
	goal.DynamicMessageType = goalType
	return &goal, nil
}

func NewDynamicActionResultType(name string) (ActionResultType, error) {
	result := DynamicActionResultType{}
	resultType, err := NewDynamicMessageTypeLiteral(name)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	// A goal type requires the fields GoalID and Header
	header, err := NewDynamicHeaderType()
	if err != nil {
		return nil, err
	}
	status, err := NewDynamicStatusType()
	if err != nil {
		return nil, err
	}
	resultType.Data()["status"] = status.NewStatusMessage()
	resultType.Data()["header"] = header.NewHeaderMessage()
	result.DynamicMessageType = resultType
	return &result, nil
}

// DynamicAction* Message instantiation Functions
// NewGoalMessage creates a new ActionGoal message
func (a *DynamicActionGoalType) NewGoalMessage() ActionGoal {
	m := DynamicActionGoal{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// NewFeedbackMessage creates a new ActionFeedback message
func (a *DynamicActionFeedbackType) NewFeedbackMessage() ActionFeedback {
	m := DynamicActionFeedback{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// NewResultMessage creates a new ActionResult message
func (a *DynamicActionResultType) NewResultMessage() ActionResult {
	m := DynamicActionResult{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// DynamicActionType* Type instantiation functions
// NewDynamicGoalIDType creates a new ActionGoalIDType interface
func NewDynamicGoalIDType() (ActionGoalIDType, error) {
	goalid := DynamicActionGoalIDType{}
	goalType, err := NewDynamicMessageTypeLiteral("actionlib_msgs/GoalID")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}

	goalid.DynamicMessageType = goalType
	return &goalid, nil
}

// NewDynamicHeaderType creates a new ActionHeaderType interface, which is essentially a std_msg header
func NewDynamicHeaderType() (ActionHeaderType, error) {
	header := DynamicActionHeaderType{}
	headerType, err := NewDynamicMessageTypeLiteral("std_msgs/Header")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	header.DynamicMessageType = headerType
	return &header, nil
}

// NewDynamicStatusType creates a new ActionStatusType interface
func NewDynamicStatusType() (ActionStatusType, error) {
	status := DynamicActionStatusType{}
	statusType, err := NewDynamicMessageTypeLiteral("actionlib_msgs/GoalStatus")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	// An action status type requires the field GoalID
	goalid, err := NewDynamicGoalIDType()
	if err != nil {
		return nil, err
	}
	statusType.Data()["goal_id"] = goalid.NewGoalIDMessage()
	status.DynamicMessageType = statusType
	return &status, nil
}

// NewDynamicStatusArrayType creates a new ActionStatusArrayType interface
func NewDynamicStatusArrayType() (ActionStatusArrayType, error) {
	statusArray := DynamicActionStatusArrayType{}
	statusArrayType, err := NewDynamicMessageTypeLiteral("actionlib_msgs/GoalStatusArray")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	// A status array type requires an action status array
	statusType, err := NewDynamicStatusType()
	if err != nil {
		return nil, err
	}
	status := statusType.NewStatusMessage()
	internalStatusArray := make([]ActionStatus, 0)
	internalStatusArray = append(internalStatusArray, status)
	statusArrayType.Data()["status_list"] = internalStatusArray
	// Also requires a header
	header, err := NewDynamicHeaderType()
	if err != nil {
		return nil, err
	}
	statusArrayType.Data()["header"] = header.NewHeaderMessage()

	statusArray.DynamicMessageType = statusArrayType
	return &statusArray, nil
}

// DynamicActionType* Message instantiation functions (parallels NewMessage() functionality)
// NewGoalIDMessage creates a new ActionGoalID message
func (a *DynamicActionGoalIDType) NewGoalIDMessage() ActionGoalID {
	m := DynamicActionGoalID{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// NewHeaderMessage creates a new ActionHeader message
func (a *DynamicActionHeaderType) NewHeaderMessage() ActionHeader {
	m := DynamicActionHeader{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// NewStatusMessage creates a new ActionStatus message
func (a *DynamicActionStatusType) NewStatusMessage() ActionStatus {
	m := DynamicActionStatus{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// NewStatusArrayMessage creates a new ActionStatusArray message
func (a *DynamicActionStatusArrayType) NewStatusArrayMessage() ActionStatusArray {
	m := DynamicActionStatusArray{}
	m.DynamicMessage = *a.DynamicMessageType.NewDynamicMessage()
	return &m
}

// Dynamic Action Goal Interface
// Get and set functions of DynamicActionGoal interface
func (m *DynamicActionGoal) SetGoal(goal Message) { m.Data()["goal"] = goal }
func (m *DynamicActionGoal) GetGoal() Message     { return m.Data()["goal"].(*DynamicMessage) }
func (m *DynamicActionGoal) GetGoalId() ActionGoalID {
	return m.Data()["goal_id"].(*DynamicActionGoalID)
}
func (m *DynamicActionGoal) SetGoalId(goalid ActionGoalID) { m.Data()["goal_id"] = goalid }
func (m *DynamicActionGoal) GetHeader() ActionHeader       { return m.Data()["header"].(*DynamicActionHeader) }
func (m *DynamicActionGoal) SetHeader(header ActionHeader) { m.Data()["header"] = header }

// Get and Set Functions of DynamicActionGoalID
func (m *DynamicActionGoalID) GetID() string       { return m.Data()["id"].(string) }
func (m *DynamicActionGoalID) SetID(id string)     { m.Data()["id"] = id }
func (m *DynamicActionGoalID) GetStamp() Time      { return m.Data()["stamp"].(Time) }
func (m *DynamicActionGoalID) SetStamp(stamp Time) { m.Data()["stamp"] = stamp }

//

// Dynamic Action Feedback Interface
// Get and set functions of DynamicActionFeedback interface
func (m *DynamicActionFeedback) SetFeedback(goal Message) { m.Data()["feedback"] = goal }
func (m *DynamicActionFeedback) GetFeedback() Message {
	return m.Data()["feedback"].(*DynamicMessage)
}
func (m *DynamicActionFeedback) GetStatus() ActionStatus {
	return m.Data()["status"].(*DynamicActionStatus)
}
func (m *DynamicActionFeedback) SetStatus(status ActionStatus) { m.Data()["status"] = status }
func (m *DynamicActionFeedback) GetHeader() ActionHeader {
	return m.Data()["header"].(*DynamicActionHeader)
}
func (m *DynamicActionFeedback) SetHeader(header ActionHeader) { m.Data()["header"] = header }

//

// Dynamic Action Feedback Interface
// Get and set functions of DynamicActionFeedback interface
func (m *DynamicActionResult) SetResult(result Message) { m.Data()["result"] = result }
func (m *DynamicActionResult) GetResult() Message       { return m.Data()["result"].(*DynamicMessage) }
func (m *DynamicActionResult) GetStatus() ActionStatus {
	// goalStat := m.Data()["status"].(*DynamicMessage)
	// goalStatusType := NewDynamicStatusType()
	// goalStatus := goalStatusType.NewStatusMessage()
	// goalStatus.Set
	return m.Data()["status"].(*DynamicActionStatus)
}
func (m *DynamicActionResult) SetStatus(status ActionStatus) { m.Data()["status"] = status }
func (m *DynamicActionResult) GetHeader() ActionHeader {
	return m.Data()["header"].(*DynamicActionHeader)
}
func (m *DynamicActionResult) SetHeader(header ActionHeader) { m.Data()["header"] = header }

// Get and Set Functions of shared type DynamicActionStatus
func (m *DynamicActionStatus) GetGoalID() ActionGoalID {
	return m.Data()["goal_id"].(*DynamicActionGoalID)
}
func (m *DynamicActionStatus) SetGoalID(id ActionGoalID) { m.Data()["goal_id"] = id }
func (m *DynamicActionStatus) GetStatus() uint8          { return m.Data()["status"].(uint8) }
func (m *DynamicActionStatus) SetStatus(status uint8)    { m.Data()["status"] = status }
func (m *DynamicActionStatus) GetStatusText() string     { return m.Data()["text"].(string) }
func (m *DynamicActionStatus) SetStatusText(text string) { m.Data()["text"] = text }

// Get and set functions of shared type DynamicActionHeader
func (m *DynamicActionHeader) GetStamp() Time      { return m.Data()["stamp"].(Time) }
func (m *DynamicActionHeader) SetStamp(stamp Time) { m.Data()["stamp"] = stamp }

// Get and set functions of the shared type DynamicActionStatusArray
func (m *DynamicActionStatusArray) GetStatusArray() []ActionStatus {
	return m.Data()["status_list"].([]ActionStatus)
}
func (m *DynamicActionStatusArray) SetStatusArray(statusArray []ActionStatus) {
	m.Data()["status_list"] = statusArray
}
func (m *DynamicActionStatusArray) GetHeader() ActionHeader {
	return m.Data()["header"].(*DynamicActionHeader)
}
func (m *DynamicActionStatusArray) SetHeader(header ActionHeader) { m.Data()["header"] = header }
