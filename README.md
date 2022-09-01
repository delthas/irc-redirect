# irc-redirect

A simple IRC load balancer / request router that redirects clients to upstream IRC servers.

The redirection is done at the application level (with RPL_BOUNCE), distributing load using a round-robin algorithm.

## Usage

```shell
irc-redirect -upstream upstream1.example.com -upstream upstream2.example.com
```

See `irc-redirect -help` for details.

## License

MIT
