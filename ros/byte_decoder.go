package ros

import "bytes"

// ByteDecoder provides an interface that dynamic message deserialization expects for decoding ROS messages.
type ByteDecoder interface {
	DecodeBoolArray(buf *bytes.Reader, size int) ([]bool, error)
	DecodeInt8Array(buf *bytes.Reader, size int) ([]int8, error)
	DecodeInt16Array(buf *bytes.Reader, size int) ([]int16, error)
	DecodeInt32Array(buf *bytes.Reader, size int) ([]int32, error)
	DecodeInt64Array(buf *bytes.Reader, size int) ([]int64, error)
	DecodeUint8Array(buf *bytes.Reader, size int) ([]uint8, error)
	DecodeUint16Array(buf *bytes.Reader, size int) ([]uint16, error)
	DecodeUint32Array(buf *bytes.Reader, size int) ([]uint32, error)
	DecodeUint64Array(buf *bytes.Reader, size int) ([]uint64, error)
	DecodeFloat32Array(buf *bytes.Reader, size int) ([]JsonFloat32, error)
	DecodeFloat64Array(buf *bytes.Reader, size int) ([]JsonFloat64, error)
	DecodeStringArray(buf *bytes.Reader, size int) ([]string, error)
	DecodeTimeArray(buf *bytes.Reader, size int) ([]Time, error)
	DecodeDurationArray(buf *bytes.Reader, size int) ([]Duration, error)
	DecodeMessageArray(buf *bytes.Reader, size int, msgType *DynamicMessageType) ([]Message, error)

	DecodeBool(buf *bytes.Reader) (bool, error)
	DecodeInt8(buf *bytes.Reader) (int8, error)
	DecodeInt16(buf *bytes.Reader) (int16, error)
	DecodeInt32(buf *bytes.Reader) (int32, error)
	DecodeInt64(buf *bytes.Reader) (int64, error)
	DecodeUint8(buf *bytes.Reader) (uint8, error)
	DecodeUint16(buf *bytes.Reader) (uint16, error)
	DecodeUint32(buf *bytes.Reader) (uint32, error)
	DecodeUint64(buf *bytes.Reader) (uint64, error)
	DecodeFloat32(buf *bytes.Reader) (JsonFloat32, error)
	DecodeFloat64(buf *bytes.Reader) (JsonFloat64, error)
	DecodeString(buf *bytes.Reader) (string, error)
	DecodeTime(buf *bytes.Reader) (Time, error)
	DecodeDuration(buf *bytes.Reader) (Duration, error)
	DecodeMessage(buf *bytes.Reader, msgType *DynamicMessageType) (Message, error)
}
