package ros

// IMPORT REQUIRED PACKAGES.

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/team-rocos/rosgo/libgengo"
)

// DEFINE PUBLIC STRUCTURES.

// DEFINE PRIVATE STRUCTURES.

// DEFINE PUBLIC GLOBALS.

// DEFINE PRIVATE GLOBALS.

// DEFINE PUBLIC STATIC FUNCTIONS.

// DEFINE PUBLIC RECEIVER FUNCTIONS.

//	DynamicMessageType

// GenerateJSONSchema generates a (primitive) JSON schema for the associated DynamicMessageType; however note that since
// we are mostly interested in making schema's for particular _topics_, the function takes a string prefix, and string topic name, which are
// used to id the resulting schema.
func (t *DynamicMessageType) GenerateJSONSchema(prefix string, topic string) ([]byte, error) {
	// The JSON schema for a message consist of the (recursive) properties names/types:
	schemaItems, err := t.generateJSONSchemaProperties(prefix + topic)
	if err != nil {
		return nil, err
	}

	// Plus some extra keywords:
	schemaItems["$schema"] = "https://json-schema.org/draft-07/schema#"
	schemaItems["$id"] = prefix + topic

	// The schema itself is created from the map of properties.
	schemaString, err := json.Marshal(schemaItems)
	if err != nil {
		return nil, err
	}

	// All done.
	return schemaString, nil
}

func (t *DynamicMessageType) generateJSONSchemaProperties(topic string) (map[string]interface{}, error) {
	// Each message's schema indicates that it is an 'object' with some nested properties: those properties are the fields and their types.
	properties := make(map[string]interface{})
	schemaItems := make(map[string]interface{})
	schemaItems["type"] = "object"
	schemaItems["properties"] = properties

	// Iterate over each of the fields in the message.
	for _, field := range t.spec.Fields {
		if field.IsArray {
			// It's an array.
			propertyContent := make(map[string]interface{})
			properties[field.Name] = propertyContent

			if field.GoType == "uint8" {
				propertyContent["type"] = "string"
			} else {
				// Arrays all have a type of 'array', regardless of that the hold, then the 'item' keyword determines what type goes in the array.
				propertyContent["type"] = "array"
				arrayItems := make(map[string]interface{})
				propertyContent["items"] = arrayItems

				// Need to handle each type appropriately.
				if field.IsBuiltin {
					if field.Type == "string" {
						arrayItems["type"] = "string"
					} else if field.Type == "time" {
						timeItems := make(map[string]interface{})
						timeItems["sec"] = map[string]string{"type": "integer"}
						timeItems["nsec"] = map[string]string{"type": "integer"}
						arrayItems["type"] = "object"
						arrayItems["properties"] = timeItems
					} else if field.Type == "duration" {
						timeItems := make(map[string]interface{})
						timeItems["sec"] = map[string]string{"type": "integer"}
						timeItems["nsec"] = map[string]string{"type": "integer"}
						arrayItems["type"] = "object"
						arrayItems["properties"] = timeItems
					} else {
						// It's a primitive.
						var jsonType string
						if field.GoType == "int8" || field.GoType == "int16" || field.GoType == "uint16" ||
							field.GoType == "int32" || field.GoType == "uint32" || field.GoType == "int64" || field.GoType == "uint64" {
							jsonType = "integer"
						} else if field.GoType == "float32" || field.GoType == "float64" {
							jsonType = "number"
						} else if field.GoType == "bool" {
							jsonType = "bool"
						} else {
							// Something went wrong.
							return nil, errors.New("we haven't implemented this primitive yet")
						}
						arrayItems["type"] = jsonType
					}

				} else {
					// It's another nested message.

					// Generate the nested type.
					msgType, err := newDynamicMessageTypeNested(field.Type, field.Package, nil, nil)
					if err != nil {
						return nil, errors.Wrap(err, "Schema Field: "+field.Name)
					}

					// Recursively generate schema information for the nested type.
					schemaElement, err := msgType.generateJSONSchemaProperties(topic + Sep + field.Name)
					if err != nil {
						return nil, errors.Wrap(err, "Schema Field:"+field.Name)
					}
					arrayItems["type"] = schemaElement
				}
			}
		} else {
			// It's a scalar.
			if field.IsBuiltin {
				propertyContent := make(map[string]interface{})
				properties[field.Name] = propertyContent

				if field.Type == "string" {
					propertyContent["type"] = "string"
				} else if field.Type == "time" {
					timeItems := make(map[string]interface{})
					timeItems["sec"] = map[string]string{"type": "integer"}
					timeItems["nsec"] = map[string]string{"type": "integer"}
					propertyContent["type"] = "object"
					propertyContent["properties"] = timeItems
				} else if field.Type == "duration" {
					timeItems := make(map[string]interface{})
					timeItems["sec"] = map[string]string{"type": "integer"}
					timeItems["nsec"] = map[string]string{"type": "integer"}
					propertyContent["type"] = "object"
					propertyContent["properties"] = timeItems
				} else {
					// It's a primitive.
					var jsonType string
					if field.GoType == "int8" || field.GoType == "uint8" || field.GoType == "int16" || field.GoType == "uint16" ||
						field.GoType == "int32" || field.GoType == "uint32" || field.GoType == "int64" || field.GoType == "uint64" {
						jsonType = "integer"
						jsonType = "integer"
						jsonType = "integer"
					} else if field.GoType == "float32" || field.GoType == "float64" {
						jsonType = "number"
					} else if field.GoType == "bool" {
						jsonType = "bool"
					} else {
						// Something went wrong.
						return nil, errors.New("we haven't implemented this primitive yet")
					}
					propertyContent["type"] = jsonType
				}
			} else {
				// It's another nested message.

				// Generate the nested type.
				msgType, err := newDynamicMessageTypeNested(field.Type, field.Package, nil, nil)
				if err != nil {
					return nil, errors.Wrap(err, "Schema Field: "+field.Name)
				}

				// Recursively generate schema information for the nested type.
				schemaElement, err := msgType.generateJSONSchemaProperties(topic + Sep + field.Name)
				if err != nil {
					return nil, errors.Wrap(err, "Schema Field:"+field.Name)
				}
				properties[field.Name] = schemaElement
			}
		}
	}

	// All done.
	return schemaItems, nil
}

//	DynamicMessage

// MarshalJSON provides a custom implementation of JSON marshalling, so that when the DynamicMessage is recursively
// marshalled using the standard Go json.marshal() mechanism, the resulting JSON representation is a compact representation
// of just the important message payload (and not the message definition).  It's important that this representation matches
// the schema generated by GenerateJSONSchema().
func (m *DynamicMessage) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, m.dynamicType.jsonPrealloc)

	buf = append(buf, byte('{'))
	for i, field := range m.dynamicType.spec.Fields {
		if i > 0 {
			buf = append(buf, byte(','))
		}
		buf = strconv.AppendQuote(buf, field.Name)
		buf = append(buf, byte(':'))
		v, ok := m.data[field.Name]
		if !ok {
			return nil, errors.Wrap(errors.New("key not in data"), "key: "+field.Name)
		}

		if field.IsArray {
			if field.IsBuiltin == false {
				// The type encapsulates an array of ROS messages, so we marshal the DynamicMessages.
				if nestedArray, ok := v.([]Message); ok {
					buf = append(buf, byte('['))
					for i, nested := range nestedArray {
						if i > 0 {
							buf = append(buf, byte(','))
						}
						nestedBuf, err := nested.(*DynamicMessage).MarshalJSON()
						if err != nil {
							return nil, errors.Wrap(err, "field: "+field.Name)
						}
						buf = append(buf, nestedBuf...)
					}
					buf = append(buf, byte(']'))
				} else {
					return nil, errors.Wrap(errors.New("unknown type"), "Field: "+field.Name)
				}
				continue
			}
			switch field.GoType {
			case "bool":
				buf = append(buf, byte('['))
				for i, item := range v.([]bool) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					if item == true {
						buf = append(buf, []byte("true")...)
					} else {
						buf = append(buf, []byte("false")...)
					}
				}
				buf = append(buf, byte(']'))
			case "int8":
				buf = append(buf, byte('['))
				items := v.([]int8)
				for i := range items {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendInt(buf, int64(items[i]), 10)
				}
				buf = append(buf, byte(']'))
			case "int16":
				buf = append(buf, byte('['))
				for i, item := range v.([]int16) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendInt(buf, int64(item), 10)
				}
				buf = append(buf, byte(']'))
			case "int32":
				buf = append(buf, byte('['))
				for i, item := range v.([]int32) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendInt(buf, int64(item), 10)
				}
				buf = append(buf, byte(']'))
			case "int64":
				buf = append(buf, byte('['))
				for i, item := range v.([]int64) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendInt(buf, item, 10)
				}
				buf = append(buf, byte(']'))
			case "uint8":
				buf = append(buf, byte('"'))
				uint8Slice := v.([]uint8)
				encodedLen := base64.StdEncoding.EncodedLen(len(uint8Slice))
				if (cap(buf) - len(buf)) > encodedLen {
					dst := buf[len(buf) : len(buf)+encodedLen]
					base64.StdEncoding.Encode(dst, uint8Slice)
					buf = buf[:len(buf)+encodedLen]
				} else {
					dst := make([]byte, encodedLen) // TODO: for biig arrays, see if we can avoid allocating this slice
					base64.StdEncoding.Encode(dst, uint8Slice)
					buf = append(buf, dst...)
				}
				buf = append(buf, byte('"'))
			case "uint16":
				buf = append(buf, byte('['))
				for i, item := range v.([]uint16) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendUint(buf, uint64(item), 10)
				}
				buf = append(buf, byte(']'))
			case "uint32":
				buf = append(buf, byte('['))
				for i, item := range v.([]uint32) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendUint(buf, uint64(item), 10)
				}
				buf = append(buf, byte(']'))
			case "uint64":
				buf = append(buf, byte('['))
				for i, item := range v.([]uint64) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendUint(buf, item, 10)
				}
				buf = append(buf, byte(']'))
			case "float32":
				buf = append(buf, byte('['))
				for i, item := range v.([]JsonFloat32) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					f := float64(item.F)
					if math.IsNaN(f) {
						buf = strconv.AppendQuote(buf, "nan")
					} else if math.IsInf(f, 1) {
						buf = strconv.AppendQuote(buf, "+inf")
					} else if math.IsInf(f, -1) {
						buf = strconv.AppendQuote(buf, "-inf")
					} else {
						buf = strconv.AppendFloat(buf, f, byte('g'), -1, 32)
					}
				}
				buf = append(buf, byte(']'))
			case "float64":
				buf = append(buf, byte('['))
				for i, item := range v.([]JsonFloat64) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					f := item.F
					if math.IsNaN(f) {
						buf = strconv.AppendQuote(buf, "nan")
					} else if math.IsInf(f, 1) {
						buf = strconv.AppendQuote(buf, "+inf")
					} else if math.IsInf(f, -1) {
						buf = strconv.AppendQuote(buf, "-inf")
					} else {
						buf = strconv.AppendFloat(buf, f, byte('g'), -1, 64)
					}
				}
				buf = append(buf, byte(']'))
			case "string":
				buf = append(buf, byte('['))
				for i, item := range v.([]string) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = strconv.AppendQuote(buf, item)
				}
				buf = append(buf, byte(']'))
			case "ros.Time":
				buf = append(buf, byte('['))
				for i, item := range v.([]Time) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = append(buf, []byte("{\"Sec\":")...)
					buf = strconv.AppendUint(buf, uint64(item.Sec), 10)
					buf = append(buf, []byte(",\"NSec\":")...)
					buf = strconv.AppendUint(buf, uint64(item.NSec), 10)
					buf = append(buf, byte('}'))
				}
				buf = append(buf, byte(']'))
			case "ros.Duration":
				buf = append(buf, byte('['))
				for i, item := range v.([]Duration) {
					if i > 0 {
						buf = append(buf, byte(','))
					}
					buf = append(buf, []byte("{\"Sec\":")...)
					buf = strconv.AppendUint(buf, uint64(item.Sec), 10)
					buf = append(buf, []byte(",\"NSec\":")...)
					buf = strconv.AppendUint(buf, uint64(item.NSec), 10)
					buf = append(buf, byte('}'))
				}
				buf = append(buf, byte(']'))
			default:
				// Something went wrong.
				return nil, errors.Wrap(errors.New("unknown builtin type"), "field: "+field.Name)
			}
			continue
		}

		if field.IsBuiltin == false {
			// The type encapsulates another ROS message, so we marshal the DynamicMessage.
			if nested, ok := v.(*DynamicMessage); ok {
				nestedBuf, err := nested.MarshalJSON()
				if err != nil {
					return nil, errors.Wrap(err, "field: "+field.Name)
				}
				buf = append(buf, nestedBuf...)
			} else {
				return nil, errors.Wrap(errors.New("unknown type"), "Field: "+field.Name)
			}
			continue
		}
		switch field.GoType {
		case "bool":
			if v.(bool) {
				buf = append(buf, []byte("true")...)
			} else {
				buf = append(buf, []byte("false")...)
			}
		case "int8":
			buf = strconv.AppendInt(buf, int64(v.(int8)), 10)
		case "int16":
			buf = strconv.AppendInt(buf, int64(v.(int16)), 10)
		case "int32":
			buf = strconv.AppendInt(buf, int64(v.(int32)), 10)
		case "int64":
			buf = strconv.AppendInt(buf, v.(int64), 10)
		case "uint8":
			buf = strconv.AppendUint(buf, uint64(v.(uint8)), 10)
		case "uint16":
			buf = strconv.AppendUint(buf, uint64(v.(uint16)), 10)
		case "uint32":
			buf = strconv.AppendUint(buf, uint64(v.(uint32)), 10)
		case "uint64":
			buf = strconv.AppendUint(buf, v.(uint64), 10)
		case "float32":
			f := float64(v.(JsonFloat32).F)
			if math.IsNaN(f) {
				buf = strconv.AppendQuote(buf, "nan")
			} else if math.IsInf(f, 1) {
				buf = strconv.AppendQuote(buf, "+inf")
			} else if math.IsInf(f, -1) {
				buf = strconv.AppendQuote(buf, "-inf")
			} else {
				buf = strconv.AppendFloat(buf, f, byte('g'), -1, 32)
			}
		case "float64":
			f := v.(JsonFloat64).F
			if math.IsNaN(f) {
				buf = strconv.AppendQuote(buf, "nan")
			} else if math.IsInf(f, 1) {
				buf = strconv.AppendQuote(buf, "+inf")
			} else if math.IsInf(f, -1) {
				buf = strconv.AppendQuote(buf, "-inf")
			} else {
				buf = strconv.AppendFloat(buf, f, byte('g'), -1, 64)
			}
		case "string":
			buf = strconv.AppendQuote(buf, v.(string))
		case "ros.Time":
			buf = append(buf, []byte("{\"Sec\":")...)
			buf = strconv.AppendUint(buf, uint64(v.(Time).Sec), 10)
			buf = append(buf, []byte(",\"NSec\":")...)
			buf = strconv.AppendUint(buf, uint64(v.(Time).NSec), 10)
			buf = append(buf, byte('}'))
		case "ros.Duration":
			buf = append(buf, []byte("{\"Sec\":")...)
			buf = strconv.AppendUint(buf, uint64(v.(Duration).Sec), 10)
			buf = append(buf, []byte(",\"NSec\":")...)
			buf = strconv.AppendUint(buf, uint64(v.(Duration).NSec), 10)
			buf = append(buf, byte('}'))
		default:
			// Something went wrong.
			return nil, errors.Wrap(errors.New("unknown builtin type"), "field: "+field.Name)
		}
	}

	buf = append(buf, byte('}'))

	if length := len(buf); length > m.dynamicType.jsonPrealloc {
		m.dynamicType.jsonPrealloc = length
	}

	return buf, nil
}

//UnmarshalJSON provides a custom implementation of JSON unmarshalling. Using the dynamicMessage provided, Msgspec is used to
//determine the individual parsing of each JSON encoded payload item into the correct Go type. It is important each type is
//correct so that the message serializes correctly and is understood by the ROS system
func (m *DynamicMessage) UnmarshalJSON(buf []byte) error {

	//Delcaring temp variables to be used across the unmarshaller
	var err error
	var goField libgengo.Field
	var keyName []byte
	var oldMsgType string
	var msg *DynamicMessage
	var msgType *DynamicMessageType
	var data interface{}
	var fieldExists bool

	//Declaring jsonparser unmarshalling functions
	var arrayHandler func([]byte, jsonparser.ValueType, int, error)
	var objectHandler func([]byte, []byte, jsonparser.ValueType, int) error

	//JSON key is an array
	arrayHandler = func(key []byte, dataType jsonparser.ValueType, offset int, err error) {
		switch dataType.String() {
		//We have a string array
		case "string":
			if goField.GoType == "float32" || goField.GoType == "float64" {
				data, err = strconv.ParseFloat(string(key), 64)
				if err != nil {
					errors.Wrap(err, "Field: "+goField.Name)
				}
				if goField.GoType == "float32" {
					m.data[goField.Name] = append(m.data[goField.Name].([]JsonFloat32), JsonFloat32{F: float32((data.(float64)))})
				} else {
					m.data[goField.Name] = append(m.data[goField.Name].([]JsonFloat64), JsonFloat64{F: data.(float64)})
				}
			} else {
				buf := make([]byte, 0, len(key)+2)
				buf = append(buf, '"')
				buf = append(buf, key...)
				buf = append(buf, '"')
				unquoted, err := strconv.Unquote(string(buf))
				if err != nil {
					errors.Wrap(err, "Field: "+goField.Name)
				}
				m.data[goField.Name] = append(m.data[goField.Name].([]string), unquoted)
			}
		//We have a number or int array.
		case "number":
			//We have a float to parse
			if goField.GoType == "float64" || goField.GoType == "float32" {
				data, err = strconv.ParseFloat(string(key), 64)
				if err != nil {
					errors.Wrap(err, "Field: "+goField.Name)
				}
			} else {
				data, err = strconv.ParseInt(string(key), 0, 64)
				if err != nil {
					errors.Wrap(err, "Field: "+goField.Name)
				}
			}
			//Append field to data array
			switch goField.GoType {
			case "int8":
				m.data[goField.Name] = append(m.data[goField.Name].([]int8), int8((data.(int64))))
			case "int16":
				m.data[goField.Name] = append(m.data[goField.Name].([]int16), int16((data.(int64))))
			case "int32":
				m.data[goField.Name] = append(m.data[goField.Name].([]int32), int32((data.(int64))))
			case "int64":
				m.data[goField.Name] = append(m.data[goField.Name].([]int64), int64((data.(int64))))
			case "uint8":
				m.data[goField.Name] = append(m.data[goField.Name].([]uint8), uint8((data.(int64))))
			case "uint16":
				m.data[goField.Name] = append(m.data[goField.Name].([]uint16), uint16((data.(int64))))
			case "uint32":
				m.data[goField.Name] = append(m.data[goField.Name].([]uint32), uint32((data.(int64))))
			case "uint64":
				m.data[goField.Name] = append(m.data[goField.Name].([]uint64), uint64((data.(int64))))
			case "float32":
				m.data[goField.Name] = append(m.data[goField.Name].([]JsonFloat32), JsonFloat32{F: float32((data.(float64)))})
			case "float64":
				m.data[goField.Name] = append(m.data[goField.Name].([]JsonFloat64), JsonFloat64{F: data.(float64)})
			}
		// We have a bool array.
		case "boolean":
			data, err := jsonparser.ParseBoolean(key)
			_ = err
			m.data[goField.Name] = append(m.data[goField.Name].([]bool), data)
		//We have an object array
		case "object":
			switch goField.GoType {
			//We have a time object
			case "ros.Time":
				tmpTime := Time{}
				sec, err := jsonparser.GetInt(key, "Sec")
				nsec, err := jsonparser.GetInt(key, "NSec")
				if err == nil {
					tmpTime.Sec = uint32(sec)
					tmpTime.NSec = uint32(nsec)
				}
				m.data[goField.Name] = append(m.data[goField.Name].([]Time), tmpTime)
			//We have a duration object
			case "ros.Duration":
				tmpDuration := Duration{}
				sec, err := jsonparser.GetInt(key, "Sec")
				nsec, err := jsonparser.GetInt(key, "NSec")
				if err == nil {
					tmpDuration.Sec = uint32(sec)
					tmpDuration.NSec = uint32(nsec)
				}
				m.data[goField.Name] = append(m.data[goField.Name].([]Duration), tmpDuration)
			//We have a nested message
			default:
				newMsgType := goField.GoType
				//Check if the message type is the same as last iteration
				//This avoids generating a new type for each array item
				if oldMsgType != "" && oldMsgType == newMsgType {
					//We've already generated this type
				} else {
					msgT, err := newDynamicMessageTypeNested(goField.Type, goField.Package, nil, nil)
					_ = err
					msgType = msgT
				}
				msg = msgType.NewMessage().(*DynamicMessage)
				err = msg.UnmarshalJSON(key)

				if msgArray, ok := m.data[goField.Name].([]Message); !ok {
					errors.Wrap(errors.New("unable to convert to []Message"), "Field: "+goField.Name)
				} else {
					m.data[goField.Name] = append(msgArray, msg)
				}

				//Store msg type
				oldMsgType = newMsgType
				//No error handling in array, see next comment
				_ = err

			}
		}

		//Null error as it is not returned in ArrayEach, requires package modification
		_ = err
		//Null keyName to prevent repeat scenarios of same key usage
		_ = keyName

	}

	//JSON key handler
	objectHandler = func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		//Store keyName for usage in ArrayEach function
		keyName = key
		fieldExists = false

		// Confirm the pointers are valid
		if m == nil {
			return errors.New("nil pointer to DynamicMessage")
		} else if m.dynamicType == nil {
			return errors.New("nil pointer to dynamicType")
		} else if m.dynamicType.spec == nil {
			return errors.New("nil pointer to MsgSpec")
		} else if m.dynamicType.spec.Fields == nil {
			return errors.New("nil pointer to Fields")
		}

		//Find message spec field that matches JSON key
		for _, field := range m.dynamicType.spec.Fields {
			if string(key) == field.Name {
				goField = field
				fieldExists = true
			}
		}
		if fieldExists == true {
			//Scalars First
			switch dataType.String() {
			//We have a JSON string
			case "string":
				//Special case where we have a byte array encoded as JSON string
				if goField.GoType == "uint8" {
					data, err := base64.StdEncoding.DecodeString(string(value))
					if err != nil {
						return errors.Wrap(err, "Byte Array Field: "+goField.Name)
					}
					m.data[goField.Name] = data
					//Case where we have marshalled a special float as a string
				} else if goField.GoType == "float32" || goField.GoType == "float64" {
					data, err = strconv.ParseFloat(string(value), 64)
					if err != nil {
						errors.Wrap(err, "Field: "+goField.Name)
					}
					if goField.GoType == "float32" {
						m.data[goField.Name] = JsonFloat32{F: float32(data.(float64))}
					} else {
						m.data[goField.Name] = JsonFloat64{F: data.(float64)}
					}
				} else {
					m.data[goField.Name] = string(value)
				}
			//We have a JSON number or int
			case "number":
				//We have a float to parse
				if goField.GoType == "float64" || goField.GoType == "float32" {
					data, err = jsonparser.GetFloat(buf, string(key))
					if err != nil {
						return errors.Wrap(err, "Field: "+goField.Name)
					}
					//We have an int to parse
				} else {
					data, err = jsonparser.GetInt(buf, string(key))
					if err != nil {
						return errors.Wrap(err, "Field: "+goField.Name)
					}
				}
				//Copy number value to message field
				switch goField.GoType {
				case "int8":
					m.data[goField.Name] = int8(data.(int64))
				case "int16":
					m.data[goField.Name] = int16(data.(int64))
				case "int32":
					m.data[goField.Name] = int32(data.(int64))
				case "int64":
					m.data[goField.Name] = int64(data.(int64))
				case "uint8":
					m.data[goField.Name] = uint8(data.(int64))
				case "uint16":
					m.data[goField.Name] = uint16(data.(int64))
				case "uint32":
					m.data[goField.Name] = uint32(data.(int64))
				case "uint64":
					m.data[goField.Name] = uint64(data.(int64))
				case "float32":
					m.data[goField.Name] = JsonFloat32{F: float32(data.(float64))}
				case "float64":
					m.data[goField.Name] = JsonFloat64{F: data.(float64)}
				}
			//We have a JSON bool
			case "boolean":
				data, err := jsonparser.GetBoolean(buf, string(key))
				if err != nil {
					return errors.Wrap(err, "Field: "+goField.Name)
				}
				m.data[goField.Name] = data
			//We have a JSON object
			case "object":
				switch goField.GoType {
				//We have a time object
				case "ros.Time":
					tmpTime := Time{}
					sec, err := jsonparser.GetInt(value, "Sec")
					nsec, err := jsonparser.GetInt(value, "NSec")
					if err == nil {
						tmpTime.Sec = uint32(sec)
						tmpTime.NSec = uint32(nsec)
					}
					m.data[goField.Name] = tmpTime
				//We have a duration object
				case "ros.Duration":
					tmpDuration := Duration{}
					sec, err := jsonparser.GetInt(value, "Sec")
					nsec, err := jsonparser.GetInt(value, "NSec")
					if err == nil {
						tmpDuration.Sec = uint32(sec)
						tmpDuration.NSec = uint32(nsec)
					}
					m.data[goField.Name] = tmpDuration
				default:
					//We have a nested message
					msgType, err := newDynamicMessageTypeNested(goField.Type, goField.Package, nil, nil)
					if err != nil {
						return errors.Wrap(err, "Field: "+goField.Name)
					}
					msg := msgType.NewMessage().(*DynamicMessage)
					if err = msg.UnmarshalJSON(value); err != nil {
						return errors.Wrap(err, "Field: "+goField.Name)
					}
					m.data[goField.Name] = msg
				}
			//We have a JSON array
			case "array":
				//Redeclare message array fields incase they do not exist
				switch goField.GoType {
				case "bool":
					m.data[goField.Name] = make([]bool, 0)
				case "int8":
					m.data[goField.Name] = make([]int8, 0)
				case "int16":
					m.data[goField.Name] = make([]int16, 0)
				case "int32":
					m.data[goField.Name] = make([]int32, 0)
				case "int64":
					m.data[goField.Name] = make([]int64, 0)
				case "uint8":
					m.data[goField.Name] = make([]uint8, 0)
				case "uint16":
					m.data[goField.Name] = make([]uint16, 0)
				case "uint32":
					m.data[goField.Name] = make([]uint32, 0)
				case "uint64":
					m.data[goField.Name] = make([]uint64, 0)
				case "float32":
					m.data[goField.Name] = make([]JsonFloat32, 0)
				case "float64":
					m.data[goField.Name] = make([]JsonFloat64, 0)
				case "string":
					m.data[goField.Name] = make([]string, 0)
				case "ros.Time":
					m.data[goField.Name] = make([]Time, 0)
				case "ros.Duration":
					m.data[goField.Name] = make([]Duration, 0)
				default:
					//goType is a nested Message array
					m.data[goField.Name] = make([]Message, 0)
				}
				//Parse JSON array
				jsonparser.ArrayEach(value, arrayHandler)
			default:
				//We do nothing here as blank fields may return value type NotExist or Null
				err = errors.Wrap(err, "Null field: "+string(key))
			}
		} else {
			return errors.New("Field Unknown: " + string(key))
		}
		return err
	}
	//Perform JSON object handler function
	err = jsonparser.ObjectEach(buf, objectHandler)
	return err
}

// DEFINE PRIVATE STATIC FUNCTIONS.

// DEFINE PRIVATE RECEIVER FUNCTIONS.

// ALL DONE.
