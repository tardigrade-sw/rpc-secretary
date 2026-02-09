package main

import "github.com/tardigrade-sw/rpc-secretary/server"

func main() {
	server := server.NewDocsServer("/Users/kf/Documents/Maturita/DECK/gitea-manager/proto", "")

	server.Serve("localhost:8099")
}
