package messagestore

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

func (s *Store) Write(path string, template *Store) error {
	if template != nil {
		if template.readBinary {
			return s.WriteBin(path, template)
		} else {
			return s.WriteText(path, template)
		}
	}

	if strings.HasSuffix(path, ".bin") {
		return s.WriteBin(path, nil)
	} else {
		return s.WriteText(path, nil)
	}
}

func writeU32(w io.Writer, i int) error {
	u := uint32(i)
	return binary.Write(w, binary.LittleEndian, &u)
}

func (s *Store) WriteBin(path string, template *Store) error {
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		return err
	}

	if err := writeU32(f, BinarySignature); err != nil {
		return fmt.Errorf("failed to write message count to %s: %s", path, err)
	}

	if err := s.messageTable.Write(f); err != nil {
		return fmt.Errorf("failed to write messages table to %s: %s", path, err)
	}

	if err := s.variableTable.Write(f); err != nil {
		return fmt.Errorf("failed to write variable table to %s: %s", path, err)
	}

	if err := writeU32(f, len(s.messages)); err != nil {
		return fmt.Errorf("failed to write message count to %s: %s", path, err)
	}

	for name, msg := range s.messages {
		if err := writeU32(f, len(msg.id)); err != nil {
			return fmt.Errorf("failed to write length of string %s to %s: %s", name, path, err)
		}
		if _, err := f.WriteString(msg.id); err != nil {
			return fmt.Errorf("failed to write string %s to %s: %s", name, path, err)
		}
		if err := writeU32(f, msg.index); err != nil {
			return fmt.Errorf("failed to index of %s to %s: %s", name, path, err)
		}
		if err := writeU32(f, msg.helpIndex); err != nil {
			return fmt.Errorf("failed to help index of %s to %s: %s", name, path, err)
		}
		if err := writeU32(f, len(msg.varIndices)); err != nil {
			return fmt.Errorf("failed to variable count of %s to %s: %s", name, path, err)
		}
		for i, index := range msg.varIndices {
			if err := writeU32(f, index); err != nil {
				return fmt.Errorf("failed to variable %d of %s to %s: %s", i, name, path, err)
			}
		}
	}

	return nil
}

func (s *Store) WriteText(path string, template *Store) error {
	return nil
}
