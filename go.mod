module github.com/gopherd/gopherd

go 1.16

require (
	github.com/gopherd/doge v0.0.7
	github.com/gopherd/jwt v0.0.2
	github.com/gopherd/log v0.1.5
	github.com/gopherd/redis v0.0.11
	github.com/gopherd/zmq v0.0.6
	github.com/oschwald/geoip2-golang v1.5.0 // indirect
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	google.golang.org/api v0.52.0
	google.golang.org/protobuf v1.27.1
)

replace github.com/gopherd/doge => ../doge
