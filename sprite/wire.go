package sprite

import "encoding/json"

func jsonUnmarshalString(data []byte, out *string) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	*out = name
	return nil
}
