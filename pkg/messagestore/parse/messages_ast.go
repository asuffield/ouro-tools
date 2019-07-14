package parse

//go:generate bash -c "pigeon messages.peg | goimports > messages.go.tmp"
//go:generate mv messages.go.tmp messages.go

import (
	"fmt"
	"strings"
)

type MessageFile struct {
	Lines    []Line
	TypeFile bool
}

type MessageData interface {
	Message(string) string
	MessageVarTypes(string) map[string]string
}

type Line interface {
	Format() string
	FormatWith(MessageData) string
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

// Reconstruct an AST from some data, when we don't have a template
func NewFromData(header string, messages []Message, types []Type) *MessageFile {
	lines := []Line{&comment{comment: header}}
	for _, m := range messages {
		lines = append(lines, &message{id: m.Id, gap: " "})
	}
	// There is no valid format for this
	//for _, t := range types {
	//	vars := []*varType{}
	//	for _, v := range t.Vars {
	//		vars = append(vars, &varType{name: v.Name, ty: v.Ty})
	//	}
	//	lines = append(lines, &messageType{id: t.Id, varTypes: vars})
	//}
	return &MessageFile{Lines: lines}
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

func newMessageFile(messages interface{}, types bool) (*MessageFile, error) {
	ms := []Line{}
	for _, m := range toIfaceSlice(messages) {
		ms = append(ms, m.(Line))
	}
	return &MessageFile{ms, types}, nil
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
		return fmt.Sprintf("import %s %s\r\n", i.MessageFile, i.TypeFile)
	} else {
		return fmt.Sprintf("import %s\r\n", i.MessageFile)
	}
}

func (i *Import) FormatWith(_ MessageData) string {
	return i.Format()
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

func (b *blank) FormatWith(_ MessageData) string {
	return b.Format()
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

func (c *comment) FormatWith(_ MessageData) string {
	return c.Format()
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

func (m *message) FormatWith(d MessageData) string {
	content := d.Message(m.id)
	if content == "" {
		return fmt.Sprintf("// %s", m.Format())
	}
	if m.message.multiline || strings.ContainsAny(content, "\r\n") {
		return fmt.Sprintf("\"%s\"%s<<%s>>%s\r\n", m.id, m.gap, content, m.junk)
	} else {
		return fmt.Sprintf("\"%s\"%s\"%s\"%s\r\n", m.id, m.gap, content, m.junk)
	}
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
	if s.p2 != "" || s.ty != "" {
		return fmt.Sprintf("%s{%s%s,%s%s}", s.junk, s.p1, s.name, s.p2, s.ty)
	} else {
		return fmt.Sprintf("%s{%s%s}", s.junk, s.p1, s.name)
	}
}

func (s *varType) FormatWith(ty string) string {
	if s.p2 != "" || ty != "" {
		return fmt.Sprintf("%s{%s%s,%s%s}", s.junk, s.p1, s.name, s.p2, ty)
	} else {
		return fmt.Sprintf("%s{%s%s}", s.junk, s.p1, s.name)
	}
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

func (m *messageType) Format() string {
	var str strings.Builder
	str.WriteString(fmt.Sprintf("\"%s\"", m.id))
	for _, t := range m.varTypes {
		str.WriteString(t.Format())
	}
	str.WriteString(m.junk)
	str.WriteString("\r\n")
	return str.String()
}

func (m *messageType) FormatWith(d MessageData) string {
	tys := d.MessageVarTypes(m.id)
	if len(tys) == 0 {
		return fmt.Sprintf("// %s", m.Format())
	}

	var str strings.Builder
	str.WriteString(fmt.Sprintf("\"%s\"", m.id))
	for _, t := range m.varTypes {
		if ty, ok := tys[t.name]; ok {
			str.WriteString(t.FormatWith(ty))
		}
	}
	str.WriteString(m.junk)
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
