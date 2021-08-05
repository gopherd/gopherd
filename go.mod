module github.com/gopherd/gopherd

go 1.16

require (
	github.com/futurenda/google-auth-id-token-verifier v0.0.0-20170311140316-2a5b89f28b7e
	github.com/gopherd/doge v0.0.7
	github.com/gopherd/jwt v0.0.2
	github.com/gopherd/log v0.1.3
	github.com/gopherd/redis v0.0.11
	github.com/gopherd/zmq v0.0.6
	golang.org/x/net v0.0.0-20210610132358-84b48f89b13b // indirect
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914 // indirect
	google.golang.org/protobuf v1.26.0
)

replace github.com/gopherd/doge => ../doge
