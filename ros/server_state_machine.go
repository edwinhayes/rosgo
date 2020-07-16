package ros

import (
	"fmt"
	"sync"
)

type Event uint8

const (
	CancelRequest Event = iota + 1
	Cancel
	Reject
	Accept
	Succeed
	Abort
)

func (e Event) String() string {
	switch e {
	case CancelRequest:
		return "CANCEL_REQUEST"
	case Cancel:
		return "CANCEL"
	case Reject:
		return "REJECT"
	case Accept:
		return "ACCEPT"
	case Succeed:
		return "SUCCEED"
	case Abort:
		return "ABORT"
	default:
		return "UNKNOWN"
	}
}

type serverStateMachine struct {
	goalStatus ActionStatus
	mutex      sync.RWMutex
}

func newServerStateMachine(goalID ActionGoalID) *serverStateMachine {
	// Create a goal status message with pending status
	statusType, _ := NewDynamicStatusType()
	status := statusType.NewStatusMessage()
	status.SetStatus(0)
	status.SetGoalID(goalID)
	return &serverStateMachine{
		goalStatus: status,
	}
}

func (sm *serverStateMachine) transition(event Event, text string) (ActionStatus, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	nextState := sm.goalStatus.GetStatus()

	switch sm.goalStatus.GetStatus() {
	case 0:
		switch event {
		case Reject:
			nextState = uint8(5)
			break
		case CancelRequest:
			nextState = uint8(7)
			break
		case Cancel:
			nextState = uint8(8)
			break
		case Accept:
			nextState = uint8(1)
			break
		default:
			return sm.goalStatus, fmt.Errorf("invalid transition Event")
		}

	case uint8(7):
		switch event {
		case Reject:
			nextState = uint8(5)
			break
		case Cancel:
			nextState = uint8(8)
			break
		case Accept:
			nextState = uint8(6)
			break
		default:
			return sm.goalStatus, fmt.Errorf("invalid transition Event")
		}

	case uint8(1):
		switch event {
		case Succeed:
			nextState = uint8(3)
			break
		case CancelRequest:
			nextState = uint8(6)
			break
		case Cancel:
			nextState = uint8(2)
			break
		case Abort:
			nextState = uint8(4)
			break
		default:
			return sm.goalStatus, fmt.Errorf("invalid transition Event")
		}

	case uint8(6):
		switch event {
		case Succeed:
			nextState = uint8(3)
			break
		case Cancel:
			nextState = uint8(2)
			break
		case Abort:
			nextState = uint8(4)
			break
		default:
			return sm.goalStatus, fmt.Errorf("invalid transition Event")
		}
	case uint8(5):
		break
	case uint8(8):
		break
	case uint8(3):
		break
	case uint8(2):
		break
	case uint8(4):
		break
	default:
		return sm.goalStatus, fmt.Errorf("invalid state")
	}

	sm.goalStatus.SetStatus(nextState)
	sm.goalStatus.SetStatusText(text)

	return sm.goalStatus, nil
}

func (sm *serverStateMachine) getStatus() ActionStatus {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.goalStatus
}
