package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	IncludePaths   []string // Additional include paths for protoc
	ReflectionAddr string
}

func NewDocsServer(protoPath, reflectionAddr string) *DocsServer {
	return &DocsServer{
		ProtoPath:      protoPath,
		ReflectionAddr: reflectionAddr,
	}
}

// AddIncludePath adds an additional search directory for .proto files.
func (s *DocsServer) AddIncludePath(path string) {
	s.IncludePaths = append(s.IncludePaths, path)
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

	if s.ProtoPath != "" {
		localDoc, err := s.parseLocal(s.ProtoPath)
		if err == nil {
			s.mergeDoc(&doc, localDoc)
		} else {
			log.Printf("%v", err)
		}
	}

	if s.ReflectionAddr != "" {
		reflectDoc, err := s.parseReflection(s.ReflectionAddr)
		if err == nil {
			s.mergeDoc(&doc, reflectDoc)
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
				log.Printf("Error parsing file %s: %v, or dir empty", file, err)
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

			// Start with the proto directory and any user-specified paths
			includes := append([]string{protoPath}, s.IncludePaths...)

			// Add common system include paths if they exist
			commonPaths := []string{"/usr/include", "/usr/local/include"}
			for _, p := range commonPaths {
				if info, err := os.Stat(p); err == nil && info.IsDir() {
					includes = append(includes, p)
				}
			}

			if err := tools.CompileProtos(includes, protoFiles, tempBundle); err == nil {
				block, err := s.parseFile2Service(tempBundle)
				if err == nil {
					s.mergeDoc(&doc, block)
				} else {
					log.Printf("Error parsing .proto file %v", err)
				}
			} else {
				log.Printf("Error parsing .proto file %v", err)
			}
		} else {
			log.Printf("Error parsing proto dir: (%v), or dir empty", err)
		}
	}

	return doc, nil
}

func (s *DocsServer) mergeDoc(base *rpcTypes.Documentation, extra rpcTypes.Documentation) {
	serviceIndices := make(map[string]int)
	for i, svc := range base.Services {
		serviceIndices[svc.Name] = i
	}

	for _, extraSvc := range extra.Services {
		if idx, exists := serviceIndices[extraSvc.Name]; exists {
			if base.Services[idx].Description == "" && extraSvc.Description != "" {
				base.Services[idx].Description = extraSvc.Description
			}
			for j := range base.Services[idx].Methods {
				if j < len(extraSvc.Methods) {
					if base.Services[idx].Methods[j].Description == "" && extraSvc.Methods[j].Description != "" {
						base.Services[idx].Methods[j].Description = extraSvc.Methods[j].Description
					}
				}
			}
		} else {
			base.Services = append(base.Services, extraSvc)
		}
	}

	for k, v := range extra.Messages {
		if existing, ok := base.Messages[k]; ok {
			if existing.Description == "" && v.Description != "" {
				base.Messages[k] = v
			}
		} else {
			base.Messages[k] = v
		}
	}

	for k, v := range extra.Enums {
		if existing, ok := base.Enums[k]; ok {
			if existing.Description == "" && v.Description != "" {
				base.Enums[k] = v
			}
		} else {
			base.Enums[k] = v
		}
	}
}

func (s *DocsServer) parseFile2Service(filePath string) (rpcTypes.Documentation, error) {
	descriptor, err := tools.ParseDescriptorSet(filePath)
	if err != nil {
		return rpcTypes.Documentation{}, err
	}

	return tools.ParseToDocumentation(descriptor), nil
}
