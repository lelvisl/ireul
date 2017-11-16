package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/garyburd/redigo/redis"
	"github.com/miekg/dns"
)

type inventory struct {
	IP       string   `json:"ip"`
	Site     string   `json:"site"`
	HostType string   `json:"hostType"`
	Apps     []string `json:"apps"`
	Active   bool     `json:"active"`
}

var conn redis.Conn

func main() {

	var err error
	conn, err = redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	server := &dns.Server{Addr: ":53", Net: "udp"}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()
	dns.HandleFunc(".", handleRequest)

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping\n", s)
}
func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	fmt.Println("handleRequest:inbound message:")
	fmt.Printf("%+v", r)
	for _, q := range r.Question {
		name := q.Name
		fmt.Println(name)
		str, err := redis.String(conn.Do("GET", name))
		if err != nil {
			fmt.Println(name, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}
		var host inventory
		err = json.Unmarshal([]byte(str), &host)
		if err != nil {
			fmt.Println("can't unmarshal", str, err)
			m.SetReply(r)
			fmt.Println(m.Answer)
			w.WriteMsg(m)
			return
		}
		answer := new(dns.A)
		answer.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA,
			Class: dns.ClassINET, Ttl: 3600}
		answer.A = net.ParseIP(host.IP) //.Preference = 10
		m.Answer = append(m.Answer, answer)

	}
	m.SetReply(r)

	fmt.Println(m.Answer)
	w.WriteMsg(m)
}
