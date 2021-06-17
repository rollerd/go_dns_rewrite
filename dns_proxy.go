package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/miekg/dns"
)

type DNSProxy struct {
	Cache         *Cache
	domains       map[string]interface{}
	servers       map[string]interface{}
	cnames        map[string]interface{}
	defaultServer []string
}

func (proxy *DNSProxy) request_interceptor(requestMsg *dns.Msg, logger *Log) (*dns.Msg, error) {
	orig_question_name := requestMsg.Question[0].Name

	if name, ok := proxy.cnames[dns.Fqdn(orig_question_name)]; ok {
		fmt.Printf("Modifying lookup from %s to %s\n", orig_question_name, name.(string))
		intercepted_msg := new(dns.Msg)
		intercepted_msg.SetQuestion(dns.Fqdn(name.(string)), dns.TypeA)
		resp, err := proxy.getResponse(intercepted_msg, logger)
		if err != nil {
			return resp, err
		}

		return resp, nil
	}

	m, err := proxy.getResponse(requestMsg, logger)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (proxy *DNSProxy) getResponse(requestMsg *dns.Msg, logger *Log) (*dns.Msg, error) {
	responseMsg := new(dns.Msg)
	if len(requestMsg.Question) > 0 {
		question := requestMsg.Question[0]

		dnsServer := proxy.getIPFromConfigs(question.Name, proxy.servers)

		var dnsServerArray []string
		if dnsServer != "" {
			dnsServerArray = []string{dnsServer}
		} else {
			dnsServerArray = proxy.defaultServer
		}

		var errorMsg error
		for i := 0; i < len(dnsServerArray); i++ {
			switch question.Qtype {
			case dns.TypeA:
				answer, err := proxy.processTypeA(dnsServerArray[i], &question, requestMsg)
				if err != nil {
					if strings.Contains(err.Error(), "i/o timeout") {
						logger.Warnf("Failed to lookup %s via %s: %s", requestMsg.Question[0].Name, dnsServerArray[i], err.Error())
						errorMsg = err
						continue
					} else {
						return responseMsg, err
					}
				}
				for _, v := range *answer {
					responseMsg.Answer = append(responseMsg.Answer, v)
				}
				return responseMsg, nil

			default:
				answer, err := proxy.processOtherTypes(dnsServerArray[i], &question, requestMsg)
				if err != nil {
					if strings.Contains(err.Error(), "i/o timeout") {
						logger.Warnf("Failed to lookup %s via %s: %s", requestMsg.Question[0].Name, dnsServerArray[i], err.Error())
						errorMsg = err
						continue
					} else {
						return responseMsg, err
					}
				}
				for _, v := range *answer {
					responseMsg.Answer = append(responseMsg.Answer, v)
				}
				return responseMsg, nil
			}
		}
		if errorMsg != nil {
			return responseMsg, errorMsg
		}
	}

	return responseMsg, nil
}

func (proxy *DNSProxy) processOtherTypes(dnsServer string, q *dns.Question, requestMsg *dns.Msg) (*[]dns.RR, error) {
	queryMsg := new(dns.Msg)
	requestMsg.CopyTo(queryMsg)
	queryMsg.Question = []dns.Question{*q}

	msg, err := lookup(dnsServer, queryMsg)
	if err != nil {
		return nil, err
	}

	if len(msg.Answer) > 0 {
		return &msg.Answer, nil
	}
	return nil, fmt.Errorf("not found")
}

func (proxy *DNSProxy) processTypeA(dnsServer string, q *dns.Question, requestMsg *dns.Msg) (*[]dns.RR, error) {
	ip := proxy.getIPFromConfigs(q.Name, proxy.domains)
	cacheMsg, found := proxy.Cache.Get(q.Name)

	if ip == "" && !found {
		queryMsg := new(dns.Msg)
		requestMsg.CopyTo(queryMsg)
		queryMsg.Question = []dns.Question{*q}

		msg, err := lookup(dnsServer, queryMsg)
		if err != nil {
			return nil, err
		}

		if len(msg.Answer) > 0 {
			proxy.Cache.Set(q.Name, &msg.Answer)
			return &msg.Answer, nil
		}

	} else if found {
		cache_hit := cacheMsg.(*[]dns.RR)
		return cache_hit, nil

	} else if ip != "" {

		answer, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
		if err != nil {
			return nil, err
		}
		var new_answer []dns.RR
		new_answer = append(new_answer, answer)
		return &new_answer, nil
	}
	return nil, fmt.Errorf("not found")
}

func (dnsProxy *DNSProxy) getIPFromConfigs(domain string, configs map[string]interface{}) string {

	for k, v := range configs {
		match, _ := regexp.MatchString(k+"\\.", domain)
		if match {
			return v.(string)
		}
	}
	return ""
}

func GetOutboundIP() (net.IP, error) {

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

func lookup(server string, m *dns.Msg) (*dns.Msg, error) {
	dnsClient := new(dns.Client)
	dnsClient.Net = "udp"
	response, _, err := dnsClient.Exchange(m, server)
	if err != nil {
		return nil, err
	}

	return response, nil
}
