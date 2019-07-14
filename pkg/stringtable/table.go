package stringtable

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

type Table struct {
	strings []string
	index   map[string]int
}

func New() *Table {
	return &Table{
		index: map[string]int{},
	}
}

func (t *Table) Read(r io.Reader) error {
	var count, byteLen uint32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &byteLen); err != nil {
		return err
	}

	data := make([]byte, byteLen)
	if n, err := r.Read(data); err != nil || n != int(byteLen) {
		return fmt.Errorf("failed to read stringtable, got %d of %d bytes, err %s", n, byteLen, err)
	}

	strings := bytes.Split(data, []byte{0})
	if len(strings) < int(count) {
		return fmt.Errorf("failed to read stringtable, got %d of %d strings", len(strings), count)
	}

	for _, s := range strings[0:count] {
		t.strings = append(t.strings, string(s))
	}

	t.reindex()

	return nil
}

func writeU32(w io.Writer, i int) error {
	u := uint32(i)
	return binary.Write(w, binary.LittleEndian, &u)
}

func (t *Table) Write(rw io.Writer) error {
	w := bufio.NewWriter(rw)
	if err := writeU32(w, len(t.strings)); err != nil {
		return err
	}

	l := 0
	for _, s := range t.strings {
		l += len(s) + 1
	}
	if err := writeU32(w, l); err != nil {
		return err
	}

	for _, s := range t.strings {
		w.WriteString(s)
		w.WriteByte(0)
	}

	return w.Flush()
}

func (t *Table) reindex() {
	index := map[string]int{}
	for i, v := range t.strings {
		index[v] = i
	}
	t.index = index
}

func (t *Table) Summary() string {
	return fmt.Sprintf("stringtable<len %d>", len(t.strings))
}

func (t *Table) FindOrAdd(v string) int {
	if i, ok := t.index[v]; ok {
		return i
	}
	i := len(t.strings)
	t.strings = append(t.strings, v)
	t.index[v] = i
	return i
}

func (t *Table) Add(v string) int {
	i := len(t.strings)
	t.strings = append(t.strings, v)
	t.index[v] = i
	return i
}

func (t *Table) Get(i int) string {
	return t.strings[i]
}

func (t *Table) Hash(w io.Writer, sorted bool) {
	ss := t.strings
	if sorted {
		ss = append(ss[:0:0], ss...)
		sort.Strings(ss)
	}
	bs, err := json.Marshal(ss)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(bs)
	if err != nil {
		panic(err)
	}
}
