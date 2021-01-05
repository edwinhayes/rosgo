package ros

import (
	"bytes"
	"math"
	"testing"

	gengo "github.com/team-rocos/rosgo/libgengo"
)

func Float32Near(expected float32, actual float32, tol float32) bool {
	return math.Abs(float64(expected-actual)) < float64(tol)
}

func generateTestSpec(fields []gengo.Field) *gengo.MsgSpec {
	msgSpec := &gengo.MsgSpec{}
	msgSpec.FullName = "TestMessage"
	msgSpec.Package = "Testing"
	msgSpec.MD5Sum = "1337beeffeed1337"
	msgSpec.ShortName = "Test"
	msgSpec.Fields = fields
	return msgSpec
}

// Simple test to prove that we have a test spec that we can start scrutinizing
func TestDynamicMessage_TypeGetters(t *testing.T) {
	fields := []gengo.Field{
		*gengo.NewField("Testing", "float32", "x", false, 0),
	}
	testMessageType := DynamicMessageType{generateTestSpec(fields)}

	if testMessageType.Name() != "TestMessage" {
		t.Fatalf("DynamicMessageType has undexpected Name %s", testMessageType.Name())
	}

	if testMessageType.MD5Sum() != "1337beeffeed1337" {
		t.Fatalf("DynamicMessageType has undexpected MD5Sum %s", testMessageType.MD5Sum())
	}
}

func TestDynamicMessage_Deserialize_Simple(t *testing.T) {
	fields := []gengo.Field{
		*gengo.NewField("Testing", "float32", "x", false, 0),
	}
	testMessageType := DynamicMessageType{generateTestSpec(fields)}

	// Using IEEE754 https://www.h-schmidt.net/FloatConverter/IEEE754.html
	// 1234.5678 = 0x449a522b
	// Then convert to little-endian
	expected := float32(1234.5678)
	byteReader := bytes.NewReader([]byte{0x2b, 0x52, 0x9a, 0x44})

	testMessage := testMessageType.NewDynamicMessage()

	if err := testMessage.Deserialize(byteReader); err != nil {
		t.Fatalf("Deserialize failed %s", err)
	}

	xWrapped, ok := testMessage.data["x"]
	if ok == false {
		t.Fatalf("Deserialize failed to extract x, got %s", testMessage.data)
	}

	x, ok := xWrapped.(JsonFloat32)
	if ok == false {
		t.Fatalf("x is not a float32, got %s", testMessage.data)
	}

	if !Float32Near(expected, x.F, 1e-4) {
		t.Fatalf("x (%f) is not near %f", x, expected)
	}
}

func TestDynamicMessage_Deserialize_SingularMedley(t *testing.T) {
	fields := []gengo.Field{
		*gengo.NewField("Testing", "uint8", "u8", false, 0),
		*gengo.NewField("Testing", "uint16", "u16", false, 0),
		*gengo.NewField("Testing", "uint32", "u32", false, 0),
		*gengo.NewField("Testing", "uint64", "u64", false, 0),
		*gengo.NewField("Testing", "int8", "i8", false, 0),
		*gengo.NewField("Testing", "int16", "i16", false, 0),
		*gengo.NewField("Testing", "int32", "i32", false, 0),
		*gengo.NewField("Testing", "int64", "i64", false, 0),
		*gengo.NewField("Testing", "bool", "b", false, 0),
		*gengo.NewField("Testing", "float32", "f32", false, 0),
		*gengo.NewField("Testing", "float64", "f64", false, 0),
		*gengo.NewField("Testing", "string", "s", false, 0),
		*gengo.NewField("Testing", "time", "t", false, 0),
		*gengo.NewField("Testing", "duration", "d", false, 0),
	}
	testMessageType := DynamicMessageType{generateTestSpec(fields)}

	// Using IEEE754 https://www.h-schmidt.net/FloatConverter/IEEE754.html
	// 1234.5678 = 0x449a522b
	// Then convert to little-endian
	var expected = map[string]interface{}{
		"u8":  uint8(0x12),
		"u16": uint16(0x3456),
		"u32": uint32(0x789abcde),
		"u64": uint64(0x123456789abcdef0),
		"i8":  int8(-2),
		"i16": int16(-2),
		"i32": int32(-2),
		"i64": int64(-2),
		"b":   true,
		"f32": float32(1234.5678),
		"f64": float64(-9876.5432), //0xC0C3 4A45 8793 DD98
		"s":   "Rocos",
		"t":   NewTime(0xfeedf00d, 0x1337beef),
		"d":   NewDuration(0xfeffffff, 0xfdffffff), // -2.000000002 sec
	}

	byteReader := bytes.NewReader([]byte{
		0x12,       // u8
		0x56, 0x32, // u16
		0xde, 0xbc, 0x9a, 0x78, // u32
		0xf0, 0xde, 0xbc, 0x9a, 0x9a, 0x56, 0x32, 0x12, // u64
		0xfe,       // i8
		0xfe, 0xff, // i16
		0xfe, 0xff, 0xff, 0xff, // i32
		0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // i64
		0x01,                   // b
		0x2b, 0x52, 0x9a, 0x44, // f32
		0x98, 0xdd, 0x93, 0x87, 0x45, 0x4a, 0xc3, 0xc0, // f64
		0x05, 0x00, 0x00, 0x00, 'R', 'o', 'c', 'o', 's', // s
		0x0d, 0xf0, 0xed, 0xfe, 0xef, 0xbe, 0x37, 0x13, // t
		0xfe, 0xff, 0xff, 0xff, 0xfd, 0xff, 0xff, 0xff, // d
	})

	testMessage := testMessageType.NewDynamicMessage()

	if err := testMessage.Deserialize(byteReader); err != nil {
		t.Fatalf("Deserialize failed %s", err)
	}

	// Check u8
	if _, ok := testMessage.data["u8"]; !ok {
		t.Fatalf("Deserialize failed to extract x, got %s", testMessage.data)
	}

	if expected["u8"] != testMessage.data["u8"].(uint8) {
		t.Fatalf("expected 0x%x != result 0x%x", expected["u8"], testMessage.data["u8"].(uint8))
	}
}
