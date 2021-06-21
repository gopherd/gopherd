module github.com/gopherd/gopherd

go 1.16

require (
	github.com/gopherd/doge v0.0.7
	github.com/gopherd/redis v0.0.4
	github.com/mkideal/log v1.1.14
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	google.golang.org/protobuf v1.26.0
)

replace (
	github.com/gopherd/doge => ../doge
	github.com/gopherd/redis => ../redis
)
