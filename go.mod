module github.com/zdnscloud/singlecloud

go 1.13

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/boltdb/bolt v1.3.2-0.20180302180052-fd01fc79c553 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.5.0
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/websocket v1.4.1
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kyokomi/emoji v2.1.0+incompatible
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/zdnscloud/cement v0.0.0-20200203063149-0351bc244b72
	github.com/zdnscloud/g53 v0.0.0-20191119101753-eb2b1813bd52
	github.com/zdnscloud/gok8s v0.0.0-20200205030309-01bcca9746a5
	github.com/zdnscloud/goproxy v0.0.0-20190815040552-89eeea17b1b4
	github.com/zdnscloud/gorest v0.0.0-20200111091734-c24da01dcca2
	github.com/zdnscloud/immense v0.0.0-20191225033521-e95fae7ebb2a
	github.com/zdnscloud/iniconfig v0.0.0-20191105013537-c8624280493d
	github.com/zdnscloud/kvzoo v0.0.0-20191105090530-d32f8b8c073f
	github.com/zdnscloud/servicemesh v0.0.0-20191212031042-0f3403ce956f
	github.com/zdnscloud/vanguard v0.0.0-20191127091955-d7bf8860bb40
	github.com/zdnscloud/zke v0.0.0-20200205053350-570c9d92a05b
	github.com/zsais/go-gin-prometheus v0.1.0
	golang.org/x/crypto v0.0.0-20191219195013-becbf705a915 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6 // indirect
	google.golang.org/genproto v0.0.0-20191223191004-3caeed10a8bf // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/yaml.v2 v2.2.7
	helm.sh/helm v3.0.0-alpha.2.0.20190820153828-fba311ba2362+incompatible
	k8s.io/api v0.0.0-20191004102255-dacd7df5a50b
	k8s.io/apiextensions-apiserver v0.0.0-20191004105443-a7d558db75c6
	k8s.io/apimachinery v0.0.0-20191004074956-01f8b7d1121a
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/metrics v0.0.0-20191004105814-56635b1b5a0c
)

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	helm.sh/helm => github.com/helm/helm v3.0.0-alpha.2.0.20190820153828-fba311ba2362+incompatible
)
