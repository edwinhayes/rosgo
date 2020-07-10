package ros

// IMPORT REQUIRED PACKAGES.

import (
	"strings"

	"github.com/edwinhayes/rosgo/actionlib"
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
	goalType     MessageType
	feedbackType MessageType
	resultType   MessageType
}

// DynamicAction abstracts an instance of a ROS Action whose type is only known at runtime.  The schema of the action is denoted by the referenced DynamicActionType, while the
// Goal, Feedback and Result references the rosgo Messages it implements.  DynamicAction implements the rosgo actionlib.Action interface, allowing
// it to be used throughout rosgo in the same manner as action types generated at compiletime by gengo.
type DynamicAction struct {
	dynamicType *DynamicActionType
	Goal        Message
	Feedback    Message
	Result      Message
}

// DEFINE PRIVATE STRUCTURES.

// DEFINE PUBLIC GLOBALS.

// DEFINE PRIVATE GLOBALS.

// DEFINE PUBLIC STATIC FUNCTIONS.

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

	// Create a message context if for some reason it does not exist yet, as it also contains service definitions
	if context == nil {
		// Create context for our ROS install.
		c, err := libgengo.NewMsgContext(strings.Split(GetRuntimePackagePath(), ":"))
		if err != nil {
			return nil, err
		}
		context = c
	}
	// We need to try to look up the full name, in case we've just been given a short name.
	fullname := typeName

	// Load context for the target message.
	spec, err := context.LoadAction(fullname)
	if err != nil {
		return nil, err
	}

	// Now we know all about the service!
	m.name = spec.ShortName
	m.md5sum = spec.MD5Sum
	m.text = spec.Text
	m.goalType, err = NewDynamicMessageType(spec.Goal.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating request type")
	}
	m.feedbackType, err = NewDynamicMessageType(spec.Feedback.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating request type")
	}
	m.resultType, err = NewDynamicMessageType(spec.Result.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating request type")
	}

	// We've successfully made a new service type matching the requested ROS type.
	return m, nil

}

// DEFINE PUBLIC RECEIVER FUNCTIONS.

//	DynamicActionType

// Name returns the full ROS name of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) Name() string {
	return t.name
}

// MD5Sum returns the ROS compatible MD5 sum of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) MD5Sum() string {
	return t.md5sum
}

// Text returns the full ROS text of the action type; required for actionlib.ActionType.
func (t *DynamicActionType) Text() string {
	return t.text
}

// GoalType returns the full ROS MessageType of the action goalType; required for actionlib.ActionType.
func (t *DynamicActionType) GoalType() MessageType {
	return t.goalType
}

// FeedbackType returns the full ROS MessageType of the action feedbackType; required for actionlib.ActionType.
func (t *DynamicActionType) FeedbackType() MessageType {
	return t.feedbackType
}

// ResultType returns the full ROS MessageType of the action resultType; required for actionlib.ActionType.
func (t *DynamicActionType) ResultType() MessageType {
	return t.resultType
}

// NewAction creates a new DynamicAction instantiating the action type; required for actionlib.ActionType.
func (t *DynamicActionType) NewAction() actionlib.Action {
	// Don't instantiate actions for incomplete types.
	if t == nil {
		return nil
	}
	// But otherwise, make a new one.
	a := new(DynamicAction)
	a.dynamicType = t
	a.Goal = t.goalType().NewMessage()
	a.Feedback = t.feedbackType().NewMessage()
	a.Result = t.resultType().NewMessage()
	return d
}

//	DynamicAction

// GetActionGoal returns the actionlib.ActionGoal of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionGoal() actionlib.ActionGoal {
	return a.Goal
}

// GetActionFeedback returns the actionlib.ActionFeedback of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionFeedback() actionlib.ActionFeedback {
	return a.Feedback
}

// GetActionResult returns the actionlib.ActionResult of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetResult() actionlib.ActionResult {
	return a.Result
}
