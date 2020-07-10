package actionlib

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"

	"github.com/edwinhayes/rosgo/ros"
)

type CommState uint8

const (
	WaitingForGoalAck CommState = iota
	Pending
	Active
	WaitingForResult
	WaitingForCancelAck
	Recalling
	Preempting
	Done
	Lost
)

func (cs CommState) String() string {
	switch cs {
	case WaitingForGoalAck:
		return "WAITING_FOR_GOAL_ACK"
	case Pending:
		return "PENDING"
	case Active:
		return "ACTIVE"
	case WaitingForResult:
		return "WAITING_FOR_RESULT"
	case WaitingForCancelAck:
		return "WAITING_FOR_CANCEL_ACK"
	case Recalling:
		return "RECALLING"
	case Preempting:
		return "PREEMPTING"
	case Done:
		return "DONE"
	case Lost:
		return "LOST"
	default:
		return "UNKNOWN"
	}
}

type clientStateMachine struct {
	state      CommState
	goalStatus *ros.DynamicMessage
	goalResult ActionResult
	mutex      sync.RWMutex
}

func newClientStateMachine() *clientStateMachine {
	// Create a goal status message for the state machine
	msgtype, _ := ros.NewDynamicMessageType("actionlib_msgs/GoalStatus")
	msg := msgtype.NewMessage().(*ros.DynamicMessage)
	// Set the status to pending
	msg.Data()["status"] = uint8(0)
	// Return new state machine with message and state
	return &clientStateMachine{
		state:      WaitingForGoalAck,
		goalStatus: msg,
	}
}

func (sm *clientStateMachine) getState() CommState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.state
}

func (sm *clientStateMachine) getGoalStatus() *ros.DynamicMessage {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.goalStatus
}

func (sm *clientStateMachine) getGoalResult() ActionResult {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.goalResult
}

func (sm *clientStateMachine) setState(state CommState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.state = state
}

func (sm *clientStateMachine) setGoalStatus(id *ros.DynamicMessage, status uint8, text string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalStatus.Data()["goalid"] = id
	sm.goalStatus.Data()["status"] = status
	sm.goalStatus.Data()["text"] = text
}

func (sm *clientStateMachine) setGoalResult(result ActionResult) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalResult = result
}

func (sm *clientStateMachine) setAsLost() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalStatus.Data()["status"] = uint8(Lost)
}

func (sm *clientStateMachine) transitionTo(state CommState, gh ClientGoalHandler, callback interface{}) {
	sm.setState(state)
	if callback != nil {
		fun := reflect.ValueOf(callback)
		args := []reflect.Value{reflect.ValueOf(gh)}
		numArgsNeeded := fun.Type().NumIn()

		if numArgsNeeded <= 1 {
			fun.Call(args[:numArgsNeeded])
		}
	}
}

func (sm *clientStateMachine) getTransitions(goalStatus *ros.DynamicMessage) (stateList list.List, err error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	status := goalStatus.Data()["status"].(uint8)

	switch sm.state {
	case WaitingForGoalAck:
		switch status {
		case 0:
			stateList.PushBack(Pending)
			break
		case 1:
			stateList.PushBack(Active)
			break
		case 5:
			stateList.PushBack(Pending)
			stateList.PushBack(WaitingForCancelAck)
			break
		case 7:
			stateList.PushBack(Pending)
			stateList.PushBack(Recalling)
			break
		case 8:
			stateList.PushBack(Pending)
			stateList.PushBack(WaitingForResult)
			break
		case 2:
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			break
		}
		break

	case Pending:
		switch status {
		case 0:
			break
		case 1:
			stateList.PushBack(Active)
			break
		case 5:
			stateList.PushBack(WaitingForResult)
			break
		case 7:
			stateList.PushBack(Recalling)
			break
		case 8:
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case 2:
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			break
		}
		break
	case Active:
		switch status {
		case 0:
			err = fmt.Errorf("invalid transition from Active to Pending")
			break
		case 1:
			break
		case 5:
			err = fmt.Errorf("invalid transition from Active to Rejected")
			break
		case 7:
			err = fmt.Errorf("invalid transition from Active to Recalling")
			break
		case 8:
			err = fmt.Errorf("invalid transition from Active to Recalled")
			break
		case 2:
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			stateList.PushBack(Preempting)
			break
		}
		break
	case WaitingForResult:
		switch status {
		case 0:
			err = fmt.Errorf("invalid transition from WaitingForResult to Pending")
			break
		case 1:
			break
		case 5:
			break
		case 7:
			err = fmt.Errorf("invalid transition from WaitingForResult to Recalling")
			break
		case 8:
			break
		case 2:
			break
		case 3:
			break
		case 4:
			break
		case 6:
			err = fmt.Errorf("invalid transition from WaitingForResult to Preempting")
			break
		}
		break
	case WaitingForCancelAck:
		switch status {
		case 0:
			break
		case 1:
			break
		case 5:
			stateList.PushBack(WaitingForResult)
			break
		case 7:
			stateList.PushBack(Recalling)
			break
		case 8:
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case 2:
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			stateList.PushBack(Preempting)
			break
		}
		break
	case Recalling:
		switch status {
		case 0:
			err = fmt.Errorf("invalid transition from Recalling to Pending")
			break
		case 1:
			err = fmt.Errorf("invalid transition from Recalling to Active")
			break
		case 5:
			stateList.PushBack(WaitingForResult)
			break
		case 7:
			break
		case 8:
			stateList.PushBack(WaitingForResult)
			break
		case 2:
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			stateList.PushBack(Preempting)
			break
		}
		break
	case Preempting:
		switch status {
		case 0:
			err = fmt.Errorf("invalid transition from Preempting to Pending")
			break
		case 1:
			err = fmt.Errorf("invalid transition from Preempting to Active")
			break
		case 5:
			err = fmt.Errorf("invalid transition from Preempting to Rejected")
			break
		case 7:
			err = fmt.Errorf("invalid transition from Preempting to Recalling")
			break
		case 8:
			err = fmt.Errorf("invalid transition from Preempting to Recalled")
			break
		case 2:
			stateList.PushBack(WaitingForResult)
			break
		case 3:
			stateList.PushBack(WaitingForResult)
			break
		case 4:
			stateList.PushBack(WaitingForResult)
			break
		case 6:
			break
		}
		break
	case Done:
		switch status {
		case 0:
			err = fmt.Errorf("invalid transition from Done to Pending")
			break
		case 1:
			err = fmt.Errorf("invalid transition from Done to Active")
			break
		case 5:
			break
		case 7:
			err = fmt.Errorf("invalid transition from Done to Recalling")
			break
		case 8:
			break
		case 2:
			break
		case 3:
			break
		case 4:
			break
		case 6:
			err = fmt.Errorf("invalid transition from Done to Preempting")
			break
		}
		break
	}

	return
}
