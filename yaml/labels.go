package yaml

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

// Labels is a sorted set of labels. Order has to be guaranteed upon
// instantiation.
type Labels []Label

// Label is a key/value pair of strings.
type Label struct {
	Name, Value string
}

// Range calls f on each label.
func (ls Labels) Range(f func(l Label)) {
	for _, l := range ls {
		f(l)
	}
}

const (
	labelSep = '\xfe' // Used at beginning of `Bytes` return.
	sep      = '\xff' // Used between labels in `Bytes` and `Hash`.
)

// A LabelName is a key for a LabelSet or Metric.  It has a value associated
// therewith.
type LabelName string

// IsValidLegacy returns true iff name matches the pattern of LabelNameRE for
// legacy names. It does not use LabelNameRE for the check but a much faster
// hardcoded implementation.
func (ln LabelName) IsValidLegacy() bool {
	if len(ln) == 0 {
		return false
	}
	for i, b := range ln {
		if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || (b >= '0' && b <= '9' && i > 0)) {
			return false
		}
	}
	return true
}
func (ls Labels) String() string {
	var bytea [1024]byte // On stack to avoid memory allocation while building the output.
	b := bytes.NewBuffer(bytea[:0])

	b.WriteByte('{')
	i := 0
	ls.Range(func(l Label) {
		if i > 0 {
			b.WriteByte(',')
			b.WriteByte(' ')
		}
		if !LabelName(l.Name).IsValidLegacy() {
			b.Write(strconv.AppendQuote(b.AvailableBuffer(), l.Name))
		} else {
			b.WriteString(l.Name)
		}
		b.WriteByte('=')
		b.Write(strconv.AppendQuote(b.AvailableBuffer(), l.Value))
		i++
	})
	b.WriteByte('}')
	return b.String()
}

// Bytes returns ls as a byte slice.
// It uses an byte invalid character as a separator and so should not be used for printing.
func (ls Labels) Bytes(buf []byte) []byte {
	b := bytes.NewBuffer(buf[:0])
	b.WriteByte(labelSep)
	for i, l := range ls {
		if i > 0 {
			b.WriteByte(sep)
		}
		b.WriteString(l.Name)
		b.WriteByte(sep)
		b.WriteString(l.Value)
	}
	return b.Bytes()
}

// Len 返回Labels的长度
func (ls Labels) Len() int { return len(ls) }

// Swap labels交换
func (ls Labels) Swap(i, j int) { ls[i], ls[j] = ls[j], ls[i] }

// Less returns true iff ls[i].Name < ls[j].Name.
func (ls Labels) Less(i, j int) bool { return ls[i].Name < ls[j].Name }

// Map returns a string map of the labels.
func (ls Labels) Map() map[string]string {
	m := make(map[string]string)
	ls.Range(func(l Label) {
		m[l.Name] = l.Value
	})
	return m
}

// FromMap returns new sorted Labels from the given map.
func FromMap(m map[string]string) Labels {
	l := make([]Label, 0, len(m))
	for k, v := range m {
		l = append(l, Label{Name: k, Value: v})
	}
	return New(l...)
}

// New returns a sorted Labels from the given labels.
// The caller has to guarantee that all label names are unique.
func New(ls ...Label) Labels {
	set := make(Labels, 0, len(ls))
	set = append(set, ls...)
	slices.SortFunc(set, func(a, b Label) int { return strings.Compare(a.Name, b.Name) })

	return set
}

// 通过实现插件实现在发射过程中支持自定义标签

// MarshalJSON implements json.Marshaler.
func (ls Labels) MarshalJSON() ([]byte, error) {
	return json.Marshal(ls.Map())
}

// UnmarshalJSON implements json.Unmarshaler.
func (ls *Labels) UnmarshalJSON(b []byte) error {
	// 先定义其他类型，避免在进行反射过程中出现无线递归
	var m map[string]string

	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	*ls = FromMap(m)
	return nil
}

// MarshalYAML implements yaml.Marshaler.
func (ls Labels) MarshalYAML() (interface{}, error) {
	return ls.Map(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (ls *Labels) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var m map[string]string
	// 按照map的形式解析自定义类型 Labels
	if err := unmarshal(&m); err != nil {
		return err
	}

	*ls = FromMap(m)
	return nil
}
