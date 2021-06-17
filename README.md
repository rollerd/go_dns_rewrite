# DNS Proxy w/ CNAME override
A simple DNS proxy/cname override written in go based on [github.com/miekg/dns](https://github.com/miekg/dns)

## How to use it

Run via Docker container with:

```
docker run -p 53:53/udp go-dns-intercept
```

To send DNS requests to the container, update the /etc/resolv.conf file on the host to point to `127.0.0.1`

When running, be sure to add `--restart always` to the docker run command.

It can be started as 
```
docker run --name=dns_intercept -p 53:53/udp --restart=unless-stopped --detach=true dns_intercept:latest --expiration 60
```

## Arguments

```
	-file		 config filename
	-log-level	 log level(info,error or discard)
	-expiration      cache expiration time in seconds
	-use-outbound	 use outbound address as host for server
	-config-json     configs as json 
```

## Config file format

You can either build the image with the updated config.json or update it inside a running container (depending on how you start the process)

```json
{
    "host": "192.168.1.4:53",
    "defaultDns": ["8.8.8.8:53", "8.8.4.4:53"],
    "servers": {
        "google.com" : "8.8.8.8:53"
    },
    "domains": {
        ".*.com" : "8.8.8.8"
    },
    "cname_overrides": {
        "test.com." : "example.com",
        "google.com." : "yahoo.com"
}
```
Note: the '.' root is required at the end of the cname_override key


Based on [go-dns-proxy](https://github.com/katakonst/go-dns-proxy)
