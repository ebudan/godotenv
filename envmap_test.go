package godotenv

import (
	"testing"
)

func TestEnvMap(t *testing.T) {
	m := NewEnvMap()
	old, at := m.Set("a", "A")
	if old != "" || at != -1 {
		t.Errorf("Failed set 1")
	}
	old, at = m.Set("b", "B")
	if old != "" || at != -1 {
		t.Errorf("Failed set 2")
	}
	old, at = m.Set("c", "C")
	if old != "" || at != -1 {
		t.Errorf("Failed set 3")
	}

	// m.Emit(os.Stdout, true)

	if m.Len() != 3 {
		t.Errorf("Invalid len")
	}
	k, at := m.Get("a")
	if k != "A" || at != 0 {
		t.Errorf("Failed get 'a'")
	}
	k, at = m.Get("b")
	if k != "B" || at != 1 {
		t.Errorf("Failed get 'a'")
	}
	k, at = m.Get("c")
	if k != "C" || at != 2 {
		t.Errorf("Failed get 'a'")
	}

	old, at = m.SetAt("o", "O", 0)
	if old != "" || at != -1 {
		t.Errorf("Failed set at 0")
	}
	old, at = m.SetAt("u", "U", m.Len())
	if old != "" || at != -1 {
		t.Errorf("Failed set at last")
	}
	k, at = m.Get("o")
	if k != "O" || at != 0 {
		t.Errorf("Failed get 'o'")
	}
	k, at = m.Get("u")
	if k != "U" || at != 4 {
		t.Errorf("Failed get 'U'")
	}
	k, at = m.Get("a")
	if k != "A" || at != 1 {
		t.Errorf("Failed get 'a' #2")
	}

	// m.Emit(os.Stdout, true)

	k, at = m.Remove("b")
	if k != "B" || at != 2 || m.Len() != 4 {
		t.Errorf("Failed remove")
	}
	k, at = m.Remove("u")
	if k != "U" || at != 3 || m.Len() != 3 {
		t.Errorf("Failed remove")
	}
	k, at = m.RemoveAt(2)
	if k != "C" || at != 2 || m.Len() != 2 {
		t.Errorf("Failed remove at")
	}

	// m.Emit(os.Stdout, true)

}

func TestEnvMapIter(t *testing.T) {
	m := NewEnvMap()
	m.Set("0", "A")
	m.Set("1", "B")
	m.Set("2", "C")

	found := 0
	m.Iter(func(k, v string) { found += 1 })
	if found != 3 {
		t.Errorf("Failed iter test")
	}
	// TBD
}
