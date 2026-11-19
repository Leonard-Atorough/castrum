package sprite

import "encoding/json"

// jsonUnmarshalString decodes a JSON string, so Kind.UnmarshalJSON can
// reject unknown wire identities without duplicating decode rules.
func jsonUnmarshalString(data []byte, out *string) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	*out = name
	return nil
}
