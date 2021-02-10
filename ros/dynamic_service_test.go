package ros

import (
	"testing"
)

func TestDynamicService_ServiceType_Load(t *testing.T) {
	serviceType, err := NewDynamicServiceType("turtlesim/TeleportRelative")

	if err != nil {
		t.Skipf("test skipped because ROS environment not set up, err: %s", err)
		return
	}

	requestType, ok := serviceType.reqType.(*DynamicMessageType)

	if !ok {
		t.Fatalf("expected request type to be dynamic message type")
	}

	if requestType.nested == nil {
		t.Fatalf("request type nested is nil!")
	}

	if requestType.spec == nil {
		t.Fatalf("request type spec is nil!")
	}

	if requestType.spec.Fields == nil {
		t.Fatalf("request type Fields is nil!")
	}

	responseType, ok := serviceType.resType.(*DynamicMessageType)

	if !ok {
		t.Fatalf("expected response type to be dynamic message type")
	}

	if responseType.nested == nil {
		t.Fatalf("response type nested is nil!")
	}

	if responseType.spec == nil {
		t.Fatalf("response type spec is nil!")
	}

	if responseType.spec.Fields == nil {
		t.Fatalf("response type Fields is nil!")
	}

}
