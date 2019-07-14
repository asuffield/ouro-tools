package messagestore

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asuffield/ouro-tools/pkg/messagestore/parse"
	"github.com/asuffield/ouro-tools/pkg/stringtable"
)

const BinarySignature = 20090521

type Store struct {
	Verbose bool
	BaseDir string

	readBinary    bool
	useHelpIndex  bool
	messageTable  *stringtable.Table
	variableTable *stringtable.Table
	messages      map[string]*Message
	inputFiles    map[string]*parse.MessageFile
}

type Message struct {
	id         string
	index      int
	helpIndex  int
	varIndices []int
}

func NewStore() *Store {
	return &Store{
		messageTable:  stringtable.New(),
		variableTable: stringtable.New(),
		inputFiles:    map[string]*parse.MessageFile{},
		messages:      map[string]*Message{},
	}
}

func (s *Store) Hash(w io.Writer, contentOnly bool) {
	if !contentOnly {
		s.messageTable.Hash(w, false)
		s.variableTable.Hash(w, false)
	}
	ids := []string{}
	for id := range s.messages {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		m := s.messages[id]
		i := m.index
		if s.useHelpIndex {
			i = m.helpIndex
		}
		data := []string{m.id, s.messageTable.Get(i)}
		for _, j := range m.varIndices {
			data = append(data, s.variableTable.Get(j), s.variableTable.Get(j+1))
		}
		bs, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		_, err = w.Write(bs)
		if err != nil {
			panic(err)
		}
	}
}

func (s *Store) Summary() string {
	return fmt.Sprintf("strings: %s, vars: %s, have %d messages", s.messageTable.Summary(), s.variableTable.Summary(), len(s.messages))
}

func (s *Store) MessageIDs() []string {
	ids := []string{}
	for _, msg := range s.messages {
		ids = append(ids, msg.id)
	}
	return ids
}

func (s *Store) HasMessage(id string) bool {
	_, ok := s.messages[strings.ToLower(id)]
	return ok
}

func (s *Store) Message(id string) string {
	msg, ok := s.messages[strings.ToLower(id)]
	if !ok {
		return ""
	}
	if s.useHelpIndex {
		return s.messageTable.Get(msg.helpIndex)
	} else {
		return s.messageTable.Get(msg.index)
	}
}

func (s *Store) MessageVarTypes(id string) map[string]string {
	types := map[string]string{}
	msg := s.messages[strings.ToLower(id)]
	if msg != nil {
		for _, index := range msg.varIndices {
			name := s.variableTable.Get(index)
			ty := s.variableTable.Get(index + 1)
			types[name] = ty
		}
	}
	return types
}

func (s *Store) tryAbs(path string) string {
	// Make abs, if possible
	a, err := filepath.Abs(path)
	if err == nil {
		path = a
	}

	if s.BaseDir == "" {
		return path
	}

	// Make relative, if possible
	base, err := filepath.Abs(s.BaseDir)
	if err != nil {
		return path
	}
	r, err := filepath.Rel(base, path)
	if err == nil {
		path = r
	}

	return path
}

func (s *Store) addInputFile(path string, f *parse.MessageFile) {
	path = s.tryAbs(path)
	s.inputFiles[path] = f
}

func (s *Store) hasInputFile(path string) bool {
	path = s.tryAbs(path)
	_, ok := s.inputFiles[path]
	return ok
}

func (s *Store) inputFile(path string) *parse.MessageFile {
	path = s.tryAbs(path)
	return s.inputFiles[path]
}

func (s *Store) insert(id string) *Message {
	m := s.find(id)
	if m == nil {
		m = &Message{id: id}
		s.messages[strings.ToLower(id)] = m
	}
	return m
}

func (s *Store) find(id string) *Message {
	if m, ok := s.messages[strings.ToLower(id)]; ok {
		return m
	} else {
		return nil
	}
}
