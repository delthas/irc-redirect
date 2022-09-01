package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"gopkg.in/sorcix/irc.v2"
	"net"
	"net/url"
	"os"
	"strings"
)

type stringSliceFlag []string

func (v *stringSliceFlag) String() string {
	return fmt.Sprint([]string(*v))
}

func (v *stringSliceFlag) Set(s string) error {
	*v = append(*v, s)
	return nil
}

func parseEndpoint(endpoint string) (host string, port string, tls bool, err error) {
	var u *url.URL
	if u, err = url.Parse(endpoint); err == nil && u.Scheme != "" {
		switch u.Scheme {
		case "irc", "ircs":
			tls = true
		case "irc+insecure":
		default:
			err = fmt.Errorf("invalid endpoint: %v", endpoint)
			return
		}
		host = u.Hostname()
		port = strings.TrimPrefix(u.Port(), "+")
		if port == "" {
			port = "6697"
		}
		return
	}
	if host, port, err = net.SplitHostPort(endpoint); err == nil {
		if strings.HasPrefix(port, "+") {
			tls = true
			port = strings.TrimPrefix(port, "+")
		}
		return
	}
	err = nil
	host = endpoint
	port = "6697"
	tls = true
	return
}

type redirect struct {
	host string
	port string
}

func main() {
	var upstreams []string
	flag.Var((*stringSliceFlag)(&upstreams), "upstream", "")
	listen := flag.String("listen", ":+6697", "listen addr:port")
	flag.Parse()

	if len(upstreams) == 0 {
		fmt.Fprintf(os.Stderr, "-upstream is required\n")
		os.Exit(1)
		return
	}
	redirects := make([]redirect, len(upstreams))
	for i, u := range upstreams {
		host, port, tls, err := parseEndpoint(u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed parsing -upstream %q: %v\n", u, err)
			os.Exit(1)
			return
		}
		if tls {
			port = "+" + port
		}
		redirects[i] = redirect{
			host: host,
			port: port,
		}
	}

	lHost, lPort, lTLS, err := parseEndpoint(*listen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed parsing -listen address: %v\n", err)
		os.Exit(1)
		return
	}
	var l net.Listener
	if lTLS {
		l, err = tls.Listen("tcp", net.JoinHostPort(lHost, lPort), nil)
	} else {
		l, err = net.Listen("tcp", net.JoinHostPort(lHost, lPort))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed listening on %q: %v\n", *listen, err)
		os.Exit(1)
		return
	}

	i := 0
	for {
		cc, err := l.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "accept: %v\n", err)
			continue
		}
		c := irc.NewConn(cc)
		redirect := redirects[i]
		i = (i + 1) % len(redirects)
		go func() {
			reason := fmt.Sprintf("Please connect to server %v:%v", redirect.host, redirect.port)
			c.Encode(&irc.Message{
				Command: "010", // RPL_BOUNCE
				Params:  []string{"*", redirect.host, redirect.port, reason},
			})
			c.Encode(&irc.Message{
				Command: "ERROR",
				Params:  []string{reason},
			})
			c.Close()
		}()
	}
}
