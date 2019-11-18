package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"text/template"

	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type TempConf struct {
	TargetPtah string
	Resources  []string
}

func main() {
	var targetPtah string
	flag.StringVar(&targetPtah, "path", "../../docs/resources/", "generate target path")
	flag.Parse()

	tmpFile := randomdata.RandString(10) + ".go"
	resourcesName := getReosurcesName()
	if err := genGofile(resourcesName, targetPtah, tmpFile); err != nil {
		panic(err)
	}
	_, err := os.Stat(tmpFile)
	if err != nil {
		panic(err)
	}
	if err := goRun(tmpFile); err != nil {
		panic(err)
	}
	if err := os.Remove(tmpFile); err != nil {
		panic(err)
	}
	log.Printf("generate resource doc success")
}

func getReosurcesName() []string {
	resourcesName := make([]string, len(types.Resources()))
	for i, resource := range types.Resources() {
		resourcesName[i] = string(reflect.TypeOf(resource).Name())
	}
	return resourcesName
}

func genGofile(resourcesName []string, targetPtah, file string) error {
	tp, err := template.New("tp").Parse(goTemp)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	conf := TempConf{
		TargetPtah: targetPtah,
		Resources:  resourcesName,
	}
	if err := tp.Execute(buf, conf); err != nil {
		return err
	}
	if err = ioutil.WriteFile(file, []byte(buf.String()), 0644); err != nil {
		return err
	}
	return nil
}

func goRun(file string) error {
	if _, err := exec.Command("go", "run", file).Output(); err != nil {
		return err
	}
	return nil
}

const goTemp = `
package main

import (
        "log"

        restresource "github.com/zdnscloud/gorest/resource"
        "github.com/zdnscloud/gorest/resource/schema"
        "github.com/zdnscloud/singlecloud/pkg/handler"
        "github.com/zdnscloud/singlecloud/pkg/types"
)

var (
        Version = restresource.APIVersion{
                Version: "v1",
                Group:   "zcloud.cn",
        }
)

func main() {
	targetPtah := "{{.TargetPtah}}"
        schemas := importResource()
        if err := schemas.WriteJsonDocs(&Version, targetPtah); err != nil {
                log.Fatalf("generate resource doc failed. %s", err.Error())
        }
        log.Printf("generate resource doc success")
}
func importResource() *schema.SchemaManager {
        schemas := schema.NewSchemaManager()
        {{range .Resources}}
        schemas.MustImport(&Version, types.{{.}}{}, &handler.{{.}}Manager{}){{end}}
        return schemas
}
`
