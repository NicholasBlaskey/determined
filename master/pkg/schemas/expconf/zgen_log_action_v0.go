// Code generated by gen.py. DO NOT EDIT.

package expconf

import (
	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func (l LogActionV0) Type() LogActionType {
	return l.RawType
}

func (l *LogActionV0) SetType(val LogActionType) {
	l.RawType = val
}

func (l LogActionV0) ParsedSchema() interface{} {
	return schemas.ParsedLogActionV0()
}

func (l LogActionV0) SanityValidator() *jsonschema.Schema {
	return schemas.GetSanityValidator("http://determined.ai/schemas/expconf/v0/log-action.json")
}

func (l LogActionV0) CompletenessValidator() *jsonschema.Schema {
	return schemas.GetCompletenessValidator("http://determined.ai/schemas/expconf/v0/log-action.json")
}
