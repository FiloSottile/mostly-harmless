package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/cloudflare/dns"
	"github.com/gopherjs/gopherjs/js"
)

func main() {
	js.Global.Set("go", map[string]interface{}{
		"ToDS": ToDS,
	})
}

const Flags = dns.SEP | dns.ZONE

func ToDS() {
	go func() {
		// document.getElementById('dnskey').value
		zone := js.Global.Get("document").Call("getElementById", "dnskey").Get("value").String()

		var result string
		for x := range dns.ParseZone(strings.NewReader(zone), "", "") {
			if x.Error != nil {
				log.Println(x.Error)
				continue
			}

			dnskey, ok := x.RR.(*dns.DNSKEY)
			if !ok {
				log.Println("Not a DNSKEY:", x.RR)
				continue
			}

			if dnskey.Flags&Flags != Flags {
				log.Println("Ignoring non-KSK:", x.RR)
				continue
			}
			
			ds := dnskey.ToDS(dns.SHA256)
			result += fmt.Sprintf("%s\n", ds)
		}

		// document.getElementById('ds').innerHTML
		js.Global.Get("document").Call("getElementById", "ds").Set("innerHTML", result)
	}()
}
