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

func ToDS(zone string) {
	go func() {
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

			if dnskey.Flags&dns.SEP == 0 {
				log.Println("Ignoring ZSK:", x.RR)
				continue
			}

			ds1 := dnskey.ToDS(dns.SHA1)
			ds2 := dnskey.ToDS(dns.SHA256)

			js.Global.Get("document").Call("write", fmt.Sprintf("%s\n%s\n", ds1, ds2))
		}
	}()
}
