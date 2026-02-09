package types

// ProtoType is a union-like interface for types used in gRPC definitions.
// It can be a Message, an Enum, or a Primitive.
type ProtoType interface {
	isProtoType()
}

func (Message) isProtoType() {}
func (Enum) isProtoType()    {}

type Primitive string

func (Primitive) isProtoType() {}
