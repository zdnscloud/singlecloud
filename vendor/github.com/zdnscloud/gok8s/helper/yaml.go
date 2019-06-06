package helper

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/zdnscloud/gok8s/client"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func CreateResourceFromYaml(cli client.Client, yaml string) error {
	return mapOnRuntimeObject(yaml, cli.Create)
}

func DeleteResourceFromYaml(cli client.Client, yaml string) error {
	return mapOnRuntimeObject(yaml, func(ctx context.Context, obj runtime.Object) error {
		return cli.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationForeground))
	})
}

func UpdateResourceFromYaml(cli client.Client, yaml string) error {
	return mapOnRuntimeObject(yaml, cli.Update)
}

func mapOnYamlDocument(data string, fn func([]byte) error) error {
	reader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) > 4 {
			doc = bytes.TrimPrefix(doc, []byte("---\n"))
		}

		if len(doc) == 0 {
			continue
		}

		if err := fn(doc); err != nil {
			return err
		}
	}
	return nil
}

func mapOnRuntimeObject(data string, fn func(context.Context, runtime.Object) error) error {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	return mapOnYamlDocument(data, func(doc []byte) error {
		obj, _, err := decode(doc, nil, nil)
		if err != nil {
			if strings.Index(err.Error(), "no kind") != -1 {
				json, err := yaml.ToJSON([]byte(data))
				if err != nil {
					return err
				}
				obj, _, err = unstructured.UnstructuredJSONScheme.Decode(json, nil, nil)
				if err != nil {
					return err
				}
			}
		}

		if err := fn(context.TODO(), obj); err != nil {
			if apierrors.IsAlreadyExists(err) == false &&
				apierrors.IsNotFound(err) == false {
				return err
			}
		}
		return nil
	})
}
