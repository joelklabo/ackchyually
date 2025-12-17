package tomledit

import (
	"bytes"
	"errors"

	"github.com/BurntSushi/toml"
)

type Document struct {
	data map[string]any
}

func Parse(b []byte) (*Document, error) {
	doc := &Document{data: map[string]any{}}
	if len(bytes.TrimSpace(b)) == 0 {
		return doc, nil
	}

	if err := toml.Unmarshal(b, &doc.data); err != nil {
		return nil, err
	}
	return doc, nil
}

func (d *Document) Get(path ...string) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}

	var cur any = d.data
	for _, k := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[k]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func (d *Document) Set(value any, path ...string) error {
	if len(path) == 0 {
		return errors.New("tomledit: empty path")
	}

	cur := d.data
	for i, k := range path {
		isLeaf := i == len(path)-1
		if isLeaf {
			cur[k] = value
			return nil
		}

		next, ok := cur[k]
		if !ok {
			child := map[string]any{}
			cur[k] = child
			cur = child
			continue
		}

		child, ok := next.(map[string]any)
		if ok {
			cur = child
			continue
		}

		// Existing value is not a table; replace with a table so we can keep going.
		child = map[string]any{}
		cur[k] = child
		cur = child
	}

	return nil
}

func (d *Document) Delete(path ...string) bool {
	if len(path) == 0 {
		return false
	}

	cur := d.data
	for i, k := range path {
		isLeaf := i == len(path)-1
		if isLeaf {
			if _, ok := cur[k]; !ok {
				return false
			}
			delete(cur, k)
			return true
		}

		next, ok := cur[k]
		if !ok {
			return false
		}
		child, ok := next.(map[string]any)
		if !ok {
			return false
		}
		cur = child
	}

	return false
}

func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(d.data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
