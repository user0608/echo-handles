package echohandles

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JsonField struct {
	Field string
	Value any
}
type JsonObject []JsonField

// / json marshal
func (o JsonObject) MarshalJSON() ([]byte, error) {
	var parts []string
	for _, v := range o {
		bytesValue, err := json.Marshal(v.Value)
		if err != nil {
			return nil, err
		}
		parts = append(parts, fmt.Sprintf(`"%s":%s`, v.Field, string(bytesValue)))
	}
	return []byte("{" + strings.Join(parts, ",") + "}"), nil
}
