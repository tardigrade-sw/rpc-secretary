package tools

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tardigrade-sw/rpc-secretary/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ParseToDocumentation converts a FileDescriptorSet into a full Documentation structure.
func ParseToDocumentation(fds *descriptorpb.FileDescriptorSet) types.Documentation {
	doc := types.Documentation{
		Messages: make(map[string]types.Message),
		Enums:    make(map[string]types.Enum),
	}

	for _, file := range fds.GetFile() {
		fDoc := ParseFile(file)
		doc.Services = append(doc.Services, fDoc.Services...)
		maps.Copy(doc.Messages, fDoc.Messages)
		maps.Copy(doc.Enums, fDoc.Enums)
	}

	return doc
}

// ParseFile converts a single FileDescriptorProto into a Documentation subset.
func ParseFile(file *descriptorpb.FileDescriptorProto) types.Documentation {
	doc := types.Documentation{
		Messages: make(map[string]types.Message),
		Enums:    make(map[string]types.Enum),
	}

	comments := buildCommentMap(file)
	pkg := file.GetPackage()

	// 1. Top-level Enums
	for i, enm := range file.GetEnumType() {
		parseEnum(enm, pkg, []int32{5, int32(i)}, comments, doc.Enums)
	}

	// 2. Top-level Messages
	for i, msg := range file.GetMessageType() {
		parseMessage(msg, pkg, []int32{4, int32(i)}, comments, doc.Messages, doc.Enums)
	}

	// 3. Services
	for i, svc := range file.GetService() {
		path := []int32{6, int32(i)}
		s := types.Service{
			Name:        svc.GetName(),
			Description: comments[pathKey(path)],
		}

		for j, mth := range svc.GetMethod() {
			mPath := append(path, 2, int32(j))
			s.Methods = append(s.Methods, types.Method{
				Name:            mth.GetName(),
				InputType:       types.Primitive(strings.TrimPrefix(mth.GetInputType(), ".")),
				OutputType:      types.Primitive(strings.TrimPrefix(mth.GetOutputType(), ".")),
				ClientStreaming: mth.GetClientStreaming(),
				ServerStreaming: mth.GetServerStreaming(),
				Description:     comments[pathKey(mPath)],
			})
		}
		doc.Services = append(doc.Services, s)
	}

	return doc
}

// Internal recursive parser for Messages
func parseMessage(msg *descriptorpb.DescriptorProto, prefix string, path []int32, comments map[string]string, msgTarget map[string]types.Message, enumTarget map[string]types.Enum) {
	fullPath := prefix + "." + msg.GetName()
	m := types.Message{
		Name:        msg.GetName(),
		Description: comments[pathKey(path)],
	}

	for i, field := range msg.GetField() {
		fPath := append(path, 2, int32(i))
		m.Properties = append(m.Properties, types.Property{
			Name:        field.GetName(),
			Description: comments[pathKey(fPath)],
			IsRepeated:  field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
			Type:        types.Primitive(strings.TrimPrefix(field.GetTypeName(), ".")),
		})
	}
	msgTarget[fullPath] = m

	for i, nested := range msg.GetNestedType() {
		parseMessage(nested, fullPath, append(path, 3, int32(i)), comments, msgTarget, enumTarget)
	}
	for i, nested := range msg.GetEnumType() {
		parseEnum(nested, fullPath, append(path, 4, int32(i)), comments, enumTarget)
	}
}

// Internal parser for Enums
func parseEnum(enm *descriptorpb.EnumDescriptorProto, prefix string, path []int32, comments map[string]string, target map[string]types.Enum) {
	e := types.Enum{
		Name:        enm.GetName(),
		Description: comments[pathKey(path)],
		Values:      make(map[string]int32),
	}
	for _, val := range enm.GetValue() {
		e.Values[val.GetName()] = val.GetNumber()
	}
	target[prefix+"."+enm.GetName()] = e
}

// buildCommentMap extracts documentation from SourceCodeInfo
func buildCommentMap(file *descriptorpb.FileDescriptorProto) map[string]string {
	m := make(map[string]string)
	if file.SourceCodeInfo == nil {
		return m
	}
	for _, loc := range file.SourceCodeInfo.Location {
		if loc.LeadingComments != nil {
			m[pathKey(loc.Path)] = strings.TrimSpace(*loc.LeadingComments)
		}
	}
	return m
}

func pathKey(path []int32) string {
	var s []string
	for _, p := range path {
		s = append(s, fmt.Sprint(p))
	}
	return strings.Join(s, ",")
}

// ParseDescriptorSet reads a FileDescriptorSet bundle from disk
func ParseDescriptorSet(filePath string) (*descriptorpb.FileDescriptorSet, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(b, fds); err != nil {
		return nil, err
	}
	return fds, nil
}

// CompileProtos compiles .proto files into a temporary binary bundle for parsing
func CompileProtos(protoPath string, files []string, outputPath string) error {
	args := []string{"--include_source_info", "--include_imports", "--descriptor_set_out=" + outputPath, "-I" + protoPath}
	for _, f := range files {
		rel, err := filepath.Rel(protoPath, f)
		if err == nil {
			args = append(args, rel)
		} else {
			args = append(args, f)
		}
	}
	cmd := exec.Command("protoc", args...)
	cmd.Dir = protoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("protoc failed: %w\nOutput: %s", err, string(out))
	}
	return nil
}
