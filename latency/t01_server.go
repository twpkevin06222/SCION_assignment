//https://github.com/netsec-ethz/scion-homeworks/blob/master/latency/timestamp_server.go

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/scionproto/scion/go/lib/sciond"
	"github.com/scionproto/scion/go/lib/snet"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func printUsage() {
	fmt.Println("\ntimestamp_server -s server_SCION_addr")
	fmt.Println("\tThe SCION address is specified as ISD-AS,[IP Address]:Port")
}

func main() {
	var (
		server_addr string

		err    error
		server *snet.Addr

		udpConnection *snet.Conn
	)

	// Fetch arguments from command line
	flag.StringVar(&server_addr, "s", "", "Server SCION Address")
	flag.Parse()

	// Create the SCION UDP socket
	if len(server_addr) > 0 {
		server, err = snet.AddrFromString(server_addr)
		check(err)
	} else {
		printUsage()
		check(fmt.Errorf("Error, server address needs to be specified with -s"))
	}

	dispatcherAddr := "/run/shm/dispatcher/default.sock"
	snet.Init(server.IA, sciond.GetDefaultSCIONDPath(nil), dispatcherAddr)

	udpConnection, err = snet.ListenSCION("udp4", server)
	check(err)

	receivePacketBuffer := make([]byte, 2500)
	for {
		n, client_addr, err := udpConnection.ReadFrom(receivePacketBuffer)
		check(err)

		// Packet received, send back response to same client
		m := binary.PutVarint(receivePacketBuffer[n:], time.Now().UnixNano())
		_, err = udpConnection.WriteTo(receivePacketBuffer[:n+m], client_addr)
		check(err)
		fmt.Println("Connected to client", client_addr)
	}
}
