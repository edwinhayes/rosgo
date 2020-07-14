package actionlib

// IMPORT REQUIRED PACKAGES.

import (
	"os"
	"strings"

	"github.com/edwinhayes/rosgo/libgengo"
	"github.com/edwinhayes/rosgo/ros"
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
type DynamicActionGoal struct{ ros.DynamicMessage }
type DynamicActionFeedback struct{ ros.DynamicMessage }
type DynamicActionResult struct{ ros.DynamicMessage }

// DynamicActionType* type interfaces
type DynamicActionGoalType struct{ ros.DynamicMessageType }
type DynamicActionFeedbackType struct{ ros.DynamicMessageType }
type DynamicActionResultType struct{ ros.DynamicMessageType }

// DynamicAction** shared interfaces.
type DynamicActionGoalID struct{ ros.DynamicMessage }
type DynamicActionHeader struct{ ros.DynamicMessage }
type DynamicActionStatus struct{ ros.DynamicMessage }
type DynamicActionStatusArray struct{ ros.DynamicMessage }

// DynamicActionType** shared type interfaces
type DynamicActionGoalIDType struct{ ros.DynamicMessageType }
type DynamicActionHeaderType struct{ ros.DynamicMessageType }
type DynamicActionStatusType struct{ ros.DynamicMessageType }
type DynamicActionStatusArrayType struct{ ros.DynamicMessageType }

// DEFINE PRIVATE STRUCTURES.

// DEFINE PUBLIC GLOBALS.

// DEFINE PRIVATE GLOBALS.

var rosPkgPath string // Colon separated list of paths to search for message definitions on.

var context *libgengo.PkgContext // We'll try to preserve a single message context to avoid reloading each time.

// DEFINE PUBLIC STATIC FUNCTIONS.

func GetRuntimePackagePath() string {
	// If a package path hasn't been set at the time of first use, by default we'll just use the ROS environment default.
	if rosPkgPath == "" {
		rosPkgPath = os.Getenv("ROS_PACKAGE_PATH")
	}
	// All done.
	return rosPkgPath
}

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
	goal := DynamicActionGoalType{}
	goalType, err := ros.NewDynamicMessageTypeLiteral(spec.Goal.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	goal.DynamicMessageType = goalType
	m.goalType = &goal

	// Create Dynamic Feedback Type
	feedback := DynamicActionFeedbackType{}
	feedbackType, err := ros.NewDynamicMessageTypeLiteral(spec.Feedback.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	feedback.DynamicMessageType = feedbackType
	m.feedbackType = &feedback

	// Create Dynamic Result Type
	result := DynamicActionResultType{}
	resultType, err := ros.NewDynamicMessageTypeLiteral(spec.Result.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating result type")
	}
	result.DynamicMessageType = resultType
	m.resultType = &result
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
	goalType, err := ros.NewDynamicMessageTypeLiteral("actionlib_msgs/GoalID")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	goalid.DynamicMessageType = goalType
	return &goalid, nil
}

// NewDynamicHeaderType creates a new ActionHeaderType interface, which is essentially a std_msg header
func NewDynamicHeaderType() (ActionHeaderType, error) {
	header := DynamicActionHeaderType{}
	headerType, err := ros.NewDynamicMessageTypeLiteral("std_msgs/Header")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	header.DynamicMessageType = headerType
	return &header, nil
}

// NewDynamicStatusType creates a new ActionStatusType interface
func NewDynamicStatusType() (ActionStatusType, error) {
	status := DynamicActionStatusType{}
	statusType, err := ros.NewDynamicMessageTypeLiteral("actionlib_msgs/GoalStatus")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	status.DynamicMessageType = statusType
	return &status, nil
}

// NewDynamicStatusArrayType creates a new ActionStatusArrayType interface
func NewDynamicStatusArrayType() (ActionStatusArrayType, error) {
	statusArray := DynamicActionStatusArrayType{}
	statusArrayType, err := ros.NewDynamicMessageTypeLiteral("actionlib_msgs/GoalStatusArray")
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
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
func (m *DynamicActionGoal) SetGoal(goal ros.Message)      { m.Data()["goal"] = goal }
func (m *DynamicActionGoal) GetGoal() ros.Message          { return m.Data()["goal"].(*ros.DynamicMessage) }
func (m *DynamicActionGoal) GetGoalId() ActionGoalID       { return m.Data()["goalid"].(*DynamicActionGoalID) }
func (m *DynamicActionGoal) SetGoalId(goalid ActionGoalID) { m.Data()["goalid"] = goalid }
func (m *DynamicActionGoal) GetHeader() ActionHeader       { return m.Data()["header"].(*DynamicActionHeader) }
func (m *DynamicActionGoal) SetHeader(header ActionHeader) { m.Data()["header"] = header }

// Get and Set Functions of DynamicActionGoalID
func (m *DynamicActionGoalID) GetID() string           { return m.Data()["id"].(string) }
func (m *DynamicActionGoalID) SetID(id string)         { m.Data()["id"] = id }
func (m *DynamicActionGoalID) GetStamp() ros.Time      { return m.Data()["stamp"].(ros.Time) }
func (m *DynamicActionGoalID) SetStamp(stamp ros.Time) { m.Data()["stamp"] = stamp }

//

// Dynamic Action Feedback Interface
// Get and set functions of DynamicActionFeedback interface
func (m *DynamicActionFeedback) SetFeedback(goal ros.Message) { m.Data()["feedback"] = goal }
func (m *DynamicActionFeedback) GetFeedback() ros.Message {
	return m.Data()["goal"].(*ros.DynamicMessage)
}
func (m *DynamicActionFeedback) GetStatus() ActionStatus {
	return m.Data()["goalid"].(*DynamicActionStatus)
}
func (m *DynamicActionFeedback) SetStatus(status ActionStatus) { m.Data()["status"] = status }
func (m *DynamicActionFeedback) GetHeader() ActionHeader {
	return m.Data()["header"].(*DynamicActionHeader)
}
func (m *DynamicActionFeedback) SetHeader(header ActionHeader) { m.Data()["header"] = header }

//

// Dynamic Action Feedback Interface
// Get and set functions of DynamicActionFeedback interface
func (m *DynamicActionResult) SetResult(result ros.Message) { m.Data()["result"] = result }
func (m *DynamicActionResult) GetResult() ros.Message       { return m.Data()["goal"].(*ros.DynamicMessage) }
func (m *DynamicActionResult) GetStatus() ActionStatus {
	return m.Data()["goalid"].(*DynamicActionStatus)
}
func (m *DynamicActionResult) SetStatus(status ActionStatus) { m.Data()["status"] = status }
func (m *DynamicActionResult) GetHeader() ActionHeader {
	return m.Data()["header"].(*DynamicActionHeader)
}
func (m *DynamicActionResult) SetHeader(header ActionHeader) { m.Data()["header"] = header }

// Get and Set Functions of shared type DynamicActionStatus
func (m *DynamicActionStatus) GetGoalID() ActionGoalID   { return m.Data()["id"].(*DynamicActionGoalID) }
func (m *DynamicActionStatus) SetGoalID(id ActionGoalID) { m.Data()["id"] = id }
func (m *DynamicActionStatus) GetStatus() uint8          { return m.Data()["status"].(uint8) }
func (m *DynamicActionStatus) SetStatus(status uint8)    { m.Data()["status"] = status }
func (m *DynamicActionStatus) GetStatusText() string     { return m.Data()["text"].(string) }
func (m *DynamicActionStatus) SetStatusText(text string) { m.Data()["text"] = text }

// Get and set functions of shared type DynamicActionHeader
func (m *DynamicActionHeader) GetStamp() ros.Time      { return m.Data()["stamp"].(ros.Time) }
func (m *DynamicActionHeader) SetStamp(stamp ros.Time) { m.Data()["stamp"] = stamp }

// Get and set functions of the shared type DynamicActionStatusArray
func (m *DynamicActionStatusArray) GetStatusArray() []ActionStatus {
	return m.Data()["status_list"].([]ActionStatus)
}
func (m *DynamicActionStatusArray) SetStatusArray(statusArray []ActionStatus) {
	m.Data()["status_list"] = statusArray
}
