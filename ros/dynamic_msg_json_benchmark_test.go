package ros

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/pkg/errors"
)

// Benchmarks on JSON marshalling.

var singularMessageData map[string]interface{} = map[string]interface{}{
	"u8":  uint8(0x12),
	"u16": uint16(0x3456),
	"u32": uint32(0x789abcde),
	"u64": uint64(0x123456789abcdef0),
	"i8":  int8(-2),
	"i16": int16(-2),
	"i32": int32(-2),
	"i64": int64(-2),
	"b":   true,
	"f32": JsonFloat32{1234.5678},
	"f64": JsonFloat64{-9876.5432},
	"s":   "Rocos",
	"t":   NewTime(0xfeedf00d, 0x1337beef),
	"d":   NewDuration(0x50607080, 0x10203040),
}

func BenchmarkDynamicMessage_JSONMarshal_SingularPrimitives_default(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(singularMessageData)
		if err != nil {
			b.Fatalf("marshal failed %s", err)
		}
	}
}

func BenchmarkDynamicMessage_JSONMarshal_SingularPrimitives_custom(b *testing.B) {
	testMessage := singularMessageType.NewDynamicMessage()
	testMessage.data = singularMessageData

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(testMessage)
		if err != nil {
			b.Fatalf("marshal failed %s", err)
		}
	}
}

// Experimental stuff

type typePair struct {
	name string
	t    string
	e    myTypeEnum
}

type CustomMap struct {
	d    map[string]interface{}
	spec []typePair
}

func (m *CustomMap) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0)

	for i, t := range m.spec {
		if i == 0 {
			buf = append(buf, byte('{'))
		} else {
			buf = append(buf, byte(','))
		}
		buf = strconv.AppendQuote(buf, t.name)
		buf = append(buf, byte(':'))
		v, ok := m.d[t.name]
		if !ok {
			return nil, errors.Wrap(errors.New("key not in data"), "key: "+t.name)
		}
		switch t.t {
		case "uint8":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "uint16":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "uint32":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "uint64":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "int8":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "int16":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "int32":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "int64":
			buf = strconv.AppendInt(buf, v.(int64), 10)
		case "string":
			buf = strconv.AppendQuote(buf, v.(string))
		case "float32":
			return nil, errors.Wrap(errors.New("unsupported type"), "type: "+t.t)
		case "float64":
			buf = strconv.AppendFloat(buf, v.(float64), byte('f'), -1, 64)
		}
	}

	buf = append(buf, byte('}'))

	return buf, nil
}

var jsonMapData map[string]interface{} = map[string]interface{}{
	"int":  int64(5),
	"str":  "my_string",
	"f64":  float64(64.5),
	"0f64": float64(64.5),
	"1f64": float64(64.5),
	"2f64": float64(64.5),
	"3f64": float64(64.5),
	"4f64": float64(64.5),
	"5f64": float64(64.5),
	"6f64": float64(64.5),
}

var jsonMapDataSpec []typePair = []typePair{
	{t: "int64", e: MyInt64, name: "int"},
	{t: "string", e: MyString, name: "str"},
	{t: "float64", e: MyFloat64, name: "f64"},
	{t: "float64", e: MyFloat64, name: "0f64"},
	{t: "float64", e: MyFloat64, name: "1f64"},
	{t: "float64", e: MyFloat64, name: "2f64"},
	{t: "float64", e: MyFloat64, name: "3f64"},
	{t: "float64", e: MyFloat64, name: "4f64"},
	{t: "float64", e: MyFloat64, name: "5f64"},
	{t: "float64", e: MyFloat64, name: "6f64"},
}

func Benchmark_JSONMap_Custom(b *testing.B) {
	m := &CustomMap{d: jsonMapData, spec: jsonMapDataSpec}
	var buf []byte
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err = json.Marshal(m)
		if err != nil {
			b.Fatalf("marshal failed %s", err)
		}
	}
	b.StopTimer()
	b.Log(string(buf))
}

func Benchmark_JSONMap_Default(b *testing.B) {
	data := jsonMapData
	var buf []byte
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err = json.Marshal(data)
		if err != nil {
			b.Fatalf("marshal failed %s", err)
		}
	}
	b.StopTimer()
	b.Log(string(buf))
}

func getTypeString(str string) string {
	switch str {
	case "u8":
		return "1+u8"
	case "u16":
		return "2+u16"
	case "u32":
		return "3+u32"
	case "u64":
		return "4+u64"
	case "i8":
		return "1+i8"
	case "i16":
		return "2+i16"
	case "i32":
		return "3+i32"
	case "i64":
		return "4+i64"
	case "f32":
		return "3+f32"
	case "f64":
		return "4+f64"
	default:
		return "none"
	}
}

type myTypeEnum int

const (
	MyUint8 myTypeEnum = iota
	MyUint16
	MyUint32
	MyUint64
	MyInt8
	MyInt16
	MyInt32
	MyInt64
	MyFloat32
	MyFloat64
	MyString
)

func getTypeEnum(num myTypeEnum) string {
	switch num {
	case MyUint8:
		return "1+u8"
	case MyUint16:
		return "2+u16"
	case MyUint32:
		return "3+u32"
	case MyUint64:
		return "4+u64"
	case MyInt8:
		return "1+i8"
	case MyInt16:
		return "2+i16"
	case MyInt32:
		return "3+i32"
	case MyInt64:
		return "4+i64"
	case MyFloat32:
		return "3+f32"
	case MyFloat64:
		return "4+f64"
	default:
		return "none"
	}
}

func BenchmarkSwitchCase_StringLookup(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getTypeString("u8")
		_ = getTypeString("u16")
		_ = getTypeString("u32")
		_ = getTypeString("u64")
		_ = getTypeString("i8")
		_ = getTypeString("i16")
		_ = getTypeString("i32")
		_ = getTypeString("i64")
		_ = getTypeString("f32")
		_ = getTypeString("f64")

	}
}

func BenchmarkSwitchCase_EnumLookup(b *testing.B) {

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getTypeEnum(MyUint8)
		_ = getTypeEnum(MyUint16)
		_ = getTypeEnum(MyUint32)
		_ = getTypeEnum(MyUint64)
		_ = getTypeEnum(MyInt8)
		_ = getTypeEnum(MyInt16)
		_ = getTypeEnum(MyInt32)
		_ = getTypeEnum(MyInt64)
		_ = getTypeEnum(MyFloat32)
		_ = getTypeEnum(MyFloat64)

	}
}
