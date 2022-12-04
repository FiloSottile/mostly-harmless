package dockerdns

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
)

const Name = "docker"

func init() { plugin.Register(Name, setup) }

func setup(c *caddy.Controller) error {
	u, err := url.Parse(c.Key)
	if err != nil {
		return err
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Docker{Next: next, Domain: "." + u.Hostname()}
	})

	return nil
}

type Docker struct {
	Next   plugin.Handler
	Domain string
}

func (d Docker) Name() string { return Name }

func (d Docker) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	if state.QClass() != dns.ClassINET || !strings.HasSuffix(state.Name(), d.Domain) {
		return plugin.NextOrFailure(d.Name(), d.Next, ctx, w, r)
	}
	name := "/" + strings.TrimSuffix(state.Name(), d.Domain)

	m := new(dns.Msg)
	defer w.WriteMsg(m)
	m.SetReply(r)
	m.Rcode = dns.RcodeNameError

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		m.Rcode = dns.RcodeServerFailure
		return 0, err
	}
	defer cli.Close()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		m.Rcode = dns.RcodeServerFailure
		return 0, err
	}
	for _, container := range containers {
		if slices.Contains(container.Names, name) {
			m.Rcode = dns.RcodeSuccess
			for _, network := range container.NetworkSettings.Networks {
				ip := net.ParseIP(network.IPAddress)
				if ip != nil && state.QType() == dns.TypeA {
					m.Answer = append(m.Answer, &dns.A{A: ip, Hdr: dns.RR_Header{
						Name: state.QName(), Rrtype: dns.TypeA, Class: dns.ClassINET}})
				}
			}
		}
	}

	return 0, nil
}
