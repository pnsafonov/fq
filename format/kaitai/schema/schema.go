//nolint:tagliatelle
package schema

// TODO: error line number? in error via Node?

import (
	"fmt"
	"io"

	"github.com/wader/fq/format/kaitai/ksexpr"
	"github.com/wader/fq/format/kaitai/types"
	"gopkg.in/yaml.v3"
)

type ValueError struct {
	Node *yaml.Node
	Err  error
}

func (v ValueError) Unwrap() error { return v.Err }

func (v ValueError) Error() string {
	return fmt.Sprintf("%d: %s", v.Node.Line, v.Err)
}

func valueErrorf(n *yaml.Node, format string, a ...any) ValueError {
	return ValueError{
		Node: n,
		Err:  fmt.Errorf(format, a...),
	}
}

// meta:
//   id: ...
//   endian: le | be
//   bit-endian: le | be              # for b#(|be|le)
//   endian:
//     switch-on:
//     cases:
//       <expr>: le | be              # can use _ for default
//
// seq:
//   - id: <name>
//
//     type: u1 | str | <type> | ...  # builtin scalar type or <type> (seq)
//     type:
//       switch-in <expr>             # switch on cases using expr
//       cases:
//          <expr>: <type>            # can use _ for default
//
//     size: <expr>                      # size many bytes (even for UTF-16 etc)
//     size-eos: <bool>                  # true is size if rest of stream (not an expression)
//     contents: string | [1,0x1,string] # read content size of bytes and validate (constant)
//
//     enum: <string>                 # enum mapping (not an expression)
//
//     if: <expr>                     # should be skipped?
//
//     repeat: expr | until           # id is an array of type
//     repeat-expr: <expr>            # number of times
//     repeat-until: <expr>           # until expr is true
//
// types:
//   <type>:
//     seq:
//       - id: <name>
//         ...
//
// enums:
//   <name>:
//     0: <name>
//     0:
//       - id: name>
//         doc: <doc>
//
// instances:
//   <id>:
//      value: <expr>
//   <id>:
//      pos: <expr
//      type: <string>  # TODO: switch-on?

type Expr struct {
	Str    string
	KSExpr ksexpr.Node
}

func (e *Expr) UnmarshalYAML(value *yaml.Node) error {
	if err := value.Decode(&e.Str); err != nil {
		return err
	}

	ke, err := ksexpr.Parse(e.Str)
	if err != nil {
		return fmt.Errorf("failed to parse '%s': %w", e.Str, err)
	}
	e.KSExpr = ke

	return nil
}

type Endian types.Endianess

func (e *Endian) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	switch s {
	case "le":
		(*e) = Endian(types.LE)
		return nil
	case "be":
		(*e) = Endian(types.BE)
		return nil
	default:
		return valueErrorf(value, "unknown endian %q", s)
	}
}

type CaseExpr[T any] struct {
	Expr  Expr
	Value T
}

type ValueOrSwitch[T any] struct {
	Value    *T
	SwitchOn *Expr        `yaml:"switch-on"`
	Cases    map[string]T `yaml:"cases"`

	CasesExprs []CaseExpr[T]
}

func (t *ValueOrSwitch[T]) UnmarshalYAML(value *yaml.Node) error {
	type tt ValueOrSwitch[T]
	var tv tt

	if err := value.Decode(&t.Value); err == nil {
		return nil
	} else if err := value.Decode(&tv); err == nil {
		(*t) = ValueOrSwitch[T](tv)

		for es, v := range t.Cases {
			e, err := ksexpr.Parse(es)
			if err != nil {
				return fmt.Errorf("failed to parse case expr: %w", err)
			}
			t.CasesExprs = append(t.CasesExprs, CaseExpr[T]{
				Expr:  Expr{KSExpr: e, Str: es},
				Value: v,
			})
		}

		return nil
	}

	return fmt.Errorf("failed to parse type as type name or switch-on")
}

type Meta struct {
	ID        string  `yaml:"id"`
	Title     string  `yaml:"title"`
	Endian    *Endian `yaml:"endian"`
	BitEndian *Endian `yaml:"bit-endian"`
}

type EnumEntryExpr struct {
	Expr Expr
	ID   string
}

type Enum struct {
	ToID   map[any]string
	FromID map[string]any
}

func (e *Enum) UnmarshalYAML(value *yaml.Node) error {
	var em map[string]EnumEntry
	if err := value.Decode(&em); err != nil {
		return err
	}

	e.ToID = map[any]string{}
	e.FromID = map[string]any{}

	for es, v := range em {
		en, err := ksexpr.Parse(es)
		if err != nil {
			return fmt.Errorf("failed to parse enum expr: %w", err)
		}
		ev, err := en.Eval(0)
		if err != nil {
			return fmt.Errorf("failed to eval enum expr: %w", err)
		}

		// TODO: types
		e.ToID[ev] = v.ID
		e.FromID[v.ID] = ev
	}

	return nil
}

type EnumEntry struct {
	ID string `yaml:"id"`
}

func (e *EnumEntry) UnmarshalYAML(value *yaml.Node) error {
	type et EnumEntry
	var ev et

	if err := value.Decode(&e.ID); err == nil {
		return nil
	} else if err := value.Decode(&ev); err == nil {
		// TODO: fix this
		(*e) = EnumEntry(ev)
		return nil
	}
	return fmt.Errorf("failed to parse enum entry as string or id/doc")
}

type Type struct {
	Meta *Meta                  `yaml:"meta"`
	ID   string                 `yaml:"id"`
	Type *ValueOrSwitch[string] `yaml:"type"`

	Size     *Expr `yaml:"size"`
	SizeEOS  bool  `yaml:"size-eos"`
	Contents any   `yaml:"contents"`

	Repeat      string `yaml:"repeat"`
	RepeatExpr  *Expr  `yaml:"repeat-expr"`
	RepeatUntil *Expr  `yaml:"repeat-until"`

	Enum string `yaml:"enum"`

	If *Expr `yaml:"if"`

	// instance
	Value *Expr `yaml:"value"`
	Pos   *Expr `yaml:"pos"`

	Seq       []*Type          `yaml:"seq"`
	Types     map[string]*Type `yaml:"types"`
	Enums     map[string]Enum  `yaml:"enums"`
	Instances map[string]*Type `yaml:"instances"`
}

func Parse(r io.Reader) (*Type, error) {
	t := &Type{}
	err := yaml.NewDecoder(r).Decode(t)
	if err != nil {
		return nil, err
	}

	return t, nil
}
