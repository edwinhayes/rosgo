package ros

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	gengo "github.com/team-rocos/rosgo/libgengo"
)

func TestDynamicMessage_marshalJSON_primitives(t *testing.T) {
	testCases := []struct {
		fields     []gengo.Field
		data       map[string]interface{}
		marshalled string
	}{
		// Singular Values.
		// - Unsigned integers.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint8", "u8", false, 0)},
			data:       map[string]interface{}{"u8": uint8(0x12)},
			marshalled: `{"u8":18}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint16", "u16", false, 0)},
			data:       map[string]interface{}{"u16": uint16(0x8001)},
			marshalled: `{"u16":32769}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint32", "u32", false, 0)},
			data:       map[string]interface{}{"u32": uint32(0x80000001)},
			marshalled: `{"u32":2147483649}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint64", "u64", false, 0)},
			data:       map[string]interface{}{"u64": uint64(0x8000000000001)}, // Note: Can only represent up to 52-bits due to JSON number representation!
			marshalled: `{"u64":2251799813685249}`,
		},
		// - Signed integers.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int8", "i8", false, 0)},
			data:       map[string]interface{}{"i8": int8(-20)},
			marshalled: `{"i8":-20}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int16", "i16", false, 0)},
			data:       map[string]interface{}{"i16": int16(-20_000)},
			marshalled: `{"i16":-20000}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int32", "i32", false, 0)},
			data:       map[string]interface{}{"i32": int32(-2_000_000_000)},
			marshalled: `{"i32":-2000000000}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int64", "i64", false, 0)},
			data:       map[string]interface{}{"i64": int64(-2_000_000_000_000_000_000)},
			marshalled: `{"i64":-2000000000000000000}`,
		},
		// - Booleans.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "bool", "b", false, 0)},
			data:       map[string]interface{}{"b": false},
			marshalled: `{"b":false}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "bool", "b", false, 0)},
			data:       map[string]interface{}{"b": true},
			marshalled: `{"b":true}`,
		},
		// - Floats. TODO: Move other float tests into this test
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", false, 0)},
			data:       map[string]interface{}{"f32": JsonFloat32{-1.125}},
			marshalled: `{"f32":-1.125}`,
		},
		// {
		// 	fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", false, 0)},
		// 	data:       map[string]interface{}{"f32": JsonFloat32{float32(math.NaN())}},
		// 	marshalled: `{"f32":"nan"}`,
		// },
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", false, 0)},
			data:       map[string]interface{}{"f32": JsonFloat32{float32(math.Inf(1))}},
			marshalled: `{"f32":"+inf"}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", false, 0)},
			data:       map[string]interface{}{"f32": JsonFloat32{float32(math.Inf(-1))}},
			marshalled: `{"f32":"-inf"}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", false, 0)},
			data:       map[string]interface{}{"f64": JsonFloat64{-1.125}},
			marshalled: `{"f64":-1.125}`,
		},
		// {
		// 	fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", false, 0)},
		// 	data:       map[string]interface{}{"f64": JsonFloat64{math.NaN()}},
		// 	marshalled: `{"f64":"nan"}`,
		// },
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", false, 0)},
			data:       map[string]interface{}{"f64": JsonFloat64{math.Inf(1)}},
			marshalled: `{"f64":"+inf"}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", false, 0)},
			data:       map[string]interface{}{"f64": JsonFloat64{math.Inf(-1)}},
			marshalled: `{"f64":"-inf"}`,
		},
		// - Strings. TODO: Bring other string test cases here.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "string", "s", false, 0)},
			data:       map[string]interface{}{"s": ""},
			marshalled: `{"s":""}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "string", "s", false, 0)},
			data:       map[string]interface{}{"s": "N0t  empty "},
			marshalled: `{"s":"N0t  empty "}`,
		},
		// - Time and Duration.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "time", "t", false, 0)},
			data:       map[string]interface{}{"t": NewTime(0xfeedf00d, 0x1337beef)},
			marshalled: `{"t":{"Sec":4277006349,"NSec":322420463}}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "duration", "d", false, 0)},
			data:       map[string]interface{}{"d": NewDuration(0x40302010, 0x00706050)},
			marshalled: `{"d":{"Sec":1076895760,"NSec":7364688}}`,
		},
		// Fixed and Dynamic arrays.
		// - Unsigned integers.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint8", "u8", true, 8)},
			data:       map[string]interface{}{"u8": []uint8{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}},
			marshalled: `{"u8":"8N68mnhWNBI="}`, // From https://base64.guru/converter/encode/hex
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint16", "u16", true, 5)},
			data:       map[string]interface{}{"u16": []uint16{0xf0de, 0xbc9a, 0x7856, 0x3412, 0x0}},
			marshalled: `{"u16":[61662,48282,30806,13330,0]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint32", "u32", true, 3)},
			data:       map[string]interface{}{"u32": []uint32{0xf0debc9a, 0x78563412, 0x0}},
			marshalled: `{"u32":[4041129114,2018915346,0]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint64", "u64", true, 3)},
			data:       map[string]interface{}{"u64": []uint64{0x8000000000001, 0x78563412, 0x0}},
			marshalled: `{"u64":[2251799813685249,2018915346,0]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint8", "u8", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"u8": []uint8{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12}},
			marshalled: `{"u8":"8N68mnhWNBI="}`, // From https://base64.guru/converter/encode/hex
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint16", "u16", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"u16": []uint16{0xf0de, 0xbc9a, 0x7856, 0x3412, 0x0}},
			marshalled: `{"u16":[61662,48282,30806,13330,0]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint32", "u32", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"u32": []uint32{0xf0debc9a, 0x78563412, 0x0}},
			marshalled: `{"u32":[4041129114,2018915346,0]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "uint64", "u64", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"u64": []uint64{0x8000000000001, 0x78563412, 0x0}},
			marshalled: `{"u64":[2251799813685249,2018915346,0]}`,
		},
		// - Signed integers.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int8", "i8", true, 8)},
			data:       map[string]interface{}{"i8": []int8{-128, -55, -1, 0, 1, 7, 77, 127}},
			marshalled: `{"i8":[-128,-55,-1,0,1,7,77,127]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int16", "i16", true, 7)},
			data:       map[string]interface{}{"i16": []int16{-32768, -129, -1, 0, 1, 128, 32767}},
			marshalled: `{"i16":[-32768,-129,-1,0,1,128,32767]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int32", "i32", true, 9)},
			data:       map[string]interface{}{"i32": []int32{-2147483648, -32768, -129, -1, 0, 1, 128, 32767, 2147483647}},
			marshalled: `{"i32":[-2147483648,-32768,-129,-1,0,1,128,32767,2147483647]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int64", "i64", true, 11)},
			data:       map[string]interface{}{"i64": []int64{-2251799813685249, -2147483648, -32768, -129, -1, 0, 1, 128, 32767, 2147483647, 2251799813685249}},
			marshalled: `{"i64":[-2251799813685249,-2147483648,-32768,-129,-1,0,1,128,32767,2147483647,2251799813685249]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int8", "i8", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"i8": []int8{-128, -55, -1, 0, 1, 7, 77, 127}},
			marshalled: `{"i8":[-128,-55,-1,0,1,7,77,127]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int16", "i16", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"i16": []int16{-32768, -129, -1, 0, 1, 128, 32767}},
			marshalled: `{"i16":[-32768,-129,-1,0,1,128,32767]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int32", "i32", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"i32": []int32{-2147483648, -32768, -129, -1, 0, 1, 128, 32767, 2147483647}},
			marshalled: `{"i32":[-2147483648,-32768,-129,-1,0,1,128,32767,2147483647]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "int64", "i64", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"i64": []int64{-2251799813685249, -2147483648, -32768, -129, -1, 0, 1, 128, 32767, 2147483647, 2251799813685249}},
			marshalled: `{"i64":[-2251799813685249,-2147483648,-32768,-129,-1,0,1,128,32767,2147483647,2251799813685249]}`,
		},
		// - Booleans.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "bool", "b", true, 2)},
			data:       map[string]interface{}{"b": []bool{true, false}},
			marshalled: `{"b":[true,false]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "bool", "b", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"b": []bool{true, false}},
			marshalled: `{"b":[true,false]}`,
		},
		// - Floats.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", true, 4)},
			data:       map[string]interface{}{"f32": []JsonFloat32{{-1.125}, {3.3e3}, {7.7e7}, {9.9e-9}, {float32(math.Inf(1))}, {float32(math.Inf(-1))}}},
			marshalled: `{"f32":[-1.125,3300,7.7e+07,9.9e-09,"+inf","-inf"]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", true, 7)},
			data:       map[string]interface{}{"f64": []JsonFloat64{{-1.125}, {3.3e3}, {7.7e7}, {9.9e-9}, {math.Inf(1)}, {math.Inf(-1)}}},
			marshalled: `{"f64":[-1.125,3300,7.7e+07,9.9e-09,"+inf","-inf"]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float32", "f32", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"f32": []JsonFloat32{{-1.125}, {3.3e3}, {7.7e7}, {9.9e-9}, {float32(math.Inf(1))}, {float32(math.Inf(-1))}}},
			marshalled: `{"f32":[-1.125,3300,7.7e+07,9.9e-09,"+inf","-inf"]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "float64", "f64", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"f64": []JsonFloat64{{-1.125}, {3.3e3}, {7.7e7}, {9.9e-9}, {math.Inf(1)}, {math.Inf(-1)}}},
			marshalled: `{"f64":[-1.125,3300,7.7e+07,9.9e-09,"+inf","-inf"]}`,
		},
		// - Strings.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "string", "s", true, 5)},
			data:       map[string]interface{}{"s": []string{"", "n0t empty  ", "new\nline", "\ttabbed", "s\\ash"}},
			marshalled: `{"s":["","n0t empty  ","new\nline","\ttabbed","s\\ash"]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "string", "s", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"s": []string{"", "n0t empty  ", "new\nline", "\ttabbed", "s\\ash"}},
			marshalled: `{"s":["","n0t empty  ","new\nline","\ttabbed","s\\ash"]}`,
		},
		// - Time.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "time", "t", true, 2)},
			data:       map[string]interface{}{"t": []Time{NewTime(0xfeedf00d, 0x1337beef), NewTime(0x1337beef, 0x00706050)}},
			marshalled: `{"t":[{"Sec":4277006349,"NSec":322420463},{"Sec":322420463,"NSec":7364688}]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "time", "t", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"t": []Time{NewTime(0xfeedf00d, 0x1337beef), NewTime(0x1337beef, 0x00706050)}},
			marshalled: `{"t":[{"Sec":4277006349,"NSec":322420463},{"Sec":322420463,"NSec":7364688}]}`,
		},
		// - Duration.
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "duration", "d", true, 2)},
			data:       map[string]interface{}{"d": []Duration{NewDuration(0xfeedf00d, 0x1337beef), NewDuration(0x1337beef, 0x00706050)}},
			marshalled: `{"d":[{"Sec":4277006349,"NSec":322420463},{"Sec":322420463,"NSec":7364688}]}`,
		},
		{
			fields:     []gengo.Field{*gengo.NewField("Testing", "duration", "d", true, -1)}, // Dynamic.
			data:       map[string]interface{}{"d": []Duration{NewDuration(0xfeedf00d, 0x1337beef), NewDuration(0x1337beef, 0x00706050)}},
			marshalled: `{"d":[{"Sec":4277006349,"NSec":322420463},{"Sec":322420463,"NSec":7364688}]}`,
		},
	}

	for _, testCase := range testCases {

		testMessageType := &DynamicMessageType{
			spec:         generateTestSpec(testCase.fields),
			nested:       make(map[string]*DynamicMessageType),
			jsonPrealloc: 0,
		}

		testMessage := &DynamicMessage{
			dynamicType: testMessageType,
			data:        testCase.data,
		}

		marshalled, err := json.Marshal(testMessage)
		if err != nil {
			t.Fatalf("failed to marshal dynamic message\n expected: %v\nerr: %v", testCase.marshalled, err)
		}
		if string(marshalled) != testCase.marshalled {
			t.Fatalf("marshalled data does not equal expected\nmarshalled: %v\nexpected: %v", string(marshalled), testCase.marshalled)
		}

		defaultMarshalled, err := json.Marshal(testMessage)
		if string(defaultMarshalled) != string(marshalled) || err != nil {
			t.Fatalf("library marshalling sanity check failed\nmarshalled: %v\nexpected: %v", string(marshalled), testCase.marshalled)
		}

		unmarshalledMessage := testMessageType.NewDynamicMessage()

		if err := json.Unmarshal(marshalled, unmarshalledMessage); err != nil {
			t.Fatalf("failed to unmarshal dynamic message\n json: %v\nerr: %v", testCase.marshalled, err)
		}

		if reflect.DeepEqual(testMessage.data, unmarshalledMessage.data) == false {
			t.Fatalf("original and unmarshalled data mismatch. \n Original: %v \n Unmarshalled: %v \n json: %v", testMessage.data, unmarshalledMessage.data, string(marshalled))
		}

	}
}

func TestDynamicMessage_marshalJSON_primitiveSet(t *testing.T) {

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
