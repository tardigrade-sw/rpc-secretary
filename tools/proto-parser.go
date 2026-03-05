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

func ParseFile(file *descriptorpb.FileDescriptorProto) types.Documentation {
	doc := types.Documentation{
		Messages: make(map[string]types.Message),
		Enums:    make(map[string]types.Enum),
	}

	comments := buildCommentMap(file)
	pkg := file.GetPackage()

	getDesc := func(path []int32) string {
		if d, ok := comments[pathKey(path)]; ok && d != "" {
			return d
		}
		namePath := append(path, 1)
		return comments[pathKey(namePath)]
	}

	for i, enm := range file.GetEnumType() {
		parseEnum(enm, pkg, []int32{5, int32(i)}, getDesc, doc.Enums)
	}

	for i, msg := range file.GetMessageType() {
		parseMessage(msg, pkg, []int32{4, int32(i)}, getDesc, doc.Messages, doc.Enums)
	}

	for i, svc := range file.GetService() {
		path := []int32{6, int32(i)}
		s := types.Service{
			Name:        svc.GetName(),
			Description: getDesc(path),
		}

		for j, mth := range svc.GetMethod() {
			mPath := append(path, 2, int32(j))
			s.Methods = append(s.Methods, types.Method{
				Name:            mth.GetName(),
				InputType:       types.Primitive(strings.TrimPrefix(mth.GetInputType(), ".")),
				OutputType:      types.Primitive(strings.TrimPrefix(mth.GetOutputType(), ".")),
				ClientStreaming: mth.GetClientStreaming(),
				ServerStreaming: mth.GetServerStreaming(),
				Description:     getDesc(mPath),
			})
		}
		doc.Services = append(doc.Services, s)
	}

	return doc
}

func parseMessage(msg *descriptorpb.DescriptorProto, prefix string, path []int32, getDesc func([]int32) string, msgTarget map[string]types.Message, enumTarget map[string]types.Enum) {
	fullPath := prefix + "." + msg.GetName()
	m := types.Message{
		Name:        msg.GetName(),
		Description: getDesc(path),
	}

	for i, field := range msg.GetField() {
		fPath := append(path, 2, int32(i))
		m.Properties = append(m.Properties, types.Property{
			Name:        field.GetName(),
			Description: getDesc(fPath),
			IsRepeated:  field.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
			Type:        parseFieldType(field),
		})
	}
	msgTarget[fullPath] = m

	for i, nested := range msg.GetNestedType() {
		parseMessage(nested, fullPath, append(path, 3, int32(i)), getDesc, msgTarget, enumTarget)
	}
	for i, nested := range msg.GetEnumType() {
		parseEnum(nested, fullPath, append(path, 4, int32(i)), getDesc, enumTarget)
	}
}

func parseEnum(enm *descriptorpb.EnumDescriptorProto, prefix string, path []int32, getDesc func([]int32) string, target map[string]types.Enum) {
	e := types.Enum{
		Name:        enm.GetName(),
		Description: getDesc(path),
		Values:      make(map[string]int32),
	}
	for _, val := range enm.GetValue() {
		e.Values[val.GetName()] = val.GetNumber()
	}
	target[prefix+"."+enm.GetName()] = e
}

func buildCommentMap(file *descriptorpb.FileDescriptorProto) map[string]string {
	m := make(map[string]string)
	if file.SourceCodeInfo == nil {
		return m
	}
	for _, loc := range file.SourceCodeInfo.Location {
		var parts []string
		for _, dc := range loc.LeadingDetachedComments {
			parts = append(parts, strings.TrimSpace(dc))
		}
		if loc.LeadingComments != nil {
			parts = append(parts, strings.TrimSpace(*loc.LeadingComments))
		}
		if loc.TrailingComments != nil {
			parts = append(parts, strings.TrimSpace(*loc.TrailingComments))
		}

		if len(parts) > 0 {
			m[pathKey(loc.Path)] = strings.Join(parts, "\n\n")
		}
	}
	return m
}

func parseFieldType(field *descriptorpb.FieldDescriptorProto) types.Primitive {
	if field.GetTypeName() != "" {
		return types.Primitive(strings.TrimPrefix(field.GetTypeName(), "."))
	}
	return types.Primitive(strings.ToLower(strings.TrimPrefix(field.GetType().String(), "TYPE_")))
}

func pathKey(path []int32) string {
	var s []string
	for _, p := range path {
		s = append(s, fmt.Sprint(p))
	}
	return strings.Join(s, ",")
}

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

func CompileProtos(includePaths []string, files []string, outputPath string) error {
	if len(includePaths) == 0 {
		includePaths = []string{"."}
	}
	args := []string{"--include_source_info", "--include_imports", "--descriptor_set_out=" + outputPath}
	for _, path := range includePaths {
		args = append(args, "-I"+path)
	}

	for _, f := range files {
		rel, err := filepath.Rel(includePaths[0], f)
		if err == nil && !strings.HasPrefix(rel, "..") {
			args = append(args, rel)
		} else {
			args = append(args, f)
		}
	}
	cmd := exec.Command("protoc", args...)
	if len(includePaths) > 0 {
		cmd.Dir = includePaths[0]
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("protoc failed: %w\nOutput: %s", err, string(out))
	}
	return nil
}
