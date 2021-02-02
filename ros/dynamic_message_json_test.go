package ros

import (
	"encoding/json"
	"reflect"
	"testing"

	gengo "github.com/team-rocos/rosgo/libgengo"
)

// Marshalling dynamic message is equivalent to marshalling the raw data in dynamic message.
func TestDynamicMessage_marshalJSON_primitives(t *testing.T) {

	fields := []gengo.Field{
		// Singular primitives.
		*gengo.NewField("Testing", "uint8", "singular_u8", false, 0),
		*gengo.NewField("Testing", "uint16", "singular_u16", false, 0),
		*gengo.NewField("Testing", "uint32", "singular_u32", false, 0),
		*gengo.NewField("Testing", "uint64", "singular_u64", false, 0),
		*gengo.NewField("Testing", "int8", "singular_i8", false, 0),
		*gengo.NewField("Testing", "int16", "singular_i16", false, 0),
		*gengo.NewField("Testing", "int32", "singular_i32", false, 0),
		*gengo.NewField("Testing", "int64", "singular_i64", false, 0),
		*gengo.NewField("Testing", "bool", "singular_b", false, 0),
		*gengo.NewField("Testing", "float32", "singular_f32", false, 0),
		*gengo.NewField("Testing", "float64", "singular_f64", false, 0),
		*gengo.NewField("Testing", "string", "singular_s", false, 0),
		*gengo.NewField("Testing", "time", "singular_t", false, 0),
		*gengo.NewField("Testing", "duration", "singular_d", false, 0),
		// Fixed arrays.
		*gengo.NewField("Testing", "uint8", "fixed_u8", true, 8),
		*gengo.NewField("Testing", "uint16", "fixed_u16", true, 4),
		*gengo.NewField("Testing", "uint32", "fixed_u32", true, 2),
		*gengo.NewField("Testing", "uint64", "fixed_u64", true, 1),
		*gengo.NewField("Testing", "int8", "fixed_i8", true, 8),
		*gengo.NewField("Testing", "int16", "fixed_i16", true, 4),
		*gengo.NewField("Testing", "int32", "fixed_i32", true, 2),
		*gengo.NewField("Testing", "int64", "fixed_i64", true, 1),
		*gengo.NewField("Testing", "bool", "fixed_b", true, 8),
		*gengo.NewField("Testing", "float32", "fixed_f32", true, 2),
		*gengo.NewField("Testing", "float64", "fixed_f64", true, 1),
		*gengo.NewField("Testing", "string", "fixed_s", true, 3),
		*gengo.NewField("Testing", "time", "fixed_t", true, 2),
		*gengo.NewField("Testing", "duration", "fixed_d", true, 2),
		// Dynamic arrays.
		*gengo.NewField("Testing", "uint8", "dyn_u8", true, -1),
		*gengo.NewField("Testing", "uint16", "dyn_u16", true, -1),
		*gengo.NewField("Testing", "uint32", "dyn_u32", true, -1),
		*gengo.NewField("Testing", "uint64", "dyn_u64", true, -1),
		*gengo.NewField("Testing", "int8", "dyn_i8", true, -1),
		*gengo.NewField("Testing", "int16", "dyn_i16", true, -1),
		*gengo.NewField("Testing", "int32", "dyn_i32", true, -1),
		*gengo.NewField("Testing", "int64", "dyn_i64", true, -1),
		*gengo.NewField("Testing", "bool", "dyn_b", true, -1),
		*gengo.NewField("Testing", "float32", "dyn_f32", true, -1),
		*gengo.NewField("Testing", "float64", "dyn_f64", true, -1),
		*gengo.NewField("Testing", "string", "dyn_s", true, -1),
		*gengo.NewField("Testing", "time", "dyn_t", true, -1),
		*gengo.NewField("Testing", "duration", "dyn_d", true, -1),
	}

	data := map[string]interface{}{
		"singular_u8":  uint8(0x12),
		"singular_u16": uint16(0x3456),
		"singular_u32": uint32(0x789abcde),
		"singular_u64": uint64(0x123456789abcdef0),
		"singular_i8":  int8(-2),
		"singular_i16": int16(-2),
		"singular_i32": int32(-2),
		"singular_i64": int64(-2),
		"singular_b":   true,
		"singular_f32": JsonFloat32{1234.5678},
		"singular_f64": JsonFloat64{-9876.5432},
		"singular_s":   "Rocos",
		"singular_t":   NewTime(0xfeedf00d, 0x1337beef),
		"singular_d":   NewDuration(0x50607080, 0x10203040),
		"fixed_u8":     []uint8{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12},
		"fixed_u16":    []uint16{0xdef0, 0x9abc, 0x5678, 0x1234},
		"fixed_u32":    []uint32{0x9abcdef0, 0x12345678},
		"fixed_u64":    []uint64{0x123456789abcdef0},
		"fixed_i8":     []int8{-2, -1, 0, 1, 2, 3, 4, 5},
		"fixed_i16":    []int16{-2, -1, 0, 1},
		"fixed_i32":    []int32{-2, 1},
		"fixed_i64":    []int64{-2},
		"fixed_b":      []bool{true, true, false, false, true, false, true, false},
		"fixed_f32":    []JsonFloat32{{1234.5678}, {1234.5678}},
		"fixed_f64":    []JsonFloat64{{-9876.5432}},
		"fixed_s":      []string{"Rocos", "soroc", "croos"},
		"fixed_t":      []Time{NewTime(0xfeedf00d, 0x1337beef), NewTime(0x1337beef, 0x1337f00d)},
		"fixed_d":      []Duration{NewDuration(0x40302010, 0x00706050), NewDuration(0x50607080, 0x10203040)},
		"dyn_u8":       []uint8{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12},
		"dyn_u16":      []uint16{0xdef0, 0x9abc, 0x5678, 0x1234},
		"dyn_u32":      []uint32{0x9abcdef0, 0x12345678},
		"dyn_u64":      []uint64{0x123456789abcdef0},
		"dyn_i8":       []int8{-2, -1, 0, 1, 2, 3, 4, 5},
		"dyn_i16":      []int16{-2, -1, 0, 1},
		"dyn_i32":      []int32{-2, 1},
		"dyn_i64":      []int64{-2},
		"dyn_b":        []bool{true, true, false, false, true, false, true, false},
		"dyn_f32":      []JsonFloat32{{1234.5678}, {1234.5678}},
		"dyn_f64":      []JsonFloat64{{-9876.5432}},
		"dyn_s":        []string{"Rocos", "soroc", "croos"},
		"dyn_t":        []Time{NewTime(0xfeedf00d, 0x1337beef), NewTime(0x1337beef, 0x1337f00d)},
		"dyn_d":        []Duration{NewDuration(0x40302010, 0x00706050), NewDuration(0x50607080, 0x10203040)},
	}

	testMessageType := &DynamicMessageType{
		spec:         generateTestSpec(fields),
		nested:       make(map[string]*DynamicMessageType),
		jsonPrealloc: 0,
	}

	testMessage := &DynamicMessage{
		dynamicType: testMessageType,
		data:        data,
	}

	verifyJSONMarshalling(t, testMessage)
}

// Marshalling dynamic message strings is equivalent to the default marshaller.
func TestDynamicMessage_marshalJSON_strings(t *testing.T) {

	fields := []gengo.Field{
		*gengo.NewField("T", "string", "newline", false, 0),
		*gengo.NewField("T", "string", "quotes", false, 0),
		*gengo.NewField("T", "string", "backslash", false, 0),
	}

	data := map[string]interface{}{
		"newline":   "custom string \n with newline",
		"quotes":    "custom string \"with quotes\"",
		"backslash": "custom string with backs\\ash",
	}

	testMessageType := &DynamicMessageType{
		spec:         generateTestSpec(fields),
		nested:       make(map[string]*DynamicMessageType),
		jsonPrealloc: 0,
	}

	testMessage := &DynamicMessage{
		dynamicType: testMessageType,
		data:        data,
	}

	verifyJSONMarshalling(t, testMessage)
}

// Marshalling dynamic message floats is equivalent to the default marshaller.
func TestDynamicMessage_marshalJSON_floats(t *testing.T) {

	fields := []gengo.Field{
		*gengo.NewField("T", "float32", "f32big", false, 0),
		*gengo.NewField("T", "float32", "f32small", false, 0),
		*gengo.NewField("T", "float32", "f32nbig", false, 0),
		*gengo.NewField("T", "float32", "f32nsmall", false, 0),
		*gengo.NewField("T", "float32", "f32zero", false, 0),
		*gengo.NewField("T", "float64", "f64big", false, 0),
		*gengo.NewField("T", "float64", "f64small", false, 0),
		*gengo.NewField("T", "float64", "f64nbig", false, 0),
		*gengo.NewField("T", "float64", "f64nsmall", false, 0),
		*gengo.NewField("T", "float64", "f64zero", false, 0),
	}

	data := map[string]interface{}{
		"f32big":    JsonFloat32{F: 1.13e22},
		"f32small":  JsonFloat32{F: 1.13e-7},
		"f32nbig":   JsonFloat32{F: -1.13e22},
		"f32nsmall": JsonFloat32{F: -1.13e-7},
		"f32zero":   JsonFloat32{F: 0.0},
		"f64big":    JsonFloat64{F: 1.13e22},
		"f64small":  JsonFloat64{F: 1.13e-7},
		"f64nbig":   JsonFloat64{F: -1.13e22},
		"f64nsmall": JsonFloat64{F: -1.13e-7},
		"f64zero":   JsonFloat64{F: 0.0},
	}

	testMessageType := &DynamicMessageType{
		spec:         generateTestSpec(fields),
		nested:       make(map[string]*DynamicMessageType),
		jsonPrealloc: 0,
	}

	testMessage := &DynamicMessage{
		dynamicType: testMessageType,
		data:        data,
	}

	verifyJSONMarshalling(t, testMessage)
}

func TestDynamicMessage_marshalJSON_nested(t *testing.T) {
	testMessageType, err := NewDynamicMessageType("geometry_msgs/Pose")

	if err != nil {
		t.Skip("test skipped because ROS environment not set up")
		return
	}

	testMessage := testMessageType.NewDynamicMessage()

	verifyJSONMarshalling(t, testMessage)
}

func TestDynamicMessage_marshalJSON_nestedWithTypeError(t *testing.T) {
	testMessageType, err := NewDynamicMessageType("geometry_msgs/Pose")

	if err != nil {
		t.Skip("test skipped because ROS environment not set up")
		return
	}

	testMessage := testMessageType.NewDynamicMessage()
	testMessage.data["position"] = float64(543.21) // We expect this to be geometry_msgs/Point type.

	if _, err := json.Marshal(testMessage); err == nil {
		t.Fatalf("expected type error")
	}
}

func TestDynamicMessage_marshalJSON_arrayOfNestedMessages(t *testing.T) {
	// We don't care about Pose in this step, but we want to load libgengo's context.
	_, err := NewDynamicMessageType("geometry_msgs/Pose")

	if err != nil {
		t.Skip("test skipped because ROS environment not set up")
		return
	}
	// Structure is z->[x, x].
	fields := []gengo.Field{
		*gengo.NewField("test", "uint8", "val", false, 0),
	}
	msgSpec := generateTestSpec(fields)
	context.RegisterMsg("test/x0Message", msgSpec)

	fields = []gengo.Field{
		*gengo.NewField("test", "x0Message", "x", true, 2),
	}
	msgSpec = generateTestSpec(fields)
	context.RegisterMsg("test/z0Message", msgSpec)

	testMessageType, err := NewDynamicMessageType("test/z0Message")

	if err != nil {
		t.Fatalf("Failed to create testMessageType, error: %v", err)
	}

	testMessage := testMessageType.NewDynamicMessage()

	verifyJSONMarshalling(t, testMessage)

	// Extra check: ensure type error is handled.
	testMessage.data["x"] = []float64{543.21, 98.76} // We expect this to be xMessage array.

	if _, err := json.Marshal(testMessage); err == nil {
		t.Fatalf("expected type error")
	}
}

// Testing helpers

func verifyJSONMarshalling(t *testing.T, msg *DynamicMessage) {
	defaultMarshalledBytes, err := json.Marshal(msg.data)
	if err != nil {
		t.Fatalf("failed to marshal raw data of dynamic message")
	}

	customMarshalledBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal dynamic message, err: %v, msg: %v", err, msg.data["x"])
	}

	defaultUnmarshalledMessage := msg.dynamicType.NewDynamicMessage()
	err = json.Unmarshal(defaultMarshalledBytes, defaultUnmarshalledMessage)
	if err != nil {
		t.Fatalf("failed to unmarshal default dynamic message, %v", err)
	}

	customUnmarshalledMessage := msg.dynamicType.NewDynamicMessage()
	err = json.Unmarshal(customMarshalledBytes, customUnmarshalledMessage)
	if err != nil {
		t.Fatalf("failed to unmarshal dynamic message, %v", err)
	}

	// We won't get a perfect match, because JSON promotes human-readability over numeric correctness, just check the fields match.
	for key := range msg.data {
		if _, ok := customUnmarshalledMessage.data[key]; ok == false {
			t.Fatalf("unmarshalled dynamic message missing key %v", key)
		}
	}

	if reflect.DeepEqual(defaultUnmarshalledMessage.data, customUnmarshalledMessage.data) == false {
		t.Fatalf("default and custom marshal mismatch. \n Default: %v \n Custom: %v", defaultUnmarshalledMessage.data, customUnmarshalledMessage.data)
	}

	// TODO: get working
	// if reflect.DeepEqual(msg.data, customUnmarshalledMessage.data) == false {
	// 	t.Fatalf("original and custom marshal mismatch. \n Original: %v \n Custom: %v \n Bytes: %v", msg.data, customUnmarshalledMessage.data, string(customMarshalledBytes))
	// }
}
