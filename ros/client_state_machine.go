package ros

import (
	"container/list"
	"fmt"
	"reflect"
	"sync"
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
	goalStatus ActionStatus
	goalResult ActionResult
	mutex      sync.RWMutex
}

func newClientStateMachine() *clientStateMachine {
	// Create a goal status message for the state machine
	statusType, _ := NewDynamicStatusType()
	status := statusType.NewStatusMessage().(*DynamicActionStatus)
	// Set the status to pending
	status.SetStatus(uint8(0))
	// Return new state machine with message and state
	return &clientStateMachine{
		state:      WaitingForGoalAck,
		goalStatus: status,
	}
}

func (sm *clientStateMachine) getState() CommState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.state
}

func (sm *clientStateMachine) getGoalStatus() ActionStatus {
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

func (sm *clientStateMachine) setGoalStatus(id ActionGoalID, status uint8, text string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalStatus.SetGoalID(id)
	sm.goalStatus.SetStatus(status)
	sm.goalStatus.SetStatusText(text)
}

func (sm *clientStateMachine) setGoalResult(result ActionResult) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalResult = result
}

func (sm *clientStateMachine) setAsLost() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.goalStatus.SetStatus(uint8(Lost))
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

func (sm *clientStateMachine) getTransitions(goalStatus ActionStatus) (stateList list.List, err error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	status := goalStatus.GetStatus()

	switch sm.state {
	case WaitingForGoalAck:
		switch status {
		case uint8(0):
			stateList.PushBack(Pending)
			break
		case uint8(1):
			stateList.PushBack(Active)
			break
		case uint8(5):
			stateList.PushBack(Pending)
			stateList.PushBack(WaitingForCancelAck)
			break
		case uint8(7):
			stateList.PushBack(Pending)
			stateList.PushBack(Recalling)
			break
		case uint8(8):
			stateList.PushBack(Pending)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(2):
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			break
		}
		break

	case Pending:
		switch status {
		case uint8(0):
			break
		case uint8(1):
			stateList.PushBack(Active)
			break
		case uint8(5):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(7):
			stateList.PushBack(Recalling)
			break
		case uint8(8):
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(2):
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(Active)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			stateList.PushBack(Active)
			stateList.PushBack(Preempting)
			break
		}
		break
	case Active:
		switch status {
		case uint8(0):
			err = fmt.Errorf("invalid transition from Active to Pending")
			break
		case uint8(1):
			break
		case uint8(5):
			err = fmt.Errorf("invalid transition from Active to Rejected")
			break
		case uint8(7):
			err = fmt.Errorf("invalid transition from Active to Recalling")
			break
		case uint8(8):
			err = fmt.Errorf("invalid transition from Active to Recalled")
			break
		case uint8(2):
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			stateList.PushBack(Preempting)
			break
		}
		break
	case WaitingForResult:
		switch status {
		case uint8(0):
			err = fmt.Errorf("invalid transition from WaitingForResult to Pending")
			break
		case uint8(1):
			break
		case uint8(5):
			break
		case uint8(7):
			err = fmt.Errorf("invalid transition from WaitingForResult to Recalling")
			break
		case uint8(8):
			break
		case uint8(2):
			break
		case uint8(3):
			break
		case uint8(4):
			break
		case uint8(6):
			err = fmt.Errorf("invalid transition from WaitingForResult to Preempting")
			break
		}
		break
	case WaitingForCancelAck:
		switch status {
		case uint8(0):
			break
		case uint8(1):
			break
		case uint8(5):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(7):
			stateList.PushBack(Recalling)
			break
		case uint8(8):
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(2):
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(Recalling)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			stateList.PushBack(Preempting)
			break
		}
		break
	case Recalling:
		switch status {
		case uint8(0):
			err = fmt.Errorf("invalid transition from Recalling to Pending")
			break
		case uint8(1):
			err = fmt.Errorf("invalid transition from Recalling to Active")
			break
		case uint8(5):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(7):
			break
		case uint8(8):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(2):
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(Preempting)
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			stateList.PushBack(Preempting)
			break
		}
		break
	case Preempting:
		switch status {
		case uint8(0):
			err = fmt.Errorf("invalid transition from Preempting to Pending")
			break
		case uint8(1):
			err = fmt.Errorf("invalid transition from Preempting to Active")
			break
		case uint8(5):
			err = fmt.Errorf("invalid transition from Preempting to Rejected")
			break
		case uint8(7):
			err = fmt.Errorf("invalid transition from Preempting to Recalling")
			break
		case uint8(8):
			err = fmt.Errorf("invalid transition from Preempting to Recalled")
			break
		case uint8(2):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(3):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(4):
			stateList.PushBack(WaitingForResult)
			break
		case uint8(6):
			break
		}
		break
	case Done:
		switch status {
		case uint8(0):
			err = fmt.Errorf("invalid transition from Done to Pending")
			break
		case uint8(1):
			err = fmt.Errorf("invalid transition from Done to Active")
			break
		case uint8(5):
			break
		case uint8(7):
			err = fmt.Errorf("invalid transition from Done to Recalling")
			break
		case uint8(8):
			break
		case uint8(2):
			break
		case uint8(3):
			break
		case uint8(4):
			break
		case uint8(6):
			err = fmt.Errorf("invalid transition from Done to Preempting")
			break
		}
		break
	}

	return
}
