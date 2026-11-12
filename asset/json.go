package asset

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// SetJSON installs or replaces JSON encoding and decoding for T atomically.
// Existing loaders and savers are unaffected.
//
// Decoding reads one complete JSON value, allowing trailing whitespace but
// rejecting additional content. Standard encoding/json field and custom
// marshaling rules apply; missing fields retain their zero values. Domain
// validation, defaults, and migration remain the caller's responsibility.
func (c *Codecs) SetJSON[T any]() error {
	if c == nil {
		return fmt.Errorf("codecs cannot be nil")
	}
	key := codecKey{reflect.TypeFor[T](), FormatJSON}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.decoders == nil {
		c.decoders = make(map[codecKey]any)
	}
	if c.encoders == nil {
		c.encoders = make(map[codecKey]any)
	}
	c.decoders[key] = Decoder[T](decodeJSON[T])
	c.encoders[key] = Encoder[T](func(w io.Writer, value T) error {
		return json.NewEncoder(w).Encode(value)
	})
	return nil
}

func decodeJSON[T any](reader io.Reader) (T, error) {
	var value T
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&value); err != nil {
		return value, err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return value, err
		}
		return value, fmt.Errorf("unexpected additional JSON value")
	}
	return value, nil
}
