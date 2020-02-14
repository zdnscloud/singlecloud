module github.com/zdnscloud/singlecloud

go 1.13

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/gin-contrib/static v0.0.0-20191128031702-f81c604d8ac2
	github.com/gin-gonic/gin v1.5.0
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/protobuf v1.3.3
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/gorilla/websocket v1.4.1
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kyokomi/emoji v2.1.0+incompatible
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.11.0 // indirect
	github.com/urfave/cli v1.22.2 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/zdnscloud/cement v0.0.0-20200205075737-175eefa2a628
	github.com/zdnscloud/g53 v0.0.0-20191119101753-eb2b1813bd52
	github.com/zdnscloud/gok8s v0.0.0-20200212071629-b06587f54ee6
	github.com/zdnscloud/goproxy v0.0.0-20200205075939-521cea33b942
	github.com/zdnscloud/gorest v0.0.0-20200212094753-5f6fe59d0aa1
	github.com/zdnscloud/immense v0.0.0-20200214070751-b25b082d4c39
	github.com/zdnscloud/iniconfig v0.0.0-20191105013537-c8624280493d
	github.com/zdnscloud/kvzoo v0.0.0-20200205072604-297aba5646f7
	github.com/zdnscloud/servicemesh v0.0.0-20200205073418-8a139a9aa55d
	github.com/zdnscloud/vanguard v0.0.0-20200214072003-226d0e690d9f
	github.com/zdnscloud/zke v0.0.0-20200214063343-8d345d8f60d1
	github.com/zsais/go-gin-prometheus v0.1.0
	golang.org/x/crypto v0.0.0-20200214034016-1d94cc7ab1c6 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm v3.0.0-alpha.2.0.20190820153828-fba311ba2362+incompatible
	k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver v0.17.2
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.2
	k8s.io/metrics v0.17.2
	k8s.io/utils v0.0.0-20200124190032-861946025e34 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	helm.sh/helm => github.com/helm/helm v3.0.0-alpha.2.0.20190820153828-fba311ba2362+incompatible
)
