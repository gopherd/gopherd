// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: proto/protobuf/gatepb/gatec.proto

package gatepb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type KickoutReason int32

const (
	KickoutReason_ReasonServiceClosed      KickoutReason = 0
	KickoutReason_ReasonUserLogout         KickoutReason = 1
	KickoutReason_ReasonLoginAnotherDevice KickoutReason = 2
	KickoutReason_ReasonFrozen             KickoutReason = 3
	KickoutReason_ReasonOverflow           KickoutReason = 4
)

// Enum value maps for KickoutReason.
var (
	KickoutReason_name = map[int32]string{
		0: "ReasonServiceClosed",
		1: "ReasonUserLogout",
		2: "ReasonLoginAnotherDevice",
		3: "ReasonFrozen",
		4: "ReasonOverflow",
	}
	KickoutReason_value = map[string]int32{
		"ReasonServiceClosed":      0,
		"ReasonUserLogout":         1,
		"ReasonLoginAnotherDevice": 2,
		"ReasonFrozen":             3,
		"ReasonOverflow":           4,
	}
)

func (x KickoutReason) Enum() *KickoutReason {
	p := new(KickoutReason)
	*p = x
	return p
}

func (x KickoutReason) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (KickoutReason) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_protobuf_gatepb_gatec_proto_enumTypes[0].Descriptor()
}

func (KickoutReason) Type() protoreflect.EnumType {
	return &file_proto_protobuf_gatepb_gatec_proto_enumTypes[0]
}

func (x KickoutReason) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use KickoutReason.Descriptor instead.
func (KickoutReason) EnumDescriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{0}
}

// @Type(130)
type Error struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Errno       int32  `protobuf:"varint,1,opt,name=errno,proto3" json:"errno,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
}

func (x *Error) Reset() {
	*x = Error{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Error) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Error) ProtoMessage() {}

func (x *Error) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Error.ProtoReflect.Descriptor instead.
func (*Error) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{0}
}

func (x *Error) GetErrno() int32 {
	if x != nil {
		return x.Errno
	}
	return 0
}

func (x *Error) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

// @Type(131)
type Ping struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Content string `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *Ping) Reset() {
	*x = Ping{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ping) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ping) ProtoMessage() {}

func (x *Ping) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ping.ProtoReflect.Descriptor instead.
func (*Ping) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{1}
}

func (x *Ping) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

// @Type(132)
type Pong struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Content string `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *Pong) Reset() {
	*x = Pong{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Pong) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Pong) ProtoMessage() {}

func (x *Pong) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Pong.ProtoReflect.Descriptor instead.
func (*Pong) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{2}
}

func (x *Pong) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

// @Type(133)
type LoginRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *LoginRequest) Reset() {
	*x = LoginRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LoginRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginRequest) ProtoMessage() {}

func (x *LoginRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.
func (*LoginRequest) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{3}
}

func (x *LoginRequest) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

// @Type(134)
type LogoutRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *LogoutRequest) Reset() {
	*x = LogoutRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogoutRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutRequest) ProtoMessage() {}

func (x *LogoutRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutRequest.ProtoReflect.Descriptor instead.
func (*LogoutRequest) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{4}
}

// @Type(135)
type LogoutResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Reason KickoutReason `protobuf:"varint,1,opt,name=reason,proto3,enum=gatepb.KickoutReason" json:"reason,omitempty"`
}

func (x *LogoutResponse) Reset() {
	*x = LogoutResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogoutResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutResponse) ProtoMessage() {}

func (x *LogoutResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_protobuf_gatepb_gatec_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutResponse.ProtoReflect.Descriptor instead.
func (*LogoutResponse) Descriptor() ([]byte, []int) {
	return file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP(), []int{5}
}

func (x *LogoutResponse) GetReason() KickoutReason {
	if x != nil {
		return x.Reason
	}
	return KickoutReason_ReasonServiceClosed
}

var File_proto_protobuf_gatepb_gatec_proto protoreflect.FileDescriptor

var file_proto_protobuf_gatepb_gatec_proto_rawDesc = []byte{
	0x0a, 0x21, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x67, 0x61, 0x74, 0x65, 0x70, 0x62, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x63, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x06, 0x67, 0x61, 0x74, 0x65, 0x70, 0x62, 0x22, 0x3f, 0x0a, 0x05, 0x45,
	0x72, 0x72, 0x6f, 0x72, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6e, 0x6f, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6e, 0x6f, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65,
	0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x20, 0x0a, 0x04,
	0x50, 0x69, 0x6e, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x20,
	0x0a, 0x04, 0x50, 0x6f, 0x6e, 0x67, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x22, 0x24, 0x0a, 0x0c, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x0f, 0x0a, 0x0d, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x3f, 0x0a, 0x0e, 0x4c, 0x6f, 0x67, 0x6f, 0x75,
	0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x2d, 0x0a, 0x06, 0x72, 0x65, 0x61,
	0x73, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x15, 0x2e, 0x67, 0x61, 0x74, 0x65,
	0x70, 0x62, 0x2e, 0x4b, 0x69, 0x63, 0x6b, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x61, 0x73, 0x6f, 0x6e,
	0x52, 0x06, 0x72, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x2a, 0x82, 0x01, 0x0a, 0x0d, 0x4b, 0x69, 0x63,
	0x6b, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x12, 0x17, 0x0a, 0x13, 0x52, 0x65,
	0x61, 0x73, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x43, 0x6c, 0x6f, 0x73, 0x65,
	0x64, 0x10, 0x00, 0x12, 0x14, 0x0a, 0x10, 0x52, 0x65, 0x61, 0x73, 0x6f, 0x6e, 0x55, 0x73, 0x65,
	0x72, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x10, 0x01, 0x12, 0x1c, 0x0a, 0x18, 0x52, 0x65, 0x61,
	0x73, 0x6f, 0x6e, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x41, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x44,
	0x65, 0x76, 0x69, 0x63, 0x65, 0x10, 0x02, 0x12, 0x10, 0x0a, 0x0c, 0x52, 0x65, 0x61, 0x73, 0x6f,
	0x6e, 0x46, 0x72, 0x6f, 0x7a, 0x65, 0x6e, 0x10, 0x03, 0x12, 0x12, 0x0a, 0x0e, 0x52, 0x65, 0x61,
	0x73, 0x6f, 0x6e, 0x4f, 0x76, 0x65, 0x72, 0x66, 0x6c, 0x6f, 0x77, 0x10, 0x04, 0x42, 0x1f, 0x48,
	0x03, 0x5a, 0x0c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x70, 0x62, 0xaa,
	0x02, 0x0c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x67, 0x61, 0x74, 0x65, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_protobuf_gatepb_gatec_proto_rawDescOnce sync.Once
	file_proto_protobuf_gatepb_gatec_proto_rawDescData = file_proto_protobuf_gatepb_gatec_proto_rawDesc
)

func file_proto_protobuf_gatepb_gatec_proto_rawDescGZIP() []byte {
	file_proto_protobuf_gatepb_gatec_proto_rawDescOnce.Do(func() {
		file_proto_protobuf_gatepb_gatec_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_protobuf_gatepb_gatec_proto_rawDescData)
	})
	return file_proto_protobuf_gatepb_gatec_proto_rawDescData
}

var file_proto_protobuf_gatepb_gatec_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_protobuf_gatepb_gatec_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_proto_protobuf_gatepb_gatec_proto_goTypes = []interface{}{
	(KickoutReason)(0),     // 0: gatepb.KickoutReason
	(*Error)(nil),          // 1: gatepb.Error
	(*Ping)(nil),           // 2: gatepb.Ping
	(*Pong)(nil),           // 3: gatepb.Pong
	(*LoginRequest)(nil),   // 4: gatepb.LoginRequest
	(*LogoutRequest)(nil),  // 5: gatepb.LogoutRequest
	(*LogoutResponse)(nil), // 6: gatepb.LogoutResponse
}
var file_proto_protobuf_gatepb_gatec_proto_depIdxs = []int32{
	0, // 0: gatepb.LogoutResponse.reason:type_name -> gatepb.KickoutReason
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_proto_protobuf_gatepb_gatec_proto_init() }
func file_proto_protobuf_gatepb_gatec_proto_init() {
	if File_proto_protobuf_gatepb_gatec_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Error); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ping); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Pong); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LoginRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogoutRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_protobuf_gatepb_gatec_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogoutResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_protobuf_gatepb_gatec_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_protobuf_gatepb_gatec_proto_goTypes,
		DependencyIndexes: file_proto_protobuf_gatepb_gatec_proto_depIdxs,
		EnumInfos:         file_proto_protobuf_gatepb_gatec_proto_enumTypes,
		MessageInfos:      file_proto_protobuf_gatepb_gatec_proto_msgTypes,
	}.Build()
	File_proto_protobuf_gatepb_gatec_proto = out.File
	file_proto_protobuf_gatepb_gatec_proto_rawDesc = nil
	file_proto_protobuf_gatepb_gatec_proto_goTypes = nil
	file_proto_protobuf_gatepb_gatec_proto_depIdxs = nil
}
