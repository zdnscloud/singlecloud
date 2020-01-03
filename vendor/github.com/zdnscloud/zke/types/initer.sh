#!/bin/sh

cat >>initer.go <<EOF
package types

const imageYaml = \`$(cat $1)\`

func init() {
	mustLoadImage(imageYaml)
}
EOF

