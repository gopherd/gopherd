module github.com/gopherd/gopherd

go 1.16

require (
	github.com/gopherd/doge v0.0.7
	github.com/gopherd/jwt v0.0.0
	github.com/gopherd/redis v0.0.4
	github.com/gopherd/zmq v0.0.0
	github.com/gopherd/log v0.0.0
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	google.golang.org/protobuf v1.26.0
)

replace (
	github.com/gopherd/doge => ../doge
	github.com/gopherd/jwt => ../jwt
	github.com/gopherd/redis => ../redis
	github.com/gopherd/zmq => ../zmq
	github.com/gopherd/log => ../log
)
