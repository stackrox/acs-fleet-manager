package handlers

import (
	"github.com/stackrox/acs-fleet-manager/internal/central/compat"
)

// PresentReferenceWith ...
func PresentReferenceWith(id, obj interface{}, ObjectKind func(i interface{}) string, ObjectPath func(id string, obj interface{}) string) compat.ObjectReference {
	refID, ok := MakeReferenceID(id)

	if !ok {
		return compat.ObjectReference{}
	}

	return compat.ObjectReference{
		Id:   refID,
		Kind: ObjectKind(obj),
		Href: ObjectPath(refID, obj),
	}
}

// MakeReferenceID ...
func MakeReferenceID(id interface{}) (string, bool) {
	var refID string

	if i, ok := id.(string); ok {
		refID = i
	}

	if i, ok := id.(*string); ok {
		if i != nil {
			refID = *i
		}
	}

	return refID, refID != ""
}
