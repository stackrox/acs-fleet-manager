package shared

import (
	"encoding/json"
	"fmt"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffAsJson ...
func DiffAsJson(a interface{}, b interface{}, aName string, bName string) string {
	aData, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Sprintf("could not marshal %s to json: %v", aName, err)
	}
	bData, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Sprintf("could not marshal %s to json: %v", bName, err)
	}
	res, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(aData)),
		B:        difflib.SplitLines(string(bData)),
		FromFile: aName,
		FromDate: "",
		ToFile:   bName,
		ToDate:   "",
		Context:  1,
	})
	if err != nil {
		return fmt.Sprintf("diff failed: %v", err)
	}
	return "\n" + res
}
