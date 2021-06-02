package godotenv

import (
	"bytes"
	"fmt"
	"io"
	"math"
)

type Pair struct {
	Key, Val string
}

type EnvMap struct {
	entries []Pair
	keys    map[string]int
}

func NewEnvMap() *EnvMap {
	return &EnvMap{keys: make(map[string]int)}
}

func (m *EnvMap) Len() int {
	return len(m.entries)
}

// Set stores a key-value pair in the map.
// If the key existed previously, the entry remains in place, and the old
// value and its index are returned.
// If the key did not exist, an empty string and negative index are returned.
func (m *EnvMap) Set(key, val string) (string, int) {
	var r string
	var ok bool
	var at int
	if at, ok = m.keys[key]; ok {
		r = m.entries[at].Val
		m.entries[at] = Pair{Key: key, Val: val}
	} else {
		at = -1
		m.entries = append(m.entries, Pair{Key: key, Val: val})
		m.keys[key] = len(m.entries) - 1
	}
	return r, at
}

// Set stores a key-value pair in the map.
// If the key existed previously, the place of the key is moved, and the old
// value and index are returned.
// If the key did not exist, an empty string and negative index are returned.
func (m *EnvMap) SetAt(key, val string, at int) (string, int) {
	ex, ok := m.keys[key]
	var was string
	if ok {
		was = m.entries[ex].Val
		rpl := m.entries[:ex]
		if ex < len(m.entries)-1 {
			rpl = append(rpl, m.entries[ex+1])
		}
		m.entries = rpl
		if ex < at {
			at -= 1
		}
	} else {
		ex = -1
	}
	if at < 0 {
		at = 0
	}
	if at > len(m.entries) {
		at = len(m.entries)
	}

	var rpl []Pair
	rpl = append(rpl, m.entries[:at]...)
	rpl = append(rpl, Pair{Key: key, Val: val})
	rpl = append(rpl, m.entries[at:]...)
	m.entries = rpl
	m.keys = make(map[string]int)
	for ix, pair := range m.entries {
		m.keys[pair.Key] = ix
	}

	return was, ex
}

// Get returns a keyed value and its place in our collection, or
// empty and a negative number if it did not exist.
func (m *EnvMap) Get(key string) (string, int) {
	r, ok := m.keys[key]
	if ok {
		return m.entries[r].Val, r
	}
	return "", -1
}

// GetAt returns a key-value pair at the specified position in the map,
// or an invalid value and negative number if no such index is used.
//
// This function can be used for iterating the ordered contents, but it
// is slightly inconvenient. Prefer Iter().
func (m *EnvMap) GetAt(at int) (Pair, bool, int) {
	if at < 0 || at >= len(m.entries) {
		return Pair{}, false, -1
	}
	r := m.entries[at]
	s := at + 1
	if s >= len(m.entries) {
		s = -1
	}
	return r, true, s
}

// Iter calls the provided callback for each entry, in order.
func (m *EnvMap) Iter(f func(key, val string)) {
	for _, p := range m.entries {
		f(p.Key, p.Val)
	}
}

// Remove deletes an entry from the map, returning the old value and index,
// or an empty and negative index if not present.
func (m *EnvMap) Remove(key string) (string, int) {
	var was string
	at := -1
	r, ok := m.keys[key]
	if ok {
		at = r
		was = m.entries[at].Val
		m.entries = append(m.entries[:at], m.entries[at+1:]...)
		m.keys = make(map[string]int)
		for ix, pair := range m.entries {
			m.keys[pair.Key] = ix
		}
	}
	return was, at
}

// Removes a key-value entry at the provided index, returning the old
// value and index or emtpy and -1 if index was not valid.
func (m *EnvMap) RemoveAt(at int) (string, int) {
	var was string
	if at < 0 || at > len(m.entries) {
		return "", -1
	}
	pair := m.entries[at]
	was = pair.Val
	m.entries = append(m.entries[:at], m.entries[at+1:]...)
	m.keys = make(map[string]int)
	for ix, pair := range m.entries {
		m.keys[pair.Key] = ix
	}
	return was, at
}

// Emits the contents of the map to the writer, optionally with line numbers.
func (m *EnvMap) Emit(w io.Writer, linenos bool) {
	var buf bytes.Buffer
	form := formatIx(len(m.entries))
	for ix, p := range m.entries {
		if linenos {
			buf.WriteString(fmt.Sprintf(form, ix))
		}
		buf.WriteString(p.Key + "=\"" + p.Val + "\"\n")
	}
	w.Write(buf.Bytes())
}

func formatIx(max int) string {
	n := int(math.Log10(float64(max))) + 1
	f := "%0" + fmt.Sprintf("%d", n) + "d "
	return f
}
