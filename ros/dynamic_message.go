package ros

// IMPORT REQUIRED PACKAGES.

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
	"github.com/team-rocos/rosgo/libgengo"
)

// DEFINE PUBLIC STRUCTURES.

// DynamicMessageType abstracts the schema of a ROS Message whose schema is only known at runtime.  DynamicMessageTypes are created by looking up the relevant schema information from
// ROS Message definition files.  DynamicMessageType implements the rosgo MessageType interface, allowing it to be used throughout rosgo in the same manner as message schemas generated
// at compiletime by gengo.
type DynamicMessageType struct {
	spec   *libgengo.MsgSpec
	nested map[string]*DynamicMessageType // Map with key string = messageType name.
}

// DynamicMessage abstracts an instance of a ROS Message whose type is only known at runtime.  The schema of the message is denoted by the referenced DynamicMessageType, while the
// actual payload of the Message is stored in a map[string]interface{} which maps the name of each field to its value.  DynamicMessage implements the rosgo Message interface, allowing
// it to be used throughout rosgo in the same manner as message types generated at compiletime by gengo.
type DynamicMessage struct {
	dynamicType *DynamicMessageType
	data        map[string]interface{}
}

// DEFINE PRIVATE STRUCTURES.

type dynamicMessageLike interface {
	Serialize(buf *bytes.Buffer) error
	GetDynamicType() *DynamicMessageType
}

// GetDynamicType returns the DynamicMessageType of a DynamicMessage
func (m *DynamicMessage) GetDynamicType() *DynamicMessageType {
	return m.dynamicType
}

// DEFINE PUBLIC GLOBALS.

// DEFINE PRIVATE GLOBALS.

var rosPkgPath string // Colon separated list of paths to search for message definitions on.

var context *libgengo.PkgContext // We'll try to preserve a single message context to avoid reloading each time.

// DEFINE PUBLIC STATIC FUNCTIONS.

// SetRuntimePackagePath sets the ROS package search path which will be used by DynamicMessage to look up ROS message definitions at runtime.
func SetRuntimePackagePath(path string) {
	// We're not going to check that the result is valid, we'll just accept it blindly.
	rosPkgPath = path
	// Reset the message context
	ResetContext()
	// All done.
	return
}

// GetRuntimePackagePath returns the ROS package search path which will be used by DynamicMessage to look up ROS message definitions at runtime.  By default, this will
// be equivalent to the ${ROS_PACKAGE_PATH} environment variable.
func GetRuntimePackagePath() string {
	// If a package path hasn't been set at the time of first use, by default we'll just use the ROS environment default.
	if rosPkgPath == "" {
		rosPkgPath = os.Getenv("ROS_PACKAGE_PATH")
	}
	// All done.
	return rosPkgPath
}

// ResetContext resets the package path context so that a new one will be generated
func ResetContext() {
	context = nil
}

// NewDynamicMessageType generates a DynamicMessageType corresponding to the specified typeName from the available ROS message definitions; typeName should be a fully-qualified
// ROS message type name.  The first time the function is run, a message 'context' is created by searching through the available ROS message definitions, then the ROS message to
// be used for the definition is looked up by name.  On subsequent calls, the ROS message type is looked up directly from the existing context.
func NewDynamicMessageType(typeName string) (*DynamicMessageType, error) {
	t, err := newDynamicMessageTypeNested(typeName, "", nil, nil)
	return t, err
}

// NewDynamicMessageTypeLiteral generates a DynamicMessageType, and returns a copy of the generated type. This is required by DynamicAction.
func NewDynamicMessageTypeLiteral(typeName string) (DynamicMessageType, error) {
	t, err := NewDynamicMessageType(typeName)
	return *t, err
}

// newDynamicMessageTypeNested generates a DynamicMessageType from the available ROS message definitions.  The first time the function is run, a message 'context' is created by
// searching through the available ROS message definitions, then the ROS message type to use for the defintion is looked up by name.  On subsequent calls, the ROS message type
// is looked up directly from the existing context.  This 'nested' version of the function is able to be called recursively, where packageName should be the typeName of the
// parent ROS message; this is used internally for handling complex ROS messages.
func newDynamicMessageTypeNested(typeName string, packageName string, nested map[string]*DynamicMessageType, nestedChain map[string]struct{}) (*DynamicMessageType, error) {

	// Create an empty message type.
	m := &DynamicMessageType{}

	if nested == nil {
		nested = make(map[string]*DynamicMessageType)
	}
	if nestedChain == nil {
		nestedChain = make(map[string]struct{})
	}

	m.nested = nested

	// The nestedChain is used for recursion detection.
	if _, ok := nestedChain[typeName]; ok {
		// DynamicMessageType already created, recursive messages are scary, return error!
		return nil, errors.New("type already in nested map, message is recursive")
	}

	// Just return with the messageType if it has been defined already
	if t, ok := nested[typeName]; ok {
		return t, nil
	}

	nestedChain[typeName] = struct{}{}

	// If we haven't created a message context yet, better do that.
	if context == nil {
		// Create context for our ROS install.
		c, err := libgengo.NewPkgContext(strings.Split(GetRuntimePackagePath(), ":"))
		if err != nil {
			return m, err
		}
		context = c
	}

	// We need to try to look up the full name, in case we've just been given a short name.
	fullname := typeName

	// The Header type has some special treatment.
	if typeName == "Header" {
		fullname = "std_msgs/Header"
	} else {
		_, ok := context.GetMsgs()[fullname]
		if !ok {
			// Seems like the package_name we were give wasn't the full name.

			// Messages in the same package are allowed to use relative names, so try prefixing the package.
			if packageName != "" {
				fullname = packageName + "/" + fullname
			}
		}
	}

	// Load context for the target message.
	spec, err := context.LoadMsg(fullname)
	if err != nil {
		return m, err
	}

	// Now we know all about the message!
	m.spec = spec

	// Register type in the nested map, this prevents recursion.
	nested[typeName] = m
	m.nested = nested

	// Generate the spec for any nested messages.
	for _, field := range spec.Fields {
		if field.IsBuiltin == false {
			newType, err := newDynamicMessageTypeNested(field.Type, field.Package, nested, nestedChain)
			if err != nil {
				return m, err
			}
			m.nested[field.Name] = newType
		}
	}

	// Unravelling the nested chain, we are done.
	delete(nestedChain, typeName)

	// We've successfully made a new message type matching the requested ROS type.
	return m, nil
}

// DEFINE PUBLIC RECEIVER FUNCTIONS.

//	DynamicMessageType

// Name returns the full ROS name of the message type; required for ros.MessageType.
func (t *DynamicMessageType) Name() string {
	if t.spec == nil {
		return ""
	}
	return t.spec.FullName
}

// Text returns the full ROS message specification for this message type; required for ros.MessageType.
func (t *DynamicMessageType) Text() string {
	if t.spec == nil {
		return ""
	}
	return t.spec.Text
}

// MD5Sum returns the ROS compatible MD5 sum of the message type; required for ros.MessageType.
func (t *DynamicMessageType) MD5Sum() string {
	if t.spec == nil {
		return ""
	}
	return t.spec.MD5Sum
}

// NewMessage creates a new DynamicMessage instantiating the message type; required for ros.MessageType.
func (t *DynamicMessageType) NewMessage() Message {
	// Don't instantiate messages for incomplete types.
	return t.NewDynamicMessage()
}

// NewDynamicMessage constructs a DynamicMessage with zeroed fields.
func (t *DynamicMessageType) NewDynamicMessage() *DynamicMessage {
	// But otherwise, make a new one.
	d := &DynamicMessage{}
	d.dynamicType = t

	var err error
	d.data, err = t.zeroValueData()
	if err != nil {
		return d
	}
	return d
}

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

// Data returns the data map field of the DynamicMessage
func (m *DynamicMessage) Data() map[string]interface{} {
	return m.data
}

// Type returns the ROS type of a dynamic message; required for ros.Message.
func (m *DynamicMessage) Type() MessageType {
	return m.dynamicType
}

// MarshalJSON provides a custom implementation of JSON marshalling, so that when the DynamicMessage is recursively
// marshalled using the standard Go json.marshal() mechanism, the resulting JSON representation is a compact representation
// of just the important message payload (and not the message definition).  It's important that this representation matches
// the schema generated by GenerateJSONSchema().
func (m *DynamicMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.data)
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
				m.data[goField.Name] = append(m.data[goField.Name].([]string), string(key))
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
		//We have a bool array
		case "boolean":
			data, err := jsonparser.GetBoolean(buf, string(key))
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

// Serialize converts a DynamicMessage into a TCPROS bytestream allowing it to be published to other nodes; required for ros.Message.
func (m *DynamicMessage) Serialize(buf *bytes.Buffer) error {
	if m.dynamicType.spec == nil {
		return errors.New("dynamic message type spec is nil")
	}
	if m.dynamicType.nested == nil {
		return errors.New("dynamic message type nested is nil")
	}

	// THIS METHOD IS BASICALLY AN UNTEMPLATED COPY OF THE TEMPLATE IN LIBGENGO.

	var err error = nil

	// Iterate over each of the fields in the message.
	for _, field := range m.dynamicType.spec.Fields {

		if field.IsArray {
			// It's an array.

			// Look up the item.
			array, ok := m.data[field.Name]
			if !ok {
				return errors.New("Field: " + field.Name + ": No data found.")
			}

			if (reflect.ValueOf(array).Kind() != reflect.Array) && (reflect.ValueOf(array).Kind() != reflect.Slice) {
				return errors.New("Field: " + field.Name + ": expected an array.")
			}

			// If the array is not a fixed length, it begins with a declaration of the array size.
			var size uint32
			if field.ArrayLen < 0 {
				size = uint32(reflect.ValueOf(array).Len())
				if err := binary.Write(buf, binary.LittleEndian, size); err != nil {
					return errors.Wrap(err, "Field: "+field.Name)
				}
			} else {
				size = uint32(field.ArrayLen)

				// Make sure that the 'fixed length' array that is expected is the correct length. Pad it if necessary.
				reflectLen := uint32(reflect.ValueOf(array).Len())
				if reflectLen < size {
					array, err = padArray(array, field, reflectLen, size)
					if err != nil {
						return errors.Wrap(err, "unable to pad array to correct length")
					}
				}
			}

			// Then we just write out all the elements one after another.
			arrayValue := reflect.ValueOf(array)
			for i := uint32(0); i < size; i++ {
				//Casting the array item to interface type
				var arrayItem interface{} = arrayValue.Index(int(i)).Interface()
				// Need to handle each type appropriately.
				if field.IsBuiltin {
					if field.Type == "string" {
						// Make sure we've actually got a string.
						str, ok := arrayItem.(string)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected string.")
						}
						// The string should start with a declaration of the number of characters.
						var sizeStr uint32 = uint32(len(str))
						if err := binary.Write(buf, binary.LittleEndian, sizeStr); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
						// Then write out the actual characters.
						data := []byte(str)
						if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}

					} else if field.Type == "time" {
						// Make sure we've actually got a time.
						t, ok := arrayItem.(Time)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected ros.Time.")
						}
						// Then write out the structure.
						if err := binary.Write(buf, binary.LittleEndian, t); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}

					} else if field.Type == "duration" {
						// Make sure we've actually got a duration.
						d, ok := arrayItem.(Duration)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected ros.Duration.")
						}
						// Then write out the structure.
						if err := binary.Write(buf, binary.LittleEndian, d); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}

					} else {
						// It's a primitive.

						// Because no runtime expressions in type assertions in Go, we need to manually do this.
						switch field.GoType {
						case "bool":
							// Make sure we've actually got a bool.
							v, ok := arrayItem.(bool)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected bool.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "int8":
							// Make sure we've actually got an int8.
							v, ok := arrayItem.(int8)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected int8.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "int16":
							// Make sure we've actually got an int16.
							v, ok := arrayItem.(int16)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected int16.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "int32":
							// Make sure we've actually got an int32.
							v, ok := arrayItem.(int32)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected int32.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "int64":
							// Make sure we've actually got an int64.
							v, ok := arrayItem.(int64)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected int64.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "uint8":
							// Make sure we've actually got a uint8.
							v, ok := arrayItem.(uint8)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected uint8.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "uint16":
							// Make sure we've actually got a uint16.
							v, ok := arrayItem.(uint16)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected uint16.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "uint32":
							// Make sure we've actually got a uint32.
							v, ok := arrayItem.(uint32)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected uint32.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "uint64":
							// Make sure we've actually got a uint64.
							v, ok := arrayItem.(uint64)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected uint64.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "float32":
							// Make sure we've actually got a float32.
							v, ok := arrayItem.(JsonFloat32)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected JsonFloat32.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v.F); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						case "float64":
							// Make sure we've actually got a float64.
							v, ok := arrayItem.(JsonFloat64)
							if !ok {
								return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected JsonFloat64.")
							}
							// Then write out the value.
							if err := binary.Write(buf, binary.LittleEndian, v.F); err != nil {
								return errors.Wrap(err, "Field: "+field.Name)
							}
						default:
							// Something went wrong.
							return errors.New("we haven't implemented this primitive yet")
						}
					}

				} else {
					// Else it's not a builtin.

					// Confirm the message we're holding is actually the correct type.
					msg, ok := arrayItem.(dynamicMessageLike)
					if !ok {
						return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(arrayItem).Name() + ", expected Message.")
					}
					if msg == nil || msg.GetDynamicType() == nil || msg.GetDynamicType().spec == nil {
						return errors.New("Field: " + field.Name + ": nil pointer to MsgSpec")
					}
					if msg.GetDynamicType().spec.ShortName != field.Type {
						return errors.New("Field: " + field.Name + ": Found msg " + msg.GetDynamicType().spec.ShortName + ", expected " + field.Type + ".")
					}
					// Otherwise, we just recursively serialise it.
					if err = msg.Serialize(buf); err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}
				}
			}

		} else {
			// It's a scalar.

			// Look up the item.
			item, ok := m.data[field.Name]
			if !ok {
				return errors.New("Field: " + field.Name + ": No data found.")
			}

			// Need to handle each type appropriately.
			if field.IsBuiltin {
				if field.Type == "string" {
					// Make sure we've actually got a string.
					str, ok := item.(string)
					if !ok {
						return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected string.")
					}
					// The string should start with a declaration of the number of characters.
					var sizeStr uint32 = uint32(len(str))
					if err := binary.Write(buf, binary.LittleEndian, sizeStr); err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}
					// Then write out the actual characters.
					data := []byte(str)
					if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}

				} else if field.Type == "time" {
					// Make sure we've actually got a time.
					t, ok := item.(Time)
					if !ok {
						return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected ros.Time.")
					}
					// Then write out the structure.
					if err := binary.Write(buf, binary.LittleEndian, t); err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}

				} else if field.Type == "duration" {
					// Make sure we've actually got a duration.
					d, ok := item.(Duration)
					if !ok {
						return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected ros.Duration.")
					}
					// Then write out the structure.
					if err := binary.Write(buf, binary.LittleEndian, d); err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}

				} else {
					// It's a primitive.

					// Because no runtime expressions in type assertions in Go, we need to manually do this.
					switch field.GoType {
					case "bool":
						// Make sure we've actually got a bool.
						v, ok := item.(bool)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected bool.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "int8":
						// Make sure we've actually got an int8.
						v, ok := item.(int8)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected int8.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "int16":
						// Make sure we've actually got an int16.
						v, ok := item.(int16)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected int16.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "int32":
						// Make sure we've actually got an int32.
						v, ok := item.(int32)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected int32.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "int64":
						// Make sure we've actually got an int64.
						v, ok := item.(int64)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected int64.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "uint8":
						// Make sure we've actually got a uint8.
						v, ok := item.(uint8)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected uint8.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "uint16":
						// Make sure we've actually got a uint16.
						v, ok := item.(uint16)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected uint16.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "uint32":
						// Make sure we've actually got a uint32.
						v, ok := item.(uint32)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected uint32.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "uint64":
						// Make sure we've actually got a uint64.
						v, ok := item.(uint64)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected uint64.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "float32":
						// Make sure we've actually got a float32.
						v, ok := item.(JsonFloat32)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected JsonFloat32.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v.F); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					case "float64":
						// Make sure we've actually got a float64.
						v, ok := item.(JsonFloat64)
						if !ok {
							return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected JsonFloat64.")
						}
						// Then write out the value.
						if err := binary.Write(buf, binary.LittleEndian, v.F); err != nil {
							return errors.Wrap(err, "Field: "+field.Name)
						}
					default:
						// Something went wrong.
						return errors.New("we haven't implemented this primitive yet")
					}
				}

			} else {
				// Else it's not a builtin.

				// Confirm the message we're holding is actually the correct type.
				msg, ok := item.(dynamicMessageLike)
				if !ok {
					return errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(item).Name() + ", expected Message.")
				}
				if msg.GetDynamicType().spec.ShortName != field.Type {
					return errors.New("Field: " + field.Name + ": Found msg " + msg.GetDynamicType().spec.ShortName + ", expected " + field.Type + ".")
				}
				// Otherwise, we just recursively serialise it.
				if err = msg.Serialize(buf); err != nil {
					return errors.Wrap(err, "Field: "+field.Name)
				}
			}
		}
	}

	// All done.
	return err
}

// Deserialize parses a byte stream into a DynamicMessage, thus reconstructing the fields of a received ROS message; required for ros.Message.
func (m *DynamicMessage) Deserialize(buf *bytes.Reader) error {

	if m.dynamicType.spec == nil {
		return errors.New("dynamic message type spec is nil")
	}
	if m.dynamicType.nested == nil {
		return errors.New("dynamic message type nested is nil")
	}

	// To give more sane results in the event of a decoding issue, we decode into a copy of the data field.
	var err error = nil
	tmpData := make(map[string]interface{})
	m.data = nil

	d := LEByteDecoder{}

	// Iterate over each of the fields in the message.
	for _, field := range m.dynamicType.spec.Fields {

		if field.IsArray {
			// It's an array.

			// The array may be a fixed length, or it may be dynamic.
			size := field.ArrayLen
			if field.ArrayLen < 0 {
				// The array is dynamic, so it starts with a declaration of the number of array elements.
				var usize uint32
				if usize, err = d.DecodeUint32(buf); err != nil {
					return errors.Wrap(err, "Field: "+field.Name)
				}
				size = int(usize)
			}
			// Create an array of the target type.
			switch field.GoType {
			case "bool":
				tmpData[field.Name], err = d.DecodeBoolArray(buf, size)
			case "int8":
				tmpData[field.Name], err = d.DecodeInt8Array(buf, size)
			case "int16":
				tmpData[field.Name], err = d.DecodeInt16Array(buf, size)
			case "int32":
				tmpData[field.Name], err = d.DecodeInt32Array(buf, size)
			case "int64":
				tmpData[field.Name], err = d.DecodeInt64Array(buf, size)
			case "uint8":
				tmpData[field.Name], err = d.DecodeUint8Array(buf, size)
			case "uint16":
				tmpData[field.Name], err = d.DecodeUint16Array(buf, size)
			case "uint32":
				tmpData[field.Name], err = d.DecodeUint32Array(buf, size)
			case "uint64":
				tmpData[field.Name], err = d.DecodeUint64Array(buf, size)
			case "float32":
				tmpData[field.Name], err = d.DecodeFloat32Array(buf, size)
			case "float64":
				tmpData[field.Name], err = d.DecodeFloat64Array(buf, size)
			case "string":
				tmpData[field.Name], err = d.DecodeStringArray(buf, size)
			case "ros.Time":
				tmpData[field.Name], err = d.DecodeTimeArray(buf, size)
			case "ros.Duration":
				tmpData[field.Name], err = d.DecodeDurationArray(buf, size)
			default:
				if field.IsBuiltin {
					// Something went wrong.
					return errors.New("we haven't implemented this primitive yet")
				}

				// In this case, it will probably be because the go_type is describing another ROS message type
				if msgType, ok := m.dynamicType.nested[field.Name]; ok {
					tmpData[field.Name], err = d.DecodeMessageArray(buf, size, msgType)
				} else {
					err = errors.New("nested message type not known")
				}
			}
			if err != nil {
				return errors.Wrap(err, "Field: "+field.Name)
			}

		} else { // Else it's a scalar.

			if field.IsBuiltin {
				// It's a regular primitive element.
				switch field.GoType {
				case "bool":
					tmpData[field.Name], err = d.DecodeBool(buf)
				case "int8":
					tmpData[field.Name], err = d.DecodeInt8(buf)
				case "int16":
					tmpData[field.Name], err = d.DecodeInt16(buf)
				case "int32":
					tmpData[field.Name], err = d.DecodeInt32(buf)
				case "int64":
					tmpData[field.Name], err = d.DecodeInt64(buf)
				case "uint8":
					tmpData[field.Name], err = d.DecodeUint8(buf)
				case "uint16":
					tmpData[field.Name], err = d.DecodeUint16(buf)
				case "uint32":
					tmpData[field.Name], err = d.DecodeUint32(buf)
				case "uint64":
					tmpData[field.Name], err = d.DecodeUint64(buf)
				case "float32":
					tmpData[field.Name], err = d.DecodeFloat32(buf)
				case "float64":
					tmpData[field.Name], err = d.DecodeFloat64(buf)
				case "string":
					tmpData[field.Name], err = d.DecodeString(buf)
				case "ros.Time":
					tmpData[field.Name], err = d.DecodeTime(buf)
				case "ros.Duration":
					tmpData[field.Name], err = d.DecodeDuration(buf)
				default:
					// Something went wrong.
					return errors.New("we haven't implemented this primitive yet")
				}

				if err != nil {
					return errors.Wrap(err, "Field: "+field.Name)
				}

			} else {
				// Else it's not a built-in
				if msgType, ok := m.dynamicType.nested[field.Name]; ok {
					tmpData[field.Name], err = d.DecodeMessage(buf, msgType)
					if err != nil {
						return errors.Wrap(err, "Field: "+field.Name)
					}
				} else {
					return errors.Wrap(errors.New("nested message type not known"), "Field: "+field.Name)
				}
			}
		}
	}

	// All done.
	m.data = tmpData
	return err
}

// String returns a string which represents the encapsulared DynamicMessage data.
func (m *DynamicMessage) String() string {
	// Just print out the data!
	return fmt.Sprint(m.dynamicType.Name(), "::", m.data)
}

// DEFINE PRIVATE STATIC FUNCTIONS.

// DEFINE PRIVATE RECEIVER FUNCTIONS.

// zeroValueData creates the zeroValue (default) data map for a new DynamicMessage.
func (t *DynamicMessageType) zeroValueData() (map[string]interface{}, error) {
	//Create map
	d := make(map[string]interface{})

	// Function guards to prevent this call from panicking.
	if t.spec == nil {
		return d, errors.New("dynamic message type spec is nil")
	}
	if t.nested == nil {
		return d, errors.New("dynamic message type nested is nil")
	}

	var err error

	//Range fields in the dynamic message type.
	for _, field := range t.spec.Fields {
		if field.IsArray {
			// If the array length is static set the size, for dynamic arrays, use 0.
			var size int = 0
			if field.ArrayLen > 0 {
				size = field.ArrayLen
			}

			switch field.GoType {
			case "bool":
				d[field.Name] = make([]bool, size)
			case "int8":
				d[field.Name] = make([]int8, size)
			case "int16":
				d[field.Name] = make([]int16, size)
			case "int32":
				d[field.Name] = make([]int32, size)
			case "int64":
				d[field.Name] = make([]int64, size)
			case "uint8":
				d[field.Name] = make([]uint8, size)
			case "uint16":
				d[field.Name] = make([]uint16, size)
			case "uint32":
				d[field.Name] = make([]uint32, size)
			case "uint64":
				d[field.Name] = make([]uint64, size)
			case "float32":
				d[field.Name] = make([]JsonFloat32, size)
			case "float64":
				d[field.Name] = make([]JsonFloat64, size)
			case "string":
				d[field.Name] = make([]string, size)
			case "ros.Time":
				d[field.Name] = make([]Time, size)
			case "ros.Duration":
				d[field.Name] = make([]Duration, size)
			default:
				if field.IsBuiltin {
					// Something went wrong.
					return d, errors.New("we haven't implemented this primitive yet")
				}

				// In this case, the go_type is describing another ROS message, so we nest a DynamicMessage.
				messages := make([]Message, size)
				if msgType, ok := t.nested[field.Name]; ok {
					// Fill out our new messages.
					for i := 0; i < size; i++ {
						messages[i] = msgType.NewMessage()
					}
				} else {
					return d, errors.Wrap(errors.New("nested message type not known"), "Type: "+field.Type)
				}
				d[field.Name] = messages
			}
		} else { // Not an array.
			if field.IsBuiltin {
				// If it is a built in type.
				switch field.GoType {
				case "string":
					d[field.Name] = ""
				case "bool":
					d[field.Name] = bool(false)
				case "int8":
					d[field.Name] = int8(0)
				case "int16":
					d[field.Name] = int16(0)
				case "int32":
					d[field.Name] = int32(0)
				case "int64":
					d[field.Name] = int64(0)
				case "uint8":
					d[field.Name] = uint8(0)
				case "uint16":
					d[field.Name] = uint16(0)
				case "uint32":
					d[field.Name] = uint32(0)
				case "uint64":
					d[field.Name] = uint64(0)
				case "float32":
					d[field.Name] = JsonFloat32{F: float32(0.0)}
				case "float64":
					d[field.Name] = JsonFloat64{F: float64(0.0)}
				case "ros.Time":
					d[field.Name] = Time{}
				case "ros.Duration":
					d[field.Name] = Duration{}
				default:
					return d, errors.Wrap(err, "builtin field "+field.GoType+" not found")
				}
				//Else its a ros message type
			} else {
				//Create new dynamic message type nested
				if msgType, ok := t.nested[field.Name]; ok {
					d[field.Name] = msgType.NewMessage()
				} else {
					return d, errors.Wrap(errors.New("nested message type not known"), "Field: "+field.Name)
				}
			}
		}
	}
	return d, err
}

// padArray pads the provided array to the specified length using the default value for the array type.
func padArray(array interface{}, field libgengo.Field, actualSize, requiredSize uint32) (interface{}, error) {
	switch field.GoType {
	case "bool":
		// Make sure we've actually got a bool.
		v, ok := array.([]bool)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []bool.")
		}
		padding := make([]bool, requiredSize-actualSize)
		for i := range padding {
			padding[i] = false
		}
		v = append(v, padding...)
		return v, nil
	case "int8":
		// Make sure we've actually got an int8.
		v, ok := array.([]int8)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []int8.")
		}
		padding := make([]int8, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "int16":
		// Make sure we've actually got an int16.
		v, ok := array.([]int16)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []int16.")
		}
		padding := make([]int16, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "int32":
		// Make sure we've actually got an int32.
		v, ok := array.([]int32)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []int32.")
		}
		padding := make([]int32, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "int64":
		// Make sure we've actually got an int64.
		v, ok := array.([]int64)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []int64.")
		}
		padding := make([]int64, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "uint8":
		// Make sure we've actually got a uint8.
		v, ok := array.([]uint8)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []uint8.")
		}
		padding := make([]uint8, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "uint16":
		// Make sure we've actually got a uint16.
		v, ok := array.([]uint16)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []uint16.")
		}
		padding := make([]uint16, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "uint32":
		// Make sure we've actually got a uint32.
		v, ok := array.([]uint32)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []uint32.")
		}
		padding := make([]uint32, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "uint64":
		// Make sure we've actually got a uint64.
		v, ok := array.([]uint64)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []uint64.")
		}
		padding := make([]uint64, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "float32":
		// Make sure we've actually got a float32.
		v, ok := array.([]float32)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []float32.")
		}
		padding := make([]float32, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	case "float64":
		// Make sure we've actually got a float64.
		v, ok := array.([]float64)
		if !ok {
			return nil, errors.New("Field: " + field.Name + ": Found " + reflect.TypeOf(array).Name() + ", expected []float64.")
		}
		padding := make([]float64, requiredSize-actualSize)
		for i := range padding {
			padding[i] = 0
		}
		v = append(v, padding...)
		return v, nil
	default:
		// Something went wrong.
		return nil, errors.New("we haven't implemented this primitive yet")
	}
}

// ALL DONE.
