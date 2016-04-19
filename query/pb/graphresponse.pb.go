// Code generated by protoc-gen-go.
// source: graphresponse.proto
// DO NOT EDIT!

/*
Package pb is a generated protocol buffer package.

It is generated from these files:
	graphresponse.proto

It has these top-level messages:
	UidList
	Result
	GraphResponse
*/
package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type UidList struct {
	Uids []uint64 `protobuf:"varint,1,rep,name=uids" json:"uids,omitempty"`
}

func (m *UidList) Reset()                    { *m = UidList{} }
func (m *UidList) String() string            { return proto.CompactTextString(m) }
func (*UidList) ProtoMessage()               {}
func (*UidList) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Result struct {
	Values    [][]byte   `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
	Uidmatrix []*UidList `protobuf:"bytes,2,rep,name=uidmatrix" json:"uidmatrix,omitempty"`
}

func (m *Result) Reset()                    { *m = Result{} }
func (m *Result) String() string            { return proto.CompactTextString(m) }
func (*Result) ProtoMessage()               {}
func (*Result) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Result) GetUidmatrix() []*UidList {
	if m != nil {
		return m.Uidmatrix
	}
	return nil
}

type GraphResponse struct {
	Attribute string           `protobuf:"bytes,1,opt,name=attribute" json:"attribute,omitempty"`
	Result    *Result          `protobuf:"bytes,2,opt,name=result" json:"result,omitempty"`
	Children  []*GraphResponse `protobuf:"bytes,3,rep,name=children" json:"children,omitempty"`
}

func (m *GraphResponse) Reset()                    { *m = GraphResponse{} }
func (m *GraphResponse) String() string            { return proto.CompactTextString(m) }
func (*GraphResponse) ProtoMessage()               {}
func (*GraphResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *GraphResponse) GetResult() *Result {
	if m != nil {
		return m.Result
	}
	return nil
}

func (m *GraphResponse) GetChildren() []*GraphResponse {
	if m != nil {
		return m.Children
	}
	return nil
}

func init() {
	proto.RegisterType((*UidList)(nil), "pb.UidList")
	proto.RegisterType((*Result)(nil), "pb.Result")
	proto.RegisterType((*GraphResponse)(nil), "pb.GraphResponse")
}

var fileDescriptor0 = []byte{
	// 209 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x54, 0x8f, 0xbf, 0x4a, 0xc5, 0x30,
	0x14, 0xc6, 0xc9, 0xbd, 0x25, 0xda, 0x53, 0x1d, 0x3c, 0x82, 0x74, 0x50, 0x90, 0x4c, 0x75, 0xb0,
	0x83, 0x3e, 0x84, 0x83, 0x4e, 0x07, 0x7c, 0x80, 0xc4, 0x06, 0x1b, 0xa8, 0x6d, 0xc8, 0x1f, 0x71,
	0xf4, 0xd1, 0x4d, 0xd3, 0x60, 0xb9, 0x5b, 0x92, 0xdf, 0xc7, 0xef, 0xfb, 0x02, 0xd7, 0x9f, 0x4e,
	0xda, 0xd1, 0x69, 0x6f, 0x97, 0xd9, 0xeb, 0xde, 0xba, 0x25, 0x2c, 0x78, 0xb0, 0x4a, 0xdc, 0xc1,
	0xd9, 0xbb, 0x19, 0xde, 0x8c, 0x0f, 0x88, 0x50, 0x45, 0x33, 0xf8, 0x96, 0xdd, 0x1f, 0xbb, 0x8a,
	0xf2, 0x59, 0xbc, 0x02, 0x27, 0xed, 0xe3, 0x14, 0xf0, 0x06, 0xf8, 0xb7, 0x9c, 0xa2, 0xde, 0xf8,
	0x05, 0x95, 0x1b, 0x3e, 0x40, 0x9d, 0x92, 0x5f, 0x32, 0x38, 0xf3, 0xd3, 0x1e, 0x12, 0x6a, 0x9e,
	0x9a, 0xde, 0xaa, 0xbe, 0x58, 0x69, 0xa7, 0xe2, 0x97, 0xc1, 0xe5, 0xcb, 0xba, 0x83, 0xca, 0x0e,
	0xbc, 0x85, 0x5a, 0x86, 0xc4, 0x54, 0x0c, 0x3a, 0x79, 0x59, 0x57, 0xd3, 0xfe, 0x80, 0x02, 0xb8,
	0xcb, 0xe5, 0xc9, 0xcb, 0x92, 0x17, 0x56, 0xef, 0x36, 0x87, 0x0a, 0xc1, 0x47, 0x38, 0xff, 0x18,
	0xcd, 0x34, 0x38, 0x3d, 0xb7, 0xc7, 0xdc, 0x7e, 0xb5, 0xa6, 0x4e, 0x6a, 0xe8, 0x3f, 0xa2, 0x78,
	0xfe, 0xf9, 0xf3, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xc1, 0x1f, 0x18, 0x20, 0x10, 0x01, 0x00,
	0x00,
}
