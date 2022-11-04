// Package secrets ...
package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/santhosh-tekuri/jsonschema/v3"
	"github.com/spyzhov/ajson"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
)

// ModifySecrets ...
func ModifySecrets(schemaDom, doc api.JSON, f func(node *ajson.Node) error) (api.JSON, error) {
	secrets, err := modifySecrets(schemaDom, doc, f)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

func modifySecrets(schemaBytes, doc []byte, f func(node *ajson.Node) error) ([]byte, error) {
	fields, err := getPathsToPasswordFields(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("retrieving paths to password fields: %w", err)
	}

	root, err := ajson.Unmarshal(doc)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}

	for _, field := range fields {
		nodes, _ := root.JSONPath("$" + field)
		for _, node := range nodes {
			err := f(node)
			if err != nil {
				return nil, err
			}
		}

	}
	bytes, err := ajson.Marshal(root)
	if err != nil {
		return bytes, fmt.Errorf("marshalling: %w", err)
	}
	return bytes, nil
}

func getPathsToPasswordFields(schemaBytes []byte) ([]string, error) {

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return nil, fmt.Errorf("adding JSON schema to compiler: %w", err)
	}
	schema, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("compiling JSON schema: %w", err)
	}

	paths := map[string]bool{}
	findPathsMatching(schema, "", paths, func(path string, s *jsonschema.Schema) bool {
		if s.OneOf != nil {
			if len(s.OneOf) == 2 {
				// var s1, s2 *jsonschema.Schema
				if isTypeAndFormat(s.OneOf[0], "string", "password") && isTypeAndFormat(s.OneOf[1], "object", "") {
					return true
				}
				if isTypeAndFormat(s.OneOf[1], "string", "password") && isTypeAndFormat(s.OneOf[0], "object", "") {
					return true
				}
			}
		}
		return false
	})

	var result []string
	for s := range paths {
		result = append(result, s)
	}
	return result, nil

}

func isTypeAndFormat(s *jsonschema.Schema, t string, format string) bool {
	if s.Format != format {
		return false
	}
	return reflect.DeepEqual(s.Types, []string{t})
}

func findPathsMatching(s *jsonschema.Schema, p string, results map[string]bool, matcher func(path string, s *jsonschema.Schema) bool) {

	if matcher(p, s) {
		results[p] = true
		return
	}

	if s.Properties != nil {
		for n, v := range s.Properties {
			data, _ := json.Marshal(n)
			path := p + "[" + string(data) + "]"
			findPathsMatching(v, path, results, matcher)
		}
	}

	if s.OneOf != nil {
		for _, v := range s.OneOf {
			findPathsMatching(v, p, results, matcher)
		}
	}

	if s.AnyOf != nil {
		for _, v := range s.AnyOf {
			findPathsMatching(v, p, results, matcher)
		}
	}

}
