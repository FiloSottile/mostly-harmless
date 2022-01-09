package main

import (
	"log"
	"net"

	"github.com/nmcclain/ldap"
	"github.com/pires/go-proxyproto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var log4jHits = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "ldap_searches_total",
	Help: "LDAP search requests, partitioned by path.",
}, []string{"base"})

func startLDAPServer() error {
	s := ldap.NewServer()
	handler := ldapHandler{}
	s.BindFunc("", handler)
	s.SearchFunc("", handler)
	ln, err := net.Listen("tcp", ":3389")
	if err != nil {
		return err
	}
	return s.Serve(&proxyproto.Listener{Listener: ln})
}

type ldapHandler struct{}

func (h ldapHandler) Bind(bindDN, bindSimplePw string, conn net.Conn) (ldap.LDAPResultCode, error) {
	return ldap.LDAPResultSuccess, nil
}

func (h ldapHandler) Search(boundDN string, searchReq ldap.SearchRequest, conn net.Conn) (ldap.ServerSearchResult, error) {
	log4jHits.WithLabelValues(searchReq.BaseDN).Inc()
	log.Printf("LDAP search %q from %v", searchReq.BaseDN, conn.RemoteAddr())
	return ldap.ServerSearchResult{nil, nil, nil, ldap.LDAPResultUnwillingToPerform}, nil
}
