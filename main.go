package main

import (
	"fmt"
	"log"
	"regexp"

	"github.com/miekg/dns"
)

func main() {
	appConfigs, err := InitConfig()
	if err != nil {
		log.Fatalf("Failed to load configs: %s", err)
	}

	dnsCache := InitCache(appConfigs.CacheExpiration)
	domains, _ := appConfigs.DNSConfigs["domains"]
	servers, _ := appConfigs.DNSConfigs["servers"]
	cnames, _ := appConfigs.DNSConfigs["cname_overrides"]
	defaultServer := make([]string, len(appConfigs.DNSConfigs["defaultDns"].([]interface{})))

	for i, v := range appConfigs.DNSConfigs["defaultDns"].([]interface{}) {
		defaultServer[i] = fmt.Sprint(v)
	}

	dnsProxy := DNSProxy{
		Cache:         &dnsCache,
		domains:       domains.(map[string]interface{}),
		servers:       servers.(map[string]interface{}),
		cnames:        cnames.(map[string]interface{}),
		defaultServer: defaultServer,
	}

	logger := NewLogger(appConfigs.LogLevel)
	host := ""
	if appConfigs.UseOutbound {
		ip, err := GetOutboundIP()
		if err != nil {
			logger.Fatalf("Failed to get outbound address: %s\n ", err.Error())
		}
		host = ip.String() + ":53"
	} else {
		host = appConfigs.DNSConfigs["host"].(string)
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		switch r.Opcode {
		case dns.OpcodeQuery:
			m, err := dnsProxy.request_interceptor(r, logger)
			if err != nil {
				logger.Errorf("Failed lookup for %s with error: %s\n", r, err.Error())
				m.SetReply(r)
				w.WriteMsg(m)
				return
			}
			if len(m.Answer) > 0 {
				pattern := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
				ipAddress := pattern.FindAllString(m.Answer[0].String(), -1)

				if len(ipAddress) > 0 {
					logger.Infof("Lookup for %s with ip %s\n", m.Answer[0].Header().Name, ipAddress[0])
				} else {
					logger.Infof("Lookup for %s with response %s\n", m.Answer[0].Header().Name, m.Answer[0])
				}
			}
			m.SetReply(r)
			w.WriteMsg(m)
		}
	})

	server := &dns.Server{Addr: host, Net: "udp"}
	logger.Infof("Starting at %s\n", host)
	err = server.ListenAndServe()
	if err != nil {
		logger.Errorf("Failed to start server: %s\n ", err.Error())
	}
}
