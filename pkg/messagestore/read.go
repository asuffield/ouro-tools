package messagestore

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/asuffield/ouro-tools/pkg/messagestore/parse"
	"github.com/asuffield/ouro-tools/pkg/stringtable"
)

func (s *Store) Read(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return s.ReadDir(path)
	}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return err
	}

	var signature uint32
	err = binary.Read(f, binary.LittleEndian, &signature)
	if err != nil {
		return err
	}
	if signature == BinarySignature {
		return s.ReadBin(f, path)
	}

	f.Seek(0, 0)
	return s.ReadText(f, path)
}

func (s *Store) ReadDir(path string) error {
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk %s: %s", path, err)
		}
		if !info.IsDir() && !strings.HasSuffix(path, ".bak") {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %s", path, err)
			}
			return s.ReadText(f, path)
		}
		return nil
	})
}

func (s *Store) ReadBin(r io.Reader, path string) error {
	s.readBinary = true

	messageTable := &stringtable.Table{}
	if err := messageTable.Read(r); err != nil {
		return fmt.Errorf("failed to read messages table from %s: %s", path, err)
	}

	variableTable := &stringtable.Table{}
	if err := variableTable.Read(r); err != nil {
		return fmt.Errorf("failed to read variable string table from %s: %s", path, err)
	}

	var messageCount uint32
	if err := binary.Read(r, binary.LittleEndian, &messageCount); err != nil {
		return fmt.Errorf("failed to read message count from %s: %s", path, err)
	}

	for i := uint32(0); i < messageCount; i++ {
		var l uint32
		if err := binary.Read(r, binary.LittleEndian, &l); err != nil {
			return fmt.Errorf("failed to read length of string %d from %s: %s", i, path, err)
		}

		data := make([]byte, l)
		if n, err := r.Read(data); err != nil || n != int(l) {
			return fmt.Errorf("failed to read string %d from %s: %s", i, path, err)
		}
		name := string(data)

		var index, helpIndex uint32
		if err := binary.Read(r, binary.LittleEndian, &index); err != nil {
			return fmt.Errorf("failed to read index of string %d from %s: %s", i, path, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &helpIndex); err != nil {
			return fmt.Errorf("failed to read help index of string %d from %s: %s", i, path, err)
		}
		msg := s.insert(name)
		msg.index = int(index)
		msg.helpIndex = int(helpIndex)

		var varCount uint32
		if err := binary.Read(r, binary.LittleEndian, &varCount); err != nil {
			return fmt.Errorf("failed to read variable count of string %d from %s: %s", i, path, err)
		}

		for j := uint32(0); j < varCount; j++ {
			var index uint32
			if err := binary.Read(r, binary.LittleEndian, &index); err != nil {
				return fmt.Errorf("failed to read variable index %s of string %d from %s: %s", j, i, path, err)
			}
			msg.varIndices = append(msg.varIndices, int(index))
		}
	}

	s.messageTable = messageTable
	s.variableTable = variableTable

	return nil
}

func (s *Store) parse(r io.Reader, path string, entrypoint string) (*parse.MessageFile, error) {
	if s.hasInputFile(path) {
		return nil, fmt.Errorf("already read file %s", path)
	}

	if s.Verbose {
		fmt.Printf("reading %s\n", path)
	}

	res, errs := parse.ParseReader(path, r,
		parse.Entrypoint(entrypoint),
		parse.AllowInvalidUTF8(true))
	if errs != nil {
		return nil, errs
	}

	mf := res.(*parse.MessageFile)
	s.addInputFile(path, mf)

	return mf, nil
}

func (s *Store) ReadText(r io.Reader, path string) error {
	msgFile, err := s.parse(r, path, "MessageFile")
	if err != nil {
		return err
	}
	for _, line := range msgFile.Lines {
		if imp, ok := line.(*parse.Import); ok {
			evalName := func(n string) string {
				return filepath.Join(filepath.Dir(path), n)
			}
			if err := s.Read(evalName(imp.MessageFile)); err != nil {
				return err
			}
			if imp.TypeFile != "" {
				if err := s.ReadType(evalName(imp.TypeFile)); err != nil {
					return err
				}
			}
		}

		for _, m := range line.Messages() {
			if s.find(m.Id) != nil {
				// Ignore duplicates, take the first definition (that's how the format works)
				continue
			}
			msg := s.insert(m.Id)
			i := s.messageTable.Add(m.Content)
			if s.useHelpIndex {
				msg.helpIndex = i
			} else {
				msg.index = i
			}
		}
		//fmt.Print(line.Format())
	}
	return nil
}

func (s *Store) ReadType(path string) error {
	f, err := os.Open(path)
	defer f.Close()

	msgFile, err := s.parse(f, path, "MessageTypeFile")
	if err != nil {
		return err
	}

	for _, line := range msgFile.Lines {
		for _, t := range line.Types() {
			indices := []int{}
			for _, v := range t.Vars {
				i := s.variableTable.Add(v.Name)
				j := s.variableTable.Add(v.Ty)
				// This is as crazy as it sounds, but it's how the format works
				if j != i+1 {
					panic("messagestore format requires adjacent indices for types")
				}
				indices = append(indices, i)
			}
			// Each type entry is applied to all four instances of the message
			for _, prefix := range []string{"", "v_", "p_", "l_"} {
				id := prefix + t.Id
				if msg := s.find(id); msg != nil {
					msg.varIndices = append(msg.varIndices, indices...)
				}
			}
		}
		//fmt.Print(line.Format())
	}

	return nil
}
