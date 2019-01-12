// reference https://github.com/perrig/scionlab/blob/master/sensorapp/sensorfetcher/sensorfetcher.go

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
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
	fmt.Println("timestamp_client -s client_SCION_addr -d server_SCION_addr")
	fmt.Println("The SCION address is specified as ISD-AS,[IP Address]:Port")
}

const (
	n     = 20 //number of iterations
	n_max = 40 //max number of iterations
)

func main() {
	var (
		client_addr string
		server_addr string

		err    error
		local  *snet.Addr
		remote *snet.Addr

		udpConnection *snet.Conn
	)

	// Fetch arguments from command line
	flag.StringVar(&client_addr, "s", "", "Client SCION Address")
	flag.StringVar(&server_addr, "d", "", "Server SCION Address")
	flag.Parse()

	// Create the SCION UDP socket
	if len(client_addr) > 0 {
		local, err = snet.AddrFromString(client_addr)
		check(err)
	} else {
		printUsage()
		check(fmt.Errorf("Error, client address needs to be specified with -s"))
	}
	if len(server_addr) > 0 {
		remote, err = snet.AddrFromString(server_addr)
		check(err)
	} else {
		printUsage()
		check(fmt.Errorf("Error, server address needs to be specified with -d"))
	}

	dispatcherAddr := "/run/shm/dispatcher/default.sock"
	snet.Init(local.IA, sciond.GetDefaultSCIONDPath(nil), dispatcherAddr)

	udpConnection, err = snet.DialSCION("udp4", local, remote)
	check(err)

	receivePacketBuffer := make([]byte, 2500)
	sendPacketBuffer := make([]byte, 16)

	seed := rand.NewSource(time.Now().UnixNano())

	// Here comes the iteration for calculating average
	//initialize for sum
	var total int64 = 0
	iters := 0
	num_tries := 0

	for iters < n && num_tries < n_max {
		num_tries += 1

		id := rand.New(seed).Uint64()
		n := binary.PutUvarint(sendPacketBuffer, id)
		sendPacketBuffer[n] = 0

		time_sent := time.Now()
		_, err = udpConnection.Write(sendPacketBuffer)
		check(err)

		_, _, err = udpConnection.ReadFrom(receivePacketBuffer)
		check(err)

		ret_id, n := binary.Uvarint(receivePacketBuffer) //decodes uint64
		if ret_id == id {
			time_received, _ := binary.Varint(receivePacketBuffer[n:]) //decodes int64
			diff := (time_received - time_sent.UnixNano())             //unit:nanoseconds
			total += diff
			iters += 1
		}
	}

	if iters != n {
		check(fmt.Errorf("Error, exceeded maximum number of attempts"))
	}

	var difference float64 = float64(total) / float64(iters)

	//fmt.Printf("\nSource: %s\nDestination: %s\n", client_addr, server_addr)
	fmt.Prinf("Client: %s\n", client_addr)
	fmt.Prinf("Server: %s\n", server_addr)
	fmt.Println("Output:\n")
	fmt.Printf(" RTT: %.3fs\n", difference/1e9) //convert from ns to seconds
	fmt.Printf(" Latency: %.3fs\n", difference/2e9)
}
