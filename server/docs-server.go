package server

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tardigrade-sw/rpc-secretary/tools"
	rpcTypes "github.com/tardigrade-sw/rpc-secretary/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DocsServer struct {
	ProtoPath      string
	ReflectionAddr string
}

func NewDocsServer(protoPath, reflectionAddr string) *DocsServer {
	return &DocsServer{
		ProtoPath:      protoPath,
		ReflectionAddr: reflectionAddr,
	}
}

// Serve starts an HTTP server that provides the gRPC API documentation as JSON.
func (s *DocsServer) Serve(addr string) error {
	http.HandleFunc("/docs", s.handleDocs)
	fmt.Printf("Documentation server listening on %s/docs\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *DocsServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	doc := rpcTypes.Documentation{
		Messages: make(map[string]rpcTypes.Message),
		Enums:    make(map[string]rpcTypes.Enum),
	}

	// 1. Try local files if path is provided
	if s.ProtoPath != "" {
		localDoc, err := s.parseLocal(s.ProtoPath)
		if err == nil {
			doc.Services = append(doc.Services, localDoc.Services...)
			for k, v := range localDoc.Messages {
				doc.Messages[k] = v
			}
			for k, v := range localDoc.Enums {
				doc.Enums[k] = v
			}
		}
	}

	// 2. Try reflection if address is provided
	if s.ReflectionAddr != "" {
		reflectDoc, err := s.parseReflection(s.ReflectionAddr)
		if err == nil {
			doc.Services = append(doc.Services, reflectDoc.Services...)
			maps.Copy(doc.Messages, reflectDoc.Messages)
			maps.Copy(doc.Enums, reflectDoc.Enums)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *DocsServer) parseReflection(addr string) (rpcTypes.Documentation, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return rpcTypes.Documentation{}, err
	}
	defer conn.Close()

	client := tools.NewReflectionClient(conn)
	return client.FetchDocumentation(context.Background())
}

func (s *DocsServer) parseLocal(protoPath string) (rpcTypes.Documentation, error) {
	isDir, err := tools.IsDir(protoPath)
	if err != nil {
		return rpcTypes.Documentation{}, err
	}

	if !isDir {
		return s.parseFile2Service(protoPath)
	}

	doc := rpcTypes.Documentation{
		Messages: make(map[string]rpcTypes.Message),
		Enums:    make(map[string]rpcTypes.Enum),
	}

	// 1. Try to find binary bundles first (.pb)
	files, err := tools.GetFilesByType(protoPath, ".pb")
	if err == nil && len(files) > 0 {
		for _, file := range files {
			block, err := s.parseFile2Service(file)
			if err != nil {
				continue
			}
			s.mergeDoc(&doc, block)
		}
	}

	// 2. If no binary bundles, compile .proto files on the fly
	if len(doc.Services) == 0 {
		protoFiles, err := tools.GetFilesByType(protoPath, ".proto")
		if err == nil && len(protoFiles) > 0 {
			tempBundle := filepath.Join(os.TempDir(), "rpc-secretary-bundle.pb")
			if err := tools.CompileProtos(protoPath, protoFiles, tempBundle); err == nil {
				block, err := s.parseFile2Service(tempBundle)
				if err == nil {
					s.mergeDoc(&doc, block)
				}
			}
		}
	}

	return doc, nil
}

func (s *DocsServer) mergeDoc(base *rpcTypes.Documentation, extra rpcTypes.Documentation) {
	base.Services = append(base.Services, extra.Services...)
	maps.Copy(base.Messages, extra.Messages)
	maps.Copy(base.Enums, extra.Enums)
}

func (s *DocsServer) parseFile2Service(filePath string) (rpcTypes.Documentation, error) {
	descriptor, err := tools.ParseDescriptorSet(filePath)
	if err != nil {
		return rpcTypes.Documentation{}, err
	}

	return tools.ParseToDocumentation(descriptor), nil
}
