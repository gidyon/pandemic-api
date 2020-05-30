// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messaging.proto

package messaging

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	_ "google.golang.org/genproto/googleapis/longrunning"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// BroadCastMessageFilter is type filter for broadcast messages
type BroadCastMessageFilter int32

const (
	BroadCastMessageFilter_ALL       BroadCastMessageFilter = 0
	BroadCastMessageFilter_BY_COUNTY BroadCastMessageFilter = 1
	BroadCastMessageFilter_POSITIVES BroadCastMessageFilter = 2
	BroadCastMessageFilter_NEGATIVES BroadCastMessageFilter = 3
)

var BroadCastMessageFilter_name = map[int32]string{
	0: "ALL",
	1: "BY_COUNTY",
	2: "POSITIVES",
	3: "NEGATIVES",
}

var BroadCastMessageFilter_value = map[string]int32{
	"ALL":       0,
	"BY_COUNTY": 1,
	"POSITIVES": 2,
	"NEGATIVES": 3,
}

func (x BroadCastMessageFilter) String() string {
	return proto.EnumName(BroadCastMessageFilter_name, int32(x))
}

func (BroadCastMessageFilter) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{0}
}

// MessageType is category of a message
type MessageType int32

const (
	MessageType_ANY     MessageType = 0
	MessageType_ALERT   MessageType = 1
	MessageType_WARNING MessageType = 2
	MessageType_INFO    MessageType = 3
)

var MessageType_name = map[int32]string{
	0: "ANY",
	1: "ALERT",
	2: "WARNING",
	3: "INFO",
}

var MessageType_value = map[string]int32{
	"ANY":     0,
	"ALERT":   1,
	"WARNING": 2,
	"INFO":    3,
}

func (x MessageType) String() string {
	return proto.EnumName(MessageType_name, int32(x))
}

func (MessageType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{1}
}

// ContactData contains locational contacts infomation
type ContactData struct {
	Count                int32    `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
	UserPhone            string   `protobuf:"bytes,2,opt,name=user_phone,json=userPhone,proto3" json:"user_phone,omitempty"`
	FullName             string   `protobuf:"bytes,3,opt,name=full_name,json=fullName,proto3" json:"full_name,omitempty"`
	PatientPhone         string   `protobuf:"bytes,4,opt,name=patient_phone,json=patientPhone,proto3" json:"patient_phone,omitempty"`
	DeviceToken          string   `protobuf:"bytes,5,opt,name=device_token,json=deviceToken,proto3" json:"device_token,omitempty"`
	ContactTime          string   `protobuf:"bytes,6,opt,name=contact_time,json=contactTime,proto3" json:"contact_time,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ContactData) Reset()         { *m = ContactData{} }
func (m *ContactData) String() string { return proto.CompactTextString(m) }
func (*ContactData) ProtoMessage()    {}
func (*ContactData) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{0}
}

func (m *ContactData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ContactData.Unmarshal(m, b)
}
func (m *ContactData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ContactData.Marshal(b, m, deterministic)
}
func (m *ContactData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ContactData.Merge(m, src)
}
func (m *ContactData) XXX_Size() int {
	return xxx_messageInfo_ContactData.Size(m)
}
func (m *ContactData) XXX_DiscardUnknown() {
	xxx_messageInfo_ContactData.DiscardUnknown(m)
}

var xxx_messageInfo_ContactData proto.InternalMessageInfo

func (m *ContactData) GetCount() int32 {
	if m != nil {
		return m.Count
	}
	return 0
}

func (m *ContactData) GetUserPhone() string {
	if m != nil {
		return m.UserPhone
	}
	return ""
}

func (m *ContactData) GetFullName() string {
	if m != nil {
		return m.FullName
	}
	return ""
}

func (m *ContactData) GetPatientPhone() string {
	if m != nil {
		return m.PatientPhone
	}
	return ""
}

func (m *ContactData) GetDeviceToken() string {
	if m != nil {
		return m.DeviceToken
	}
	return ""
}

func (m *ContactData) GetContactTime() string {
	if m != nil {
		return m.ContactTime
	}
	return ""
}

// BroadCastMessageResponse is response after a message has been broadcasted containing the broadcast id
type BroadCastMessageResponse struct {
	BroadcastMessageId   string   `protobuf:"bytes,1,opt,name=broadcast_message_id,json=broadcastMessageId,proto3" json:"broadcast_message_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BroadCastMessageResponse) Reset()         { *m = BroadCastMessageResponse{} }
func (m *BroadCastMessageResponse) String() string { return proto.CompactTextString(m) }
func (*BroadCastMessageResponse) ProtoMessage()    {}
func (*BroadCastMessageResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{1}
}

func (m *BroadCastMessageResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BroadCastMessageResponse.Unmarshal(m, b)
}
func (m *BroadCastMessageResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BroadCastMessageResponse.Marshal(b, m, deterministic)
}
func (m *BroadCastMessageResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BroadCastMessageResponse.Merge(m, src)
}
func (m *BroadCastMessageResponse) XXX_Size() int {
	return xxx_messageInfo_BroadCastMessageResponse.Size(m)
}
func (m *BroadCastMessageResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_BroadCastMessageResponse.DiscardUnknown(m)
}

var xxx_messageInfo_BroadCastMessageResponse proto.InternalMessageInfo

func (m *BroadCastMessageResponse) GetBroadcastMessageId() string {
	if m != nil {
		return m.BroadcastMessageId
	}
	return ""
}

// BroadCastMessageRequest is request to broadcast message to users
type BroadCastMessageRequest struct {
	Title                string                   `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Message              string                   `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	Type                 MessageType              `protobuf:"varint,3,opt,name=type,proto3,enum=covitrace.MessageType" json:"type,omitempty"`
	Filters              []BroadCastMessageFilter `protobuf:"varint,4,rep,packed,name=filters,proto3,enum=covitrace.BroadCastMessageFilter" json:"filters,omitempty"`
	Topics               []string                 `protobuf:"bytes,5,rep,name=topics,proto3" json:"topics,omitempty"`
	Payload              map[string]string        `protobuf:"bytes,6,rep,name=payload,proto3" json:"payload,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *BroadCastMessageRequest) Reset()         { *m = BroadCastMessageRequest{} }
func (m *BroadCastMessageRequest) String() string { return proto.CompactTextString(m) }
func (*BroadCastMessageRequest) ProtoMessage()    {}
func (*BroadCastMessageRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{2}
}

func (m *BroadCastMessageRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BroadCastMessageRequest.Unmarshal(m, b)
}
func (m *BroadCastMessageRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BroadCastMessageRequest.Marshal(b, m, deterministic)
}
func (m *BroadCastMessageRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BroadCastMessageRequest.Merge(m, src)
}
func (m *BroadCastMessageRequest) XXX_Size() int {
	return xxx_messageInfo_BroadCastMessageRequest.Size(m)
}
func (m *BroadCastMessageRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BroadCastMessageRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BroadCastMessageRequest proto.InternalMessageInfo

func (m *BroadCastMessageRequest) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *BroadCastMessageRequest) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func (m *BroadCastMessageRequest) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_ANY
}

func (m *BroadCastMessageRequest) GetFilters() []BroadCastMessageFilter {
	if m != nil {
		return m.Filters
	}
	return nil
}

func (m *BroadCastMessageRequest) GetTopics() []string {
	if m != nil {
		return m.Topics
	}
	return nil
}

func (m *BroadCastMessageRequest) GetPayload() map[string]string {
	if m != nil {
		return m.Payload
	}
	return nil
}

// Message is a message payload
type Message struct {
	MessageId            string            `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	UserPhone            string            `protobuf:"bytes,2,opt,name=user_phone,json=userPhone,proto3" json:"user_phone,omitempty"`
	Title                string            `protobuf:"bytes,3,opt,name=title,proto3" json:"title,omitempty"`
	Notification         string            `protobuf:"bytes,4,opt,name=notification,proto3" json:"notification,omitempty"`
	Timestamp            int64             `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Sent                 bool              `protobuf:"varint,6,opt,name=sent,proto3" json:"sent,omitempty"`
	Seen                 bool              `protobuf:"varint,7,opt,name=seen,proto3" json:"seen,omitempty"`
	Type                 MessageType       `protobuf:"varint,8,opt,name=type,proto3,enum=covitrace.MessageType" json:"type,omitempty"`
	Data                 map[string]string `protobuf:"bytes,9,rep,name=data,proto3" json:"data,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{3}
}

func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

func (m *Message) GetUserPhone() string {
	if m != nil {
		return m.UserPhone
	}
	return ""
}

func (m *Message) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *Message) GetNotification() string {
	if m != nil {
		return m.Notification
	}
	return ""
}

func (m *Message) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Message) GetSent() bool {
	if m != nil {
		return m.Sent
	}
	return false
}

func (m *Message) GetSeen() bool {
	if m != nil {
		return m.Seen
	}
	return false
}

func (m *Message) GetType() MessageType {
	if m != nil {
		return m.Type
	}
	return MessageType_ANY
}

func (m *Message) GetData() map[string]string {
	if m != nil {
		return m.Data
	}
	return nil
}

// SendMessageResponse is response after sending message contains message id
type SendMessageResponse struct {
	MessageId            string   `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SendMessageResponse) Reset()         { *m = SendMessageResponse{} }
func (m *SendMessageResponse) String() string { return proto.CompactTextString(m) }
func (*SendMessageResponse) ProtoMessage()    {}
func (*SendMessageResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{4}
}

func (m *SendMessageResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SendMessageResponse.Unmarshal(m, b)
}
func (m *SendMessageResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SendMessageResponse.Marshal(b, m, deterministic)
}
func (m *SendMessageResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SendMessageResponse.Merge(m, src)
}
func (m *SendMessageResponse) XXX_Size() int {
	return xxx_messageInfo_SendMessageResponse.Size(m)
}
func (m *SendMessageResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SendMessageResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SendMessageResponse proto.InternalMessageInfo

func (m *SendMessageResponse) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

// ListMessagesRequest is request to get user messages
type ListMessagesRequest struct {
	PhoneNumber          string        `protobuf:"bytes,1,opt,name=phone_number,json=phoneNumber,proto3" json:"phone_number,omitempty"`
	PageToken            int32         `protobuf:"varint,2,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	PageSize             int32         `protobuf:"varint,3,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	FilterType           []MessageType `protobuf:"varint,4,rep,packed,name=filter_type,json=filterType,proto3,enum=covitrace.MessageType" json:"filter_type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *ListMessagesRequest) Reset()         { *m = ListMessagesRequest{} }
func (m *ListMessagesRequest) String() string { return proto.CompactTextString(m) }
func (*ListMessagesRequest) ProtoMessage()    {}
func (*ListMessagesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{5}
}

func (m *ListMessagesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListMessagesRequest.Unmarshal(m, b)
}
func (m *ListMessagesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListMessagesRequest.Marshal(b, m, deterministic)
}
func (m *ListMessagesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListMessagesRequest.Merge(m, src)
}
func (m *ListMessagesRequest) XXX_Size() int {
	return xxx_messageInfo_ListMessagesRequest.Size(m)
}
func (m *ListMessagesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListMessagesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListMessagesRequest proto.InternalMessageInfo

func (m *ListMessagesRequest) GetPhoneNumber() string {
	if m != nil {
		return m.PhoneNumber
	}
	return ""
}

func (m *ListMessagesRequest) GetPageToken() int32 {
	if m != nil {
		return m.PageToken
	}
	return 0
}

func (m *ListMessagesRequest) GetPageSize() int32 {
	if m != nil {
		return m.PageSize
	}
	return 0
}

func (m *ListMessagesRequest) GetFilterType() []MessageType {
	if m != nil {
		return m.FilterType
	}
	return nil
}

// Messages is a collection of user messages
type Messages struct {
	Messages             []*Message `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *Messages) Reset()         { *m = Messages{} }
func (m *Messages) String() string { return proto.CompactTextString(m) }
func (*Messages) ProtoMessage()    {}
func (*Messages) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{6}
}

func (m *Messages) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Messages.Unmarshal(m, b)
}
func (m *Messages) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Messages.Marshal(b, m, deterministic)
}
func (m *Messages) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Messages.Merge(m, src)
}
func (m *Messages) XXX_Size() int {
	return xxx_messageInfo_Messages.Size(m)
}
func (m *Messages) XXX_DiscardUnknown() {
	xxx_messageInfo_Messages.DiscardUnknown(m)
}

var xxx_messageInfo_Messages proto.InternalMessageInfo

func (m *Messages) GetMessages() []*Message {
	if m != nil {
		return m.Messages
	}
	return nil
}

// MessageRequest is request to fetch or perform action to messages owned by user
type MessageRequest struct {
	PhoneNumber          string   `protobuf:"bytes,1,opt,name=phone_number,json=phoneNumber,proto3" json:"phone_number,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MessageRequest) Reset()         { *m = MessageRequest{} }
func (m *MessageRequest) String() string { return proto.CompactTextString(m) }
func (*MessageRequest) ProtoMessage()    {}
func (*MessageRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{7}
}

func (m *MessageRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MessageRequest.Unmarshal(m, b)
}
func (m *MessageRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MessageRequest.Marshal(b, m, deterministic)
}
func (m *MessageRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MessageRequest.Merge(m, src)
}
func (m *MessageRequest) XXX_Size() int {
	return xxx_messageInfo_MessageRequest.Size(m)
}
func (m *MessageRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MessageRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MessageRequest proto.InternalMessageInfo

func (m *MessageRequest) GetPhoneNumber() string {
	if m != nil {
		return m.PhoneNumber
	}
	return ""
}

// NewMessagesCount contains the count of new messages
type NewMessagesCount struct {
	Count                int32    `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NewMessagesCount) Reset()         { *m = NewMessagesCount{} }
func (m *NewMessagesCount) String() string { return proto.CompactTextString(m) }
func (*NewMessagesCount) ProtoMessage()    {}
func (*NewMessagesCount) Descriptor() ([]byte, []int) {
	return fileDescriptor_42a1718997f046ec, []int{8}
}

func (m *NewMessagesCount) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NewMessagesCount.Unmarshal(m, b)
}
func (m *NewMessagesCount) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NewMessagesCount.Marshal(b, m, deterministic)
}
func (m *NewMessagesCount) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NewMessagesCount.Merge(m, src)
}
func (m *NewMessagesCount) XXX_Size() int {
	return xxx_messageInfo_NewMessagesCount.Size(m)
}
func (m *NewMessagesCount) XXX_DiscardUnknown() {
	xxx_messageInfo_NewMessagesCount.DiscardUnknown(m)
}

var xxx_messageInfo_NewMessagesCount proto.InternalMessageInfo

func (m *NewMessagesCount) GetCount() int32 {
	if m != nil {
		return m.Count
	}
	return 0
}

func init() {
	proto.RegisterEnum("covitrace.BroadCastMessageFilter", BroadCastMessageFilter_name, BroadCastMessageFilter_value)
	proto.RegisterEnum("covitrace.MessageType", MessageType_name, MessageType_value)
	proto.RegisterType((*ContactData)(nil), "covitrace.ContactData")
	proto.RegisterType((*BroadCastMessageResponse)(nil), "covitrace.BroadCastMessageResponse")
	proto.RegisterType((*BroadCastMessageRequest)(nil), "covitrace.BroadCastMessageRequest")
	proto.RegisterMapType((map[string]string)(nil), "covitrace.BroadCastMessageRequest.PayloadEntry")
	proto.RegisterType((*Message)(nil), "covitrace.Message")
	proto.RegisterMapType((map[string]string)(nil), "covitrace.Message.DataEntry")
	proto.RegisterType((*SendMessageResponse)(nil), "covitrace.SendMessageResponse")
	proto.RegisterType((*ListMessagesRequest)(nil), "covitrace.ListMessagesRequest")
	proto.RegisterType((*Messages)(nil), "covitrace.Messages")
	proto.RegisterType((*MessageRequest)(nil), "covitrace.MessageRequest")
	proto.RegisterType((*NewMessagesCount)(nil), "covitrace.NewMessagesCount")
}

func init() { proto.RegisterFile("messaging.proto", fileDescriptor_42a1718997f046ec) }

var fileDescriptor_42a1718997f046ec = []byte{
	// 1031 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x55, 0x4f, 0x6f, 0xe3, 0x44,
	0x14, 0x5f, 0xe7, 0x4f, 0x13, 0xbf, 0xa4, 0x25, 0x9a, 0xae, 0xba, 0x26, 0x29, 0x28, 0x75, 0x25,
	0x14, 0x0a, 0x9b, 0x6c, 0xd3, 0x95, 0x76, 0x09, 0xa7, 0xb4, 0x74, 0xab, 0x48, 0xdd, 0xb4, 0x72,
	0x03, 0xa8, 0x5c, 0xa2, 0x69, 0xfc, 0x1a, 0xac, 0x75, 0xc6, 0xc6, 0x33, 0x69, 0x95, 0x45, 0x5c,
	0xe0, 0xc4, 0x99, 0xcf, 0xc1, 0x17, 0xe1, 0x86, 0x38, 0x22, 0x71, 0xe2, 0x83, 0xa0, 0x99, 0xb1,
	0x83, 0x49, 0xb2, 0x6d, 0x39, 0xd9, 0xef, 0xcd, 0x6f, 0xde, 0x7b, 0xf3, 0x7b, 0xbf, 0x37, 0x03,
	0xef, 0x4d, 0x90, 0x73, 0x3a, 0xf6, 0xd8, 0xb8, 0x19, 0x46, 0x81, 0x08, 0x88, 0x39, 0x0a, 0x6e,
	0x3c, 0x11, 0xd1, 0x11, 0x56, 0x6b, 0xe3, 0x20, 0x18, 0xfb, 0xd8, 0x52, 0x0b, 0x57, 0xd3, 0xeb,
	0x16, 0x4e, 0x42, 0x31, 0xd3, 0xb8, 0xea, 0x6e, 0xbc, 0xe8, 0x07, 0x6c, 0x1c, 0x4d, 0x19, 0xf3,
	0xd8, 0xb8, 0x15, 0x84, 0x18, 0x51, 0xe1, 0x05, 0x8c, 0xc7, 0xa0, 0xed, 0x18, 0x44, 0x43, 0xaf,
	0x45, 0x19, 0x0b, 0xc4, 0x7f, 0x56, 0x3f, 0x55, 0x9f, 0xd1, 0xd3, 0x31, 0xb2, 0xa7, 0xfc, 0x96,
	0x8e, 0xc7, 0x18, 0xb5, 0x82, 0x50, 0x21, 0x96, 0xd1, 0xf6, 0x6f, 0x06, 0x94, 0x8e, 0x02, 0x26,
	0xe8, 0x48, 0x7c, 0x41, 0x05, 0x25, 0x8f, 0x21, 0x3f, 0x0a, 0xa6, 0x4c, 0x58, 0x46, 0xdd, 0x68,
	0xe4, 0x1d, 0x6d, 0x90, 0x0f, 0x00, 0xa6, 0x1c, 0xa3, 0x61, 0xf8, 0x6d, 0xc0, 0xd0, 0xca, 0xd4,
	0x8d, 0x86, 0xe9, 0x98, 0xd2, 0x73, 0x2e, 0x1d, 0xa4, 0x06, 0xe6, 0xf5, 0xd4, 0xf7, 0x87, 0x8c,
	0x4e, 0xd0, 0xca, 0xaa, 0xd5, 0xa2, 0x74, 0xf4, 0xe9, 0x04, 0xc9, 0x2e, 0xac, 0x87, 0x54, 0x78,
	0xc8, 0x44, 0xbc, 0x3d, 0xa7, 0x00, 0xe5, 0xd8, 0xa9, 0x23, 0xec, 0x40, 0xd9, 0xc5, 0x1b, 0x6f,
	0x84, 0x43, 0x11, 0xbc, 0x41, 0x66, 0xe5, 0x15, 0xa6, 0xa4, 0x7d, 0x03, 0xe9, 0x92, 0x90, 0x91,
	0x2e, 0x74, 0x28, 0xbc, 0x09, 0x5a, 0x6b, 0x1a, 0x12, 0xfb, 0x06, 0xde, 0x04, 0xed, 0x53, 0xb0,
	0x0e, 0xa3, 0x80, 0xba, 0x47, 0x94, 0x8b, 0xd7, 0xaa, 0x03, 0xe8, 0x20, 0x0f, 0x03, 0xc6, 0x91,
	0x3c, 0x83, 0xc7, 0x57, 0x72, 0x6d, 0x44, 0xb9, 0x18, 0xea, 0xf6, 0xe0, 0xd0, 0x73, 0xd5, 0x39,
	0x4d, 0x87, 0xcc, 0xd7, 0xe2, 0x7d, 0x3d, 0xd7, 0xfe, 0x3d, 0x03, 0x4f, 0x96, 0xc3, 0x7d, 0x37,
	0x45, 0x2e, 0x24, 0x4d, 0xc2, 0x13, 0x3e, 0xc6, 0xdb, 0xb5, 0x41, 0x2c, 0x28, 0xc4, 0x91, 0x63,
	0x8e, 0x12, 0x93, 0xec, 0x41, 0x4e, 0xcc, 0x42, 0x4d, 0xce, 0x46, 0x7b, 0xab, 0x39, 0x97, 0x43,
	0x33, 0x0e, 0x3c, 0x98, 0x85, 0xe8, 0x28, 0x0c, 0xf9, 0x1c, 0x0a, 0xd7, 0x9e, 0x2f, 0x30, 0xe2,
	0x56, 0xae, 0x9e, 0x6d, 0x6c, 0xb4, 0x77, 0x52, 0xf0, 0xc5, 0x82, 0x5e, 0x29, 0xa4, 0x93, 0xec,
	0x20, 0x5b, 0xb0, 0x26, 0x82, 0xd0, 0x1b, 0x71, 0x2b, 0x5f, 0xcf, 0x36, 0x4c, 0x27, 0xb6, 0x48,
	0x0f, 0x0a, 0x21, 0x9d, 0xf9, 0x01, 0x75, 0xad, 0xb5, 0x7a, 0xb6, 0x51, 0x6a, 0xb7, 0xee, 0x08,
	0x1a, 0x9f, 0xb2, 0x79, 0xae, 0x77, 0x1c, 0x33, 0x11, 0xcd, 0x9c, 0x64, 0x7f, 0xb5, 0x03, 0xe5,
	0xf4, 0x02, 0xa9, 0x40, 0xf6, 0x0d, 0xce, 0x62, 0x26, 0xe4, 0xaf, 0x64, 0xe7, 0x86, 0xfa, 0xd3,
	0x84, 0x05, 0x6d, 0x74, 0x32, 0x2f, 0x0d, 0xfb, 0xaf, 0x0c, 0x14, 0xe2, 0x24, 0x52, 0x54, 0x4b,
	0x7d, 0x30, 0x27, 0x09, 0xfd, 0xf7, 0x69, 0x6e, 0xde, 0x81, 0x6c, 0xba, 0x03, 0x36, 0x94, 0x59,
	0x20, 0xbc, 0x6b, 0x6f, 0xa4, 0x54, 0x9e, 0x68, 0x2d, 0xed, 0x23, 0xdb, 0x60, 0x4a, 0x01, 0x71,
	0x41, 0x27, 0xa1, 0x12, 0x5a, 0xd6, 0xf9, 0xd7, 0x41, 0x08, 0xe4, 0x38, 0x32, 0xa1, 0xe4, 0x55,
	0x74, 0xd4, 0xbf, 0xf6, 0x21, 0xb3, 0x0a, 0x89, 0x0f, 0xd9, 0xbc, 0xa3, 0xc5, 0x07, 0x74, 0xf4,
	0x19, 0xe4, 0x5c, 0x2a, 0xa8, 0x65, 0x2a, 0xe6, 0xb7, 0x97, 0xb1, 0x4d, 0x39, 0x7b, 0x9a, 0x66,
	0x85, 0xac, 0xbe, 0x00, 0x73, 0xee, 0xfa, 0x5f, 0x04, 0x3f, 0x87, 0xcd, 0x0b, 0x64, 0xee, 0xa2,
	0xfa, 0xef, 0xe6, 0xda, 0xfe, 0xd5, 0x80, 0xcd, 0x53, 0x6f, 0xde, 0x7f, 0x9e, 0xc8, 0x7c, 0x07,
	0xca, 0x8a, 0xfe, 0x21, 0x9b, 0x4e, 0xae, 0x30, 0x8a, 0x37, 0x96, 0x94, 0xaf, 0xaf, 0x5c, 0x32,
	0x72, 0x28, 0xc3, 0xea, 0xb9, 0xcd, 0xa8, 0x5b, 0xc3, 0x94, 0x1e, 0x3d, 0xb5, 0x35, 0x50, 0xc6,
	0x90, 0x7b, 0x6f, 0x75, 0xab, 0xf2, 0x4e, 0x51, 0x3a, 0x2e, 0xbc, 0xb7, 0x48, 0x5e, 0x40, 0x49,
	0xeb, 0x76, 0xa8, 0xa8, 0xd4, 0x6a, 0x7f, 0x17, 0x95, 0xa0, 0xa1, 0xf2, 0xdf, 0xee, 0x40, 0x31,
	0x29, 0x95, 0x34, 0xa1, 0x18, 0x1f, 0x84, 0x5b, 0x86, 0x22, 0x98, 0x2c, 0x47, 0x70, 0xe6, 0x18,
	0xfb, 0x00, 0x36, 0x16, 0x86, 0xf9, 0xfe, 0x53, 0xda, 0x0d, 0xa8, 0xf4, 0xf1, 0x36, 0xc9, 0x79,
	0xa4, 0x2e, 0xc5, 0x95, 0x57, 0xe5, 0x5e, 0x1f, 0xb6, 0x56, 0xcf, 0x28, 0x29, 0x40, 0xb6, 0x7b,
	0x7a, 0x5a, 0x79, 0x44, 0xd6, 0xc1, 0x3c, 0xbc, 0x1c, 0x1e, 0x9d, 0x7d, 0xd9, 0x1f, 0x5c, 0x56,
	0x0c, 0x69, 0x9e, 0x9f, 0x5d, 0xf4, 0x06, 0xbd, 0xaf, 0x8e, 0x2f, 0x2a, 0x19, 0x69, 0xf6, 0x8f,
	0x4f, 0xba, 0xda, 0xcc, 0xee, 0xbd, 0x84, 0x52, 0x8a, 0x05, 0x15, 0xa4, 0x7f, 0x59, 0x79, 0x44,
	0x4c, 0xc8, 0x77, 0x4f, 0x8f, 0x9d, 0x41, 0xc5, 0x20, 0x25, 0x28, 0x7c, 0xdd, 0x75, 0xfa, 0xbd,
	0xfe, 0x49, 0x25, 0x43, 0x8a, 0x90, 0xeb, 0xf5, 0x5f, 0x9d, 0x55, 0xb2, 0xed, 0x3f, 0xf3, 0x60,
	0xbe, 0x4e, 0xde, 0x21, 0x82, 0xb0, 0xde, 0xf5, 0x31, 0x12, 0xf1, 0x65, 0xcf, 0x49, 0x9a, 0xe7,
	0xd4, 0x0b, 0x50, 0xdd, 0x6a, 0xea, 0xe7, 0xa5, 0x99, 0x3c, 0x50, 0xcd, 0x63, 0xf9, 0x40, 0xd9,
	0xf6, 0x8f, 0x7f, 0xfc, 0xfd, 0x4b, 0x66, 0xdb, 0x7e, 0xa2, 0xde, 0x9d, 0x9b, 0xfd, 0xd6, 0xfc,
	0x8d, 0x6b, 0x51, 0x19, 0xb8, 0x63, 0xec, 0x35, 0x0c, 0xf2, 0x93, 0x01, 0x95, 0xc5, 0xf3, 0x13,
	0xfb, 0xfe, 0xbb, 0xa6, 0xba, 0x7b, 0x27, 0x46, 0xcb, 0xd8, 0xfe, 0x48, 0xd5, 0x50, 0xb7, 0x6b,
	0xcb, 0x35, 0xcc, 0x2f, 0xf0, 0x8e, 0xb1, 0x47, 0x5c, 0x28, 0xa5, 0xa6, 0x80, 0xac, 0x10, 0x44,
	0xf5, 0xc3, 0x94, 0x6f, 0xc5, 0xc4, 0xd8, 0x3b, 0x2a, 0x55, 0xcd, 0xde, 0x5a, 0x4e, 0xc5, 0x91,
	0xb9, 0x32, 0xcb, 0x0d, 0x94, 0xd3, 0x43, 0x43, 0xd2, 0x21, 0x57, 0x4c, 0x53, 0x75, 0x73, 0xb9,
	0x0c, 0x6e, 0xef, 0xab, 0x3c, 0x9f, 0x90, 0x8f, 0x97, 0xf3, 0x24, 0x92, 0x6d, 0x7d, 0x9f, 0x96,
	0xe7, 0x0f, 0xe4, 0x16, 0x0a, 0x0e, 0x52, 0xb7, 0xeb, 0xfb, 0xe4, 0xfd, 0x15, 0x52, 0x8f, 0xb3,
	0xbd, 0xab, 0x8f, 0x9f, 0xa9, 0x84, 0x07, 0xed, 0xfd, 0x07, 0x27, 0x6c, 0x45, 0x48, 0x5d, 0xea,
	0xfb, 0xe4, 0x67, 0x03, 0x36, 0x4f, 0x50, 0x2c, 0x4d, 0xc2, 0x1d, 0x55, 0xd4, 0x52, 0x4b, 0x8b,
	0xfb, 0xec, 0x8e, 0x2a, 0xe5, 0x39, 0x69, 0x3f, 0xbc, 0x14, 0x86, 0xb7, 0x6a, 0xce, 0x0e, 0x4b,
	0xdf, 0x98, 0x73, 0xf4, 0xd5, 0x9a, 0x3a, 0xe3, 0xc1, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0xab,
	0xd1, 0xe0, 0x2a, 0x78, 0x09, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MessagingClient is the client API for Messaging service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MessagingClient interface {
	// Alerts on possible contact points with a positive patient
	AlertContacts(ctx context.Context, opts ...grpc.CallOption) (Messaging_AlertContactsClient, error)
	// Broadcasts a message
	BroadCastMessage(ctx context.Context, in *BroadCastMessageRequest, opts ...grpc.CallOption) (*BroadCastMessageResponse, error)
	// Sends message to a single destination
	SendMessage(ctx context.Context, in *Message, opts ...grpc.CallOption) (*SendMessageResponse, error)
	// Retrieves user messages
	ListMessages(ctx context.Context, in *ListMessagesRequest, opts ...grpc.CallOption) (*Messages, error)
	// Marks all messages as read for a user
	ReadAll(ctx context.Context, in *MessageRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Fetches how many messages are new
	GetNewMessagesCount(ctx context.Context, in *MessageRequest, opts ...grpc.CallOption) (*NewMessagesCount, error)
}

type messagingClient struct {
	cc *grpc.ClientConn
}

func NewMessagingClient(cc *grpc.ClientConn) MessagingClient {
	return &messagingClient{cc}
}

func (c *messagingClient) AlertContacts(ctx context.Context, opts ...grpc.CallOption) (Messaging_AlertContactsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Messaging_serviceDesc.Streams[0], "/covitrace.Messaging/AlertContacts", opts...)
	if err != nil {
		return nil, err
	}
	x := &messagingAlertContactsClient{stream}
	return x, nil
}

type Messaging_AlertContactsClient interface {
	Send(*ContactData) error
	CloseAndRecv() (*empty.Empty, error)
	grpc.ClientStream
}

type messagingAlertContactsClient struct {
	grpc.ClientStream
}

func (x *messagingAlertContactsClient) Send(m *ContactData) error {
	return x.ClientStream.SendMsg(m)
}

func (x *messagingAlertContactsClient) CloseAndRecv() (*empty.Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(empty.Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *messagingClient) BroadCastMessage(ctx context.Context, in *BroadCastMessageRequest, opts ...grpc.CallOption) (*BroadCastMessageResponse, error) {
	out := new(BroadCastMessageResponse)
	err := c.cc.Invoke(ctx, "/covitrace.Messaging/BroadCastMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *messagingClient) SendMessage(ctx context.Context, in *Message, opts ...grpc.CallOption) (*SendMessageResponse, error) {
	out := new(SendMessageResponse)
	err := c.cc.Invoke(ctx, "/covitrace.Messaging/SendMessage", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *messagingClient) ListMessages(ctx context.Context, in *ListMessagesRequest, opts ...grpc.CallOption) (*Messages, error) {
	out := new(Messages)
	err := c.cc.Invoke(ctx, "/covitrace.Messaging/ListMessages", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *messagingClient) ReadAll(ctx context.Context, in *MessageRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/covitrace.Messaging/ReadAll", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *messagingClient) GetNewMessagesCount(ctx context.Context, in *MessageRequest, opts ...grpc.CallOption) (*NewMessagesCount, error) {
	out := new(NewMessagesCount)
	err := c.cc.Invoke(ctx, "/covitrace.Messaging/GetNewMessagesCount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MessagingServer is the server API for Messaging service.
type MessagingServer interface {
	// Alerts on possible contact points with a positive patient
	AlertContacts(Messaging_AlertContactsServer) error
	// Broadcasts a message
	BroadCastMessage(context.Context, *BroadCastMessageRequest) (*BroadCastMessageResponse, error)
	// Sends message to a single destination
	SendMessage(context.Context, *Message) (*SendMessageResponse, error)
	// Retrieves user messages
	ListMessages(context.Context, *ListMessagesRequest) (*Messages, error)
	// Marks all messages as read for a user
	ReadAll(context.Context, *MessageRequest) (*empty.Empty, error)
	// Fetches how many messages are new
	GetNewMessagesCount(context.Context, *MessageRequest) (*NewMessagesCount, error)
}

func RegisterMessagingServer(s *grpc.Server, srv MessagingServer) {
	s.RegisterService(&_Messaging_serviceDesc, srv)
}

func _Messaging_AlertContacts_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(MessagingServer).AlertContacts(&messagingAlertContactsServer{stream})
}

type Messaging_AlertContactsServer interface {
	SendAndClose(*empty.Empty) error
	Recv() (*ContactData, error)
	grpc.ServerStream
}

type messagingAlertContactsServer struct {
	grpc.ServerStream
}

func (x *messagingAlertContactsServer) SendAndClose(m *empty.Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *messagingAlertContactsServer) Recv() (*ContactData, error) {
	m := new(ContactData)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Messaging_BroadCastMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BroadCastMessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MessagingServer).BroadCastMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/covitrace.Messaging/BroadCastMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MessagingServer).BroadCastMessage(ctx, req.(*BroadCastMessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Messaging_SendMessage_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Message)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MessagingServer).SendMessage(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/covitrace.Messaging/SendMessage",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MessagingServer).SendMessage(ctx, req.(*Message))
	}
	return interceptor(ctx, in, info, handler)
}

func _Messaging_ListMessages_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListMessagesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MessagingServer).ListMessages(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/covitrace.Messaging/ListMessages",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MessagingServer).ListMessages(ctx, req.(*ListMessagesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Messaging_ReadAll_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MessagingServer).ReadAll(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/covitrace.Messaging/ReadAll",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MessagingServer).ReadAll(ctx, req.(*MessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Messaging_GetNewMessagesCount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MessageRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MessagingServer).GetNewMessagesCount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/covitrace.Messaging/GetNewMessagesCount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MessagingServer).GetNewMessagesCount(ctx, req.(*MessageRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Messaging_serviceDesc = grpc.ServiceDesc{
	ServiceName: "covitrace.Messaging",
	HandlerType: (*MessagingServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BroadCastMessage",
			Handler:    _Messaging_BroadCastMessage_Handler,
		},
		{
			MethodName: "SendMessage",
			Handler:    _Messaging_SendMessage_Handler,
		},
		{
			MethodName: "ListMessages",
			Handler:    _Messaging_ListMessages_Handler,
		},
		{
			MethodName: "ReadAll",
			Handler:    _Messaging_ReadAll_Handler,
		},
		{
			MethodName: "GetNewMessagesCount",
			Handler:    _Messaging_GetNewMessagesCount_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "AlertContacts",
			Handler:       _Messaging_AlertContacts_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "messaging.proto",
}
