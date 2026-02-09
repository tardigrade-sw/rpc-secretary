# RPC Secretary

RPC Secretary is a lightweight tool for gRPC API documentation discovery. It can parse local `.proto` files, compiled `DescriptorSet` bundles, or query a running gRPC server using reflection to generate a unified JSON schema of your API, including comments.

## Features

- **Hybrid Discovery**: Combine documentation from local source files and live running servers.
- **Comment Support**: Automatically extracts leading/trailing comments from `.proto` files using `SourceCodeInfo`.
- **On-the-fly Compilation**: Automatically compiles `.proto` files into temporary bundles if no binary descriptors are available.
- **Recursive Parsing**: Full support for nested Messages and Enums.
- **JSON API**: Serves the documentation through a simple HTTP endpoint.

## Requirements

- **Go**: 1.25.1 or later.
- **protoc**: (Optional) Required if you want to parse raw `.proto` files on the fly.

## Usage

### As a Documentation Server

You can use `DocsServer` to host a JSON endpoint for your API documentation.

```go
package main

import (
    "log"
    "github.com/tardigrade-sw/rpc-secretary/server"
)

func main() {
    // protoPath: Path to your .proto files or compiled .pb bundles
    // reflectionAddr: Address of a running gRPC server with reflection enabled
    ds := server.NewDocsServer("./protos", "localhost:50051")

    log.Fatal(ds.Serve(":8080")) // Documentation available at http://localhost:8080/docs
}
```

### Protocol Buffer Compilation

To ensure comments are included in your documentation, either allow the tool to compile `.proto` files for you, or manually generate descriptor sets with:

```bash
protoc --include_source_info --descriptor_set_out=bundle.pb your_service.proto
```

## JSON Schema Structure

The `/docs` endpoint returns a unified JSON object:

- `services`: A list of discovered gRPC services and their methods.
- `messages`: A map of full type names to their structure (properties, types, nested types).
- `enums`: A map of full enum names to their possible values.

## Architecture

- **`tools/`**: Contains the core logic for parsing binary descriptors and managing reflection streams.
- **`types/`**: Defines the unified model for documentation.
- **`server/`**: A helper that implements the `DocsServer` for serving the documentation over HTTP.

## License

MIT
