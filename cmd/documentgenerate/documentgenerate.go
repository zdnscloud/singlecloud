package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"text/template"

	"github.com/zdnscloud/singlecloud/pkg/types"
)

const goTemp = `
package main
        
import (
        "log"   
                        
        "github.com/zdnscloud/gorest/resource/schema"
        "github.com/zdnscloud/singlecloud/pkg/handler"
        "github.com/zdnscloud/singlecloud/pkg/types"
)                                       

func main() {                                                                           
        targetPath := "{{.TargetPath}}"                                                         
        schemas := importResource()                                                                     
        if err := schemas.WriteJsonDocs(&handler.Version, targetPath); err != nil {                                     
                log.Fatalf("generate resource doc failed. %s", err.Error())                                                     
        }                                                                                                                               
}                                                                                                                                       

func importResource() *schema.SchemaManager {                                                                                           
        schemas := schema.NewSchemaManager()                                                                                                    
        {{range .Resources}}                                                                                                                        
        schemas.MustImport(&handler.Version, types.{{.}}{}, &handler.{{.}}Manager{})                                                                              
        {{end}}                                                                                                                                                 
        return schemas                                                                                                                                  
}  
`

func main() {
	var targetPath string
	flag.StringVar(&targetPath, "path", "../../docs/resources/", "generate target path")
	flag.Parse()

	if err := genGoTmpfileAndRunIt(getReosurcesName(), targetPath); err != nil {
		log.Fatalf("generate resource doc failed. %s", err.Error())
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

func genGoTmpfileAndRunIt(resourcesName []string, targetPath string) error {
	tmpFile, err := ioutil.TempFile("./", "doc*.go")
	if err != nil {
		return fmt.Errorf("generate tmp file failed, %v", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	buf := new(bytes.Buffer)
	tp := template.Must(template.New("compiled_template").Parse(goTemp))
	conf := struct {
		TargetPath string
		Resources  []string
	}{
		targetPath,
		resourcesName,
	}
	if err := tp.Execute(buf, conf); err != nil {
		return fmt.Errorf("generate go file failed, %v", err)
	}
	tmpFile.WriteString(buf.String())
	out, err := exec.Command("go", "run", tmpFile.Name()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("go run failed.\nCmd Out:%s\nErr: %v", strings.TrimSpace(string(out)), err)
	}
	return nil
}
