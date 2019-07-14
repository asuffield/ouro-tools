package parse

//go:generate bash -c "pigeon messages.peg | goimports > messages.go.tmp"
//go:generate mv messages.go.tmp messages.go

import (
	"fmt"
	"strings"
)

type MessageFile struct {
	Lines []Line
}

type Line interface {
	Format() string
	Messages() []Message
	Types() []Type
}

type Message struct {
	Id      string
	Content string
}

type Type struct {
	Id   string
	Vars []Var
}

type Var struct {
	Name, Ty string
}

func toIfaceSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	return v.([]interface{})
}

func toFlatString(v interface{}) string {
	if v == nil {
		return ""
	}

	var str strings.Builder
	for _, d := range v.([]interface{}) {
		str.Write(d.([]byte))
	}
	return str.String()
}

func newMessageFile(messages interface{}) (*MessageFile, error) {
	ms := []Line{}
	for _, m := range toIfaceSlice(messages) {
		ms = append(ms, m.(Line))
	}
	return &MessageFile{ms}, nil
}

type empty struct {
}

func (*empty) Messages() []Message {
	return []Message{}
}

func (*empty) Types() []Type {
	return []Type{}
}

type Import struct {
	empty
	MessageFile, TypeFile string
}

func (i *Import) Format() string {
	if i.TypeFile != "" {
		return fmt.Sprintf("import %s %s", i.MessageFile, i.TypeFile)
	} else {
		return fmt.Sprintf("import %s", i.MessageFile)
	}
}

func newImport(messageFile, typeFile string) (*Import, error) {
	return &Import{
		MessageFile: messageFile,
		TypeFile:    typeFile,
	}, nil
}

type blank struct {
	empty
}

func newBlank() (*blank, error) {
	return &blank{}, nil
}

func (*blank) Format() string {
	return "\r\n"
}

type comment struct {
	empty
	comment string
}

func newComment(c string) (*comment, error) {
	return &comment{comment: c}, nil
}

func (c *comment) Format() string {
	return fmt.Sprintf("%s\r\n", c.comment)
}

type message struct {
	id        string
	message   *msgString
	gap, junk string
}

type messageString interface {
	Content() string
}

func (m *message) Format() string {
	return fmt.Sprintf("\"%s\"%s%s%s\r\n", m.id, m.gap, m.message.Format(), m.junk)
}

func (m *message) Messages() []Message {
	var content string
	if m.message.multiline {
		// Remove the << and >>
		content = m.message.content[2 : len(m.message.content)-2]
	} else {
		// Remove the "" and replace all \" with "
		content = strings.Replace(m.message.content[1:len(m.message.content)-1], "\\\"", "\"", -1)
	}
	return []Message{
		Message{m.id, content},
	}
}

func (m *message) Types() []Type {
	return []Type{}
}

func newMessage(id, gap, m, junk interface{}) (*message, error) {
	return &message{
		id:      id.(string),
		message: m.(*msgString),
		gap:     toFlatString(gap),
		junk:    toFlatString(junk),
	}, nil
}

type msgString struct {
	content   string
	multiline bool
}

func (s *msgString) Format() string {
	return s.content
}

func newString(content string) (*msgString, error) {
	return &msgString{content, false}, nil
}

func newMultilineString(content string) (*msgString, error) {
	return &msgString{content, true}, nil
}

type varType struct {
	name, ty, junk, p1, p2 string
}

func (s *varType) Format() string {
	return fmt.Sprintf("%s{%s%s,%s%s}", s.junk, s.p1, s.name, s.p2, s.ty)
}

func newVarType(name, ty, junk, p1, p2 interface{}) (*varType, error) {
	return &varType{
		name: toFlatString(name),
		ty:   toFlatString(ty),
		junk: toFlatString(junk),
		p1:   toFlatString(p1),
		p2:   toFlatString(p2),
	}, nil
}

type messageType struct {
	id       string
	varTypes []*varType
	junk     string
}

func (s *messageType) Format() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("\"%s\"", s.id))
	for _, t := range s.varTypes {
		str.WriteString(t.Format())
	}
	str.WriteString(s.junk)
	str.WriteString("\r\n")
	return str.String()
}

func (m *messageType) Messages() []Message {
	return []Message{}
}

func (m *messageType) Types() []Type {
	vars := []Var{}
	for _, v := range m.varTypes {
		vars = append(vars, Var{v.name, v.ty})
	}
	return []Type{
		Type{m.id, vars},
	}
}

func newMessageType(id, varsI, junk interface{}) (*messageType, error) {
	vs := []*varType{}
	for _, v := range toIfaceSlice(varsI) {
		vs = append(vs, v.(*varType))
	}
	return &messageType{
		id:       id.(string),
		varTypes: vs,
		junk:     toFlatString(junk),
	}, nil
}
