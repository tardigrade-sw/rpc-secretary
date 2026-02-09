package tools

import (
	"context"
	"maps"

	"github.com/tardigrade-sw/rpc-secretary/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ReflectionClient retrieves descriptors from a running gRPC server.
type ReflectionClient struct {
	client grpc_reflection_v1.ServerReflectionClient
}

// FetchAllServices retrieves all services and their definitions from the server.
func (rc *ReflectionClient) FetchDocumentation(ctx context.Context) (types.Documentation, error) {
	names, err := rc.ListServices(ctx)
	if err != nil {
		return types.Documentation{}, err
	}

	doc := types.Documentation{
		Messages: make(map[string]types.Message),
		Enums:    make(map[string]types.Enum),
	}
	seenFiles := make(map[string]bool)

	for _, name := range names {
		// Skip the reflection service itself
		if name == "grpc.reflection.v1.ServerReflection" || name == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}

		fdBytes, err := rc.GetFileDescriptorBySymbol(ctx, name)
		if err != nil {
			continue
		}

		for _, b := range fdBytes {
			fd := &descriptorpb.FileDescriptorProto{}
			if err := proto.Unmarshal(b, fd); err != nil {
				continue
			}

			if seenFiles[fd.GetName()] {
				continue
			}
			seenFiles[fd.GetName()] = true

			fDoc := ParseFile(fd)
			doc.Services = append(doc.Services, fDoc.Services...)
			maps.Copy(doc.Messages, fDoc.Messages)
			maps.Copy(doc.Enums, fDoc.Enums)
		}
	}

	return doc, nil
}

func NewReflectionClient(conn *grpc.ClientConn) *ReflectionClient {
	return &ReflectionClient{
		client: grpc_reflection_v1.NewServerReflectionClient(conn),
	}
}

// ListServices lists all services registered on the server.
func (rc *ReflectionClient) ListServices(ctx context.Context) ([]string, error) {
	stream, err := rc.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	ls := resp.GetListServicesResponse()
	var services []string
	for _, s := range ls.GetService() {
		services = append(services, s.GetName())
	}
	return services, nil
}

// GetFileDescriptorBySymbol retrieves the FileDescriptorProto for a given symbol.
func (rc *ReflectionClient) GetFileDescriptorBySymbol(ctx context.Context, symbol string) ([][]byte, error) {
	stream, err := rc.client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	return resp.GetFileDescriptorResponse().GetFileDescriptorProto(), nil
}
