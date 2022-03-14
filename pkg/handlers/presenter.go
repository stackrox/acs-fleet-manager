package handlers

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/compat"
)

func PresentReferenceWith(id, obj interface{}, ObjectKind func(i interface{}) string, ObjectPath func(id string, obj interface{}) string) compat.ObjectReference {
	refId, ok := MakeReferenceId(id)

	if !ok {
		return compat.ObjectReference{}
	}

	return compat.ObjectReference{
		Id:   refId,
		Kind: ObjectKind(obj),
		Href: ObjectPath(refId, obj),
	}
}

func MakeReferenceId(id interface{}) (string, bool) {
	var refId string

	if i, ok := id.(string); ok {
		refId = i
	}

	if i, ok := id.(*string); ok {
		if i != nil {
			refId = *i
		}
	}

	return refId, refId != ""
}
