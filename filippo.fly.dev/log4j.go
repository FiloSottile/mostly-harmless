package main

import (
	"log"
	"net"

	"github.com/nmcclain/ldap"
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
	return s.ListenAndServe(":3389")
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
