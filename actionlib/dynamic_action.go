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
	goalType     ros.MessageType
	feedbackType ros.MessageType
	resultType   ros.MessageType
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
	m.goalType, err = ros.NewDynamicMessageType(spec.Goal.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating goal type")
	}
	m.feedbackType, err = ros.NewDynamicMessageType(spec.Feedback.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating feedback type")
	}
	m.resultType, err = ros.NewDynamicMessageType(spec.Result.FullName)
	if err != nil {
		return nil, errors.Wrap(err, "error generating result type")
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
func (t *DynamicActionType) GoalType() ros.MessageType {
	return t.goalType
}

// FeedbackType returns the full ROS MessageType of the action feedbackType; required for actionlib.ActionType.
func (t *DynamicActionType) FeedbackType() ros.MessageType {
	return t.feedbackType
}

// ResultType returns the full ROS MessageType of the action resultType; required for actionlib.ActionType.
func (t *DynamicActionType) ResultType() ros.MessageType {
	return t.resultType
}

// NewAction creates a new DynamicAction instantiating the action type; required for actionlib.ActionType.
func (t *DynamicActionType) NewAction() Action {
	// Don't instantiate actions for incomplete types.
	if t == nil {
		return nil
	}
	// But otherwise, make a new one.
	a := new(DynamicAction)
	a.dynamicType = t
	a.Goal = t.GoalType().NewMessage().(ActionGoal)
	a.Feedback = t.FeedbackType().NewMessage().(ActionFeedback)
	a.Result = t.ResultType().NewMessage().(ActionResult)
	return a
}

//	DynamicAction

// GetActionGoal returns the actionlib.ActionGoal of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionGoal() ActionGoal {
	return a.Goal
}

// GetActionFeedback returns the actionlib.ActionFeedback of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionFeedback() ActionFeedback {
	return a.Feedback
}

// GetActionResult returns the actionlib.ActionResult of the DynamicAction; required for actionlib.Action.
func (a *DynamicAction) GetActionResult() ActionResult {
	return a.Result
}
