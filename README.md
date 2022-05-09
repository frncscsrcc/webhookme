WebHook me
===
Test the HTTP requests received by your webhooks. No logs, no data saved.


Build
---
```
go build -o webhookme main.go
```

Usage
---
```
  -base string
        Base path to build the links (default "http://localhost")
  -global-max-request-per-minute int
        how many request per minute the server can accept (default 60)
  -link-lenght int
        Size of the link's random token (default 8)
  -max-body-size int
        Max size of the body in bytes (default 500000)
  -port string
        Port to expose (default "8080")
  -ttl int
        Sessions TTL (sec) (default 300)
```