package main

import (
	"fmt"
	"net"
	"sync"
)

var wg sync.WaitGroup

func main() {
	host := "scanme.nmap.org"
	for p := 1; p <= 1024; p++ {
		wg.Add(1)
		go scan(host, p)
	}
	wg.Wait()
}

func scan(host string, port int) {
	defer wg.Done()
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	conn.Close()
	fmt.Printf("%d open\n", port)
}
