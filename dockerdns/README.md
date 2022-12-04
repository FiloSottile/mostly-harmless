dockerdns is a CoreDNS plugin that serves the IPs of Docker containers.

It can be built into a CoreDNS build with the code-generation strategy, or by
building a new main package.

main.go
```
package main

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"

	_ "filippo.io/mostly-harmless/dockerdns"
	_ "github.com/coredns/coredns/plugin/chaos"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/log"
)

func init() { dnsserver.Directives = append(dnsserver.Directives, "docker") }
func main() { coremain.Run() }
```

Corefile
```
docker.home.arpa:53 {
    log
    errors
    docker
    chaos
}
```
