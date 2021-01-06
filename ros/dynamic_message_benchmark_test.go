package ros

import (
	"bytes"
	"testing"

	gengo "github.com/team-rocos/rosgo/libgengo"
)

// Singular value defintions
var singularMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
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
	})}

var singularSerialized []byte = []byte{
	0x12,       // u8
	0x56, 0x34, // u16
	0xde, 0xbc, 0x9a, 0x78, // u32
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u64
	0xfe,       // i8
	0xfe, 0xff, // i16
	0xfe, 0xff, 0xff, 0xff, // i32
	0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // i64
	0x01,                   // b
	0x2b, 0x52, 0x9a, 0x44, // f32
	0x98, 0xdd, 0x93, 0x87, 0x45, 0x4a, 0xc3, 0xc0, // f64
	0x05, 0x00, 0x00, 0x00, 'R', 'o', 'c', 'o', 's', // s
	0x0d, 0xf0, 0xed, 0xfe, 0xef, 0xbe, 0x37, 0x13, // t
	0x80, 0x70, 0x60, 0x50, 0x40, 0x30, 0x20, 0x10, // d
}

func BenchmarkDynamicMessage_Deserialize_SingularMedley(b *testing.B) {

	testMessage := singularMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(singularSerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_SingularMedley_New(b *testing.B) {

	testMessage := singularMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(singularSerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// Fixed array defintions
var fixedArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "uint8", "u8", true, 8),
		*gengo.NewField("Testing", "uint16", "u16", true, 4),
		*gengo.NewField("Testing", "uint32", "u32", true, 2),
		*gengo.NewField("Testing", "uint64", "u64", true, 1),
		*gengo.NewField("Testing", "int8", "i8", true, 8),
		*gengo.NewField("Testing", "int16", "i16", true, 4),
		*gengo.NewField("Testing", "int32", "i32", true, 2),
		*gengo.NewField("Testing", "int64", "i64", true, 1),
		*gengo.NewField("Testing", "bool", "b", true, 8),
		*gengo.NewField("Testing", "float32", "f32", true, 2),
		*gengo.NewField("Testing", "float64", "f64", true, 1),
		*gengo.NewField("Testing", "string", "s", true, 3),
		*gengo.NewField("Testing", "time", "t", true, 2),
		*gengo.NewField("Testing", "duration", "d", true, 2),
	})}

var fixedArraySerialized []byte = []byte{
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u8
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u16
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u32
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u64
	0xfe, 0xff, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, // i8
	0xfe, 0xff, 0xff, 0xff, 0x00, 0x00, 0x01, 0x00, // i16
	0xfe, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00, // i32
	0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // i64
	0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, // b
	0x2b, 0x52, 0x9a, 0x44, 0x2b, 0x52, 0x9a, 0x44, // f32
	0x98, 0xdd, 0x93, 0x87, 0x45, 0x4a, 0xc3, 0xc0, // f64
	0x05, 0x00, 0x00, 0x00, 'R', 'o', 'c', 'o', 's', // s[0]
	0x05, 0x00, 0x00, 0x00, 's', 'o', 'r', 'o', 'c', // s[1]
	0x05, 0x00, 0x00, 0x00, 'c', 'r', 'o', 'o', 's', // s[2]
	0x0d, 0xf0, 0xed, 0xfe, 0xef, 0xbe, 0x37, 0x13, // t[0]
	0xef, 0xbe, 0x37, 0x13, 0x0d, 0xf0, 0x37, 0x13, // t[1]
	0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x00, // d[0]
	0x80, 0x70, 0x60, 0x50, 0x40, 0x30, 0x20, 0x10, // d[1]
}

func BenchmarkDynamicMessage_Deserialize_FixedArrayMedley(b *testing.B) {

	testMessage := fixedArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(fixedArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_FixedArrayMedley_New(b *testing.B) {

	testMessage := fixedArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(fixedArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// Negative array sizes = dynamic arrays!
var dynamicArrayMessageType DynamicMessageType = DynamicMessageType{generateTestSpec([]gengo.Field{
	*gengo.NewField("Testing", "uint8", "u8", true, -1),
	*gengo.NewField("Testing", "uint16", "u16", true, -1),
	*gengo.NewField("Testing", "uint32", "u32", true, -1),
	*gengo.NewField("Testing", "uint64", "u64", true, -1),
	*gengo.NewField("Testing", "int8", "i8", true, -1),
	*gengo.NewField("Testing", "int16", "i16", true, -1),
	*gengo.NewField("Testing", "int32", "i32", true, -1),
	*gengo.NewField("Testing", "int64", "i64", true, -1),
	*gengo.NewField("Testing", "bool", "b", true, -1),
	*gengo.NewField("Testing", "float32", "f32", true, -1),
	*gengo.NewField("Testing", "float64", "f64", true, -1),
	*gengo.NewField("Testing", "string", "s", true, -1),
	*gengo.NewField("Testing", "time", "t", true, -1),
	*gengo.NewField("Testing", "duration", "d", true, -1),
})}

var dynamicArraySerialized []byte = []byte{
	0x08, 0x00, 0x00, 0x00, // array size
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u8
	0x04, 0x00, 0x00, 0x00, // array size
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u16
	0x02, 0x00, 0x00, 0x00, // array size
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u32
	0x01, 0x00, 0x00, 0x00, // array size
	0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12, // u64
	0x08, 0x00, 0x00, 0x00, // array size
	0xfe, 0xff, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, // i8
	0x04, 0x00, 0x00, 0x00, // array size
	0xfe, 0xff, 0xff, 0xff, 0x00, 0x00, 0x01, 0x00, // i16
	0x02, 0x00, 0x00, 0x00, // array size
	0xfe, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00, // i32
	0x01, 0x00, 0x00, 0x00, // array size
	0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // i64
	0x08, 0x00, 0x00, 0x00, // array size
	0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, // b
	0x02, 0x00, 0x00, 0x00, // array size
	0x2b, 0x52, 0x9a, 0x44, 0x2b, 0x52, 0x9a, 0x44, // f32
	0x01, 0x00, 0x00, 0x00, // array size
	0x98, 0xdd, 0x93, 0x87, 0x45, 0x4a, 0xc3, 0xc0, // f64
	0x03, 0x00, 0x00, 0x00, // array size
	0x05, 0x00, 0x00, 0x00, 'R', 'o', 'c', 'o', 's', // s[0]
	0x05, 0x00, 0x00, 0x00, 's', 'o', 'r', 'o', 'c', // s[1]
	0x05, 0x00, 0x00, 0x00, 'c', 'r', 'o', 'o', 's', // s[2]
	0x02, 0x00, 0x00, 0x00, // array size
	0x0d, 0xf0, 0xed, 0xfe, 0xef, 0xbe, 0x37, 0x13, // t[0]
	0xef, 0xbe, 0x37, 0x13, 0x0d, 0xf0, 0x37, 0x13, // t[1]
	0x02, 0x00, 0x00, 0x00, // array size
	0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x00, // d[0]
	0x80, 0x70, 0x60, 0x50, 0x40, 0x30, 0x20, 0x10, // d[1]
}

func BenchmarkDynamicMessage_Deserialize_DynamicArrayMedley(b *testing.B) {

	testMessage := dynamicArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(dynamicArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_DynamicArrayMedley_New(b *testing.B) {

	testMessage := dynamicArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(dynamicArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

//
// Biiig array benchmarks!
//
var bigArraySerialized []byte = make([]byte, 1_000_000)

// bool Big array defintions
var boolBigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "bool", "b", true, 1_000_000),
	})}

func BenchmarkDynamicMessage_Deserialize_boolBigArray(b *testing.B) {

	testMessage := boolBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_boolBigArray_New(b *testing.B) {

	testMessage := boolBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// int8 Big array defintions
var int8BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "int8", "i8", true, 1_000_000),
	})}

func BenchmarkDynamicMessage_Deserialize_int8BigArray(b *testing.B) {

	testMessage := int8BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_int8BigArray_New(b *testing.B) {

	testMessage := int8BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// int16 Big array defintions
var int16BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "int16", "i16", true, 500_000),
	})}

func BenchmarkDynamicMessage_Deserialize_int16BigArray(b *testing.B) {

	testMessage := int16BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_int16BigArray_New(b *testing.B) {

	testMessage := int16BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// int32 Big array defintions
var int32BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "int32", "i32", true, 250_000),
	})}

func BenchmarkDynamicMessage_Deserialize_int32BigArray(b *testing.B) {

	testMessage := int32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_int32BigArray_New(b *testing.B) {

	testMessage := int32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// int64 Big array defintions
var int64BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "int64", "i64", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_int64BigArray(b *testing.B) {

	testMessage := int64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_int64BigArray_New(b *testing.B) {

	testMessage := int64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// uint8 Big array defintions
var uint8BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "uint8", "u8", true, 1_000_000),
	})}

func BenchmarkDynamicMessage_Deserialize_uint8BigArray(b *testing.B) {

	testMessage := uint8BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_uint8BigArray_New(b *testing.B) {

	testMessage := uint8BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// uint16 Big array defintions
var uint16BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "uint16", "u16", true, 500_000),
	})}

func BenchmarkDynamicMessage_Deserialize_uint16BigArray(b *testing.B) {

	testMessage := uint16BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_uint16BigArray_New(b *testing.B) {

	testMessage := uint16BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// uint32 Big array defintions
var uint32BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "uint32", "u32", true, 250_000),
	})}

func BenchmarkDynamicMessage_Deserialize_uint32BigArray(b *testing.B) {

	testMessage := uint32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_uint32BigArray_New(b *testing.B) {

	testMessage := uint32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// uint64 Big array defintions
var uint64BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "uint64", "u64", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_uint64BigArray(b *testing.B) {

	testMessage := uint64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_uint64BigArray_New(b *testing.B) {

	testMessage := uint64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// float32 Big array defintions
var float32BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "float32", "f32", true, 250_000),
	})}

func BenchmarkDynamicMessage_Deserialize_float32BigArray(b *testing.B) {

	testMessage := float32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_float32BigArray_New(b *testing.B) {

	testMessage := float32BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// float64 Big array defintions
var float64BigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "float64", "f64", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_float64BigArray(b *testing.B) {

	testMessage := float64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_float64BigArray_New(b *testing.B) {

	testMessage := float64BigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// string Big array defintions
var stringBigArraySerialized []byte = bytes.Repeat([]byte{0x04, 0x00, 0x00, 0x00, 'J', 'o', 'J', 'o'}, 125_000)

var stringBigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "string", "s", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_stringBigArray(b *testing.B) {

	testMessage := stringBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(stringBigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_stringBigArray_New(b *testing.B) {

	testMessage := stringBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(stringBigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// time Big array defintions
var timeBigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "time", "t", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_timeBigArray(b *testing.B) {
	testMessage := timeBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_timeBigArray_New(b *testing.B) {
	testMessage := timeBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// duration Big array defintions
var durationBigArrayMessageType DynamicMessageType = DynamicMessageType{
	generateTestSpec([]gengo.Field{
		*gengo.NewField("Testing", "duration", "d", true, 125_000),
	})}

func BenchmarkDynamicMessage_Deserialize_durationBigArray(b *testing.B) {
	testMessage := durationBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.Deserialize(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_Deserialize_durationBigArray_New(b *testing.B) {
	testMessage := durationBigArrayMessageType.NewDynamicMessage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		byteReader := bytes.NewReader(bigArraySerialized)
		if err := testMessage.DeserializeNew(byteReader); err != nil {
			b.Fatalf("Deserialize failed %s", err)
		}
	}
}

// Benchmarks on NewDynamicMessage

func BenchmarkDynamicMessage_NewMessage_Singular(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := singularMessageType.NewDynamicMessage()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_Singular_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := singularMessageType.NewDynamicMessageNew()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_FixedArray(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := fixedArrayMessageType.NewDynamicMessage()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_FixedArray_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := fixedArrayMessageType.NewDynamicMessageNew()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_DynamicArray(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := dynamicArrayMessageType.NewDynamicMessage()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_DynamicArray_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := dynamicArrayMessageType.NewDynamicMessageNew()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_BigArray(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := uint16BigArrayMessageType.NewDynamicMessage()
		_ = testMessage.data
	}
}

func BenchmarkDynamicMessage_NewMessage_BigArray_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testMessage := uint16BigArrayMessageType.NewDynamicMessageNew()
		_ = testMessage.data
	}
}
