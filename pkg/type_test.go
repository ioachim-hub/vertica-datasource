package main

import (
	"testing"
	"unicode/utf8"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"google.golang.org/protobuf/proto"
)

func TestPrepareRowForFrameEncodesBinaryBeforeProtobufSerialization(t *testing.T) {
	raw := []byte{0xff, 0xfe, 0x00, 0x80}

	row := prepareRowForFrame([]string{"VARBINARY"}, []interface{}{&raw})

	value, ok := row[0].(*string)
	if !ok {
		t.Fatalf("prepared binary value has type %T, want *string", row[0])
	}
	if got, want := *value, "fffe0080"; got != want {
		t.Fatalf("prepared binary value = %q, want %q", got, want)
	}

	frame := data.NewFrameOfFieldTypes("A", 0, data.FieldTypeNullableString)
	frame.AppendRow(row...)
	response := &backend.QueryDataResponse{
		Responses: backend.Responses{
			"A": {Frames: data.Frames{frame}},
		},
	}

	protobufResponse, err := backend.ToProto().QueryDataResponse(backend.DataFrameFormat_ARROW, response)
	if err != nil {
		t.Fatalf("convert response to protobuf: %v", err)
	}
	if _, err := proto.Marshal(protobufResponse); err != nil {
		t.Fatalf("marshal protobuf response: %v", err)
	}
}

func TestPrepareRowForFrameRepairsInvalidTextUTF8(t *testing.T) {
	invalid := string([]byte{0xff, 'a'})

	row := prepareRowForFrame([]string{"VARCHAR"}, []interface{}{&invalid})

	value, ok := row[0].(*string)
	if !ok {
		t.Fatalf("prepared text value has type %T, want *string", row[0])
	}
	if !utf8.ValidString(*value) {
		t.Fatalf("prepared text value %q is not valid UTF-8", *value)
	}
	if got, want := *value, "\uFFFDa"; got != want {
		t.Fatalf("prepared text value = %q, want %q", got, want)
	}
}
