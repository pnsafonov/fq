package kaitai

// TODO: add _meta?
// TODO: values tree somehow, ksexpr interface?
// TODO: prompt format name somehow?
// TODO: dump title/description?
// TODO: meta, endian, switch-on
// TODO: prompt, per format name?
// TODO: sizeof ceiled to bytes
// TODO: bitsizeof
// TODO: _index
// TODO: fq -o source=@format/kaitai/testdata/test.ksy -d kaitai d file
// TODO: fq -d format/kaitai/testdata/test.ksy d file

import (
	"fmt"
	"log"
	"strings"

	"github.com/wader/fq/format"
	"github.com/wader/fq/format/kaitai/ksexpr"
	"github.com/wader/fq/format/kaitai/schema"
	"github.com/wader/fq/format/kaitai/types"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
	"github.com/wader/fq/pkg/scalar"
)

func init() {
	interp.RegisterFormat(decode.Format{
		Name:         format.KAITAI,
		Description:  "Kaitai struct",
		DecodeFn:     kaitaiDecode,
		DefaultInArg: format.KaitaiIn{},
	})
}

func addStrNonEmpty(d *decode.D, name string, v string) {
	if v != "" {
		d.FieldValueStr(name, v)
	}
}

func decodeEndian(c decode.Endian, e types.Endianess) decode.Endian {
	switch e {
	case types.CurrentEndian:
		return c
	case types.LE:
		return decode.LittleEndian
	case types.BE:
		return decode.BigEndian
	default:
		panic("unreachable")
	}
}

type typeInstance struct {
	schemaType *schema.Type
	parent     *typeInstance
	root       *typeInstance

	d *decode.D

	instances map[string]any

	// TODO: used by array "_"
	last any

	seq    map[string]any
	repeat []any // TODO: hmm
}

func ksExprField(d *decode.D, name string, v any) {
	switch v := v.(type) {
	case ksexpr.Integer:
		d.FieldValueSint(name, int64(v))
	case ksexpr.BigInt:
		d.FieldValueBigInt(name, v.V)
	case ksexpr.Boolean:
		d.FieldValueBool(name, bool(v))
	case ksexpr.Float:
		d.FieldValueFlt(name, float64(v))
	case ksexpr.String:
		d.FieldValueStr(name, string(v))
	case ksexpr.Array:
		d.FieldArray(name, func(d *decode.D) {
			for _, ve := range v {
				ksExprField(d, name, ve)
			}
		})
	case ksexpr.Object:
		// TODO: not possible?
		d.FieldStruct(name, func(d *decode.D) {
			for k, ve := range v {
				ksExprField(d, k, ve)
			}
		})
	default:
		panic("unreachable")
	}
}

func (ti *typeInstance) resolveInstanceDo(t *schema.Type, name string) (any, error) {
	log.Printf("  resolveInstanceDo name: %#+v\n", name)
	switch {
	case t.Value != nil:
		log.Printf("  resolveInstanceDo t.Value: %#+v\n", t.Value)
		v := ti.mustEval(name, "value", t.Value)
		ksExprField(ti.d, name, v)
		return v, nil
	case t.Type != nil:
		if t.Pos == nil {
			ti.d.Fatalf("%s: instance with type without pos", name)
		}

		// TODO: int64
		pos := ti.mustEvalInt(name, "pos", t.Pos)

		st := *t
		st.ID = name

		log.Printf("  st: %#+v\n", st)

		log.Printf("  pos: %#+v\n", pos)
		log.Printf("  ti.d.BitsLeft()/8: %#+v\n", ti.d.BitsLeft()/8)
		// TODO: cleanup range mess
		var v any
		ti.d.RangeFn(int64(pos)*8, ti.d.BitsLeft()-int64(pos)*8, func(d *decode.D) {

			tti := &typeInstance{
				schemaType: &st,
				parent:     ti,
				root:       ti.root,
				d:          d,

				seq: map[string]any{},
			}
			// TODO: last?
			tti.decodeType()

		})

		return v, nil

	default:
		panic("unreachable")
	}
}

func (ti *typeInstance) resolveInstance(name string) (any, error) {
	tst := ti.schemaType

	log.Printf("resolveInstance name: %#+v\n", name)

	if ti.instances == nil {
		ti.instances = map[string]any{}
	}
	log.Printf("  ti.instances: %#+v\n", ti.instances)

	t, ok := tst.Instances[name]
	if !ok {
		log.Println("  NOT FOUND")
		log.Printf("   tst.Instances: %#+v\n", tst.Instances)
		return nil, fmt.Errorf("%s: instance not found", name)
	}
	// already resolved
	if v, ok := ti.instances[name]; ok {
		log.Printf("    cached v: %#+v\n", v)
		return v, nil
	}

	v, err := ti.resolveInstanceDo(t, name)
	if err != nil {
		return nil, err
	}

	ti.instances[name] = v

	log.Printf("    resolved to v: %#+v\n", v)

	return v, nil
}

func (ti *typeInstance) KSExprCall(ns string, name string, args []any) (any, error) {
	log.Printf("KSExprCall name: %#+v\n", name)

	if ns != "" {
		for p := ti; p != nil; p = p.parent {
			if e, ok := p.schemaType.Enums[ns]; ok {
				if v, ok := e.FromID[name]; ok {
					return v, nil
				}
				// TODO: error?
			}
		}
		return nil, fmt.Errorf("failed to lookup %s::%s", ns, name)
	}

	switch name {
	case "_":
		// log.Printf("ti: %#+v\n", ti)

		// log.Printf("ti.d.Value: %#+v\n", ti.d.Value)

		// log.Printf("ti.last: %#+v\n", ti.last)

		// dv := ti.d.FieldIndex(-1)
		// log.Printf("dv: %#+v\n", dv)
		// switch dvv := dv.V.(type) {
		// // TODO: mor
		// case *scalar.Uint:
		// 	return dvv.Actual, nil
		// case *scalar.Sint:
		// 	return dvv.Actual, nil
		// case *scalar.Str:
		// 	return dvv.Actual, nil
		// }

		if ti.last == nil {
			ti.d.Fatalf("no last")
		}

		log.Printf("  return ti.last: %#+v\n", ti.last)

		return ti.last, nil

	case "_io":
		// TODO: some io object?
		return ti, nil
	case "_parent":
		// TODO: no parent?
		return ti.parent, nil
	case "_root":
		log.Printf("  return ti.root: %#+v\n", ti.root)

		return ti.root, nil
	case "eof":
		log.Printf(" k.d.End(): %#+v\n", ti.d.End())
		return ti.d.End(), nil
	default:

		log.Printf("ti: %#+v\n", ti)

		log.Printf("  ti.seq: %#+v\n", ti.seq)

		// TODO: KSExprIndex

		if t, ok := ti.seq[name]; ok {
			log.Printf("  found as seq name: %#+v\n", name)
			return t, nil
		}

		dv := ti.d.FieldGet(name)
		if dv != nil {
			log.Printf("  found as field dv.V: %#+v\n", dv.V)
			switch dvv := dv.V.(type) {
			// TODO: mor
			case *scalar.Uint:
				return dvv.Actual, nil
			case *scalar.Sint:
				return dvv.Actual, nil
			case *scalar.Str:
				return dvv.Actual, nil
			}
		}

		// TODO: detect loop
		// TODO: cache?
		// TODO: always in seq?
		// TODO: in seq and in instace? look in parents?
		// TODO: if parent == nil?
		return ti.resolveInstance(name)
	}
	panic("fixme")
}

func (ti *typeInstance) eval(name string, exprSource string, e *schema.Expr) (any, error) {
	v, err := e.KSExpr.Eval(ti)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %s: %s", name, exprSource, e.Str, err)
	}
	return v, nil
}

func (ti *typeInstance) mustEval(name string, exprSource string, e *schema.Expr) any {
	v, err := ti.eval(name, exprSource, e)
	if err != nil {
		ti.d.Fatalf("%s", err)
	}
	return v
}

func (ti *typeInstance) evalInt(name string, exprSource string, e *schema.Expr) (int, error) {
	sv, err := ti.eval(name, exprSource, e)
	if err != nil {
		return 0, err
	}
	s, ok := ksexpr.ToInt(sv)
	if !ok {
		return 0, fmt.Errorf("%s: %s: %s: did not evaluate to an integer: %v", name, exprSource, e.Str, s)
	}
	return s, nil
}

func (ti *typeInstance) mustEvalInt(name string, exprSource string, e *schema.Expr) int {
	s, err := ti.evalInt(name, exprSource, e)
	if err != nil {
		return 0
	}
	return s
}

func (ti *typeInstance) mustEvalBool(name string, exprSource string, e *schema.Expr) bool {
	s := ti.mustEval(name, exprSource, e)
	switch s := s.(type) {
	case ksexpr.Boolean:
		return bool(s)
	default:
		ti.d.Fatalf("%s: %s: did not evaluated to a boolean: %s", name, exprSource, s)
		panic("unreachable")
	}
}

type typeDecoderFn func(d *decode.D) any

func contentsByteSize(c []any) (int, error) {
	s := 0

	for _, e := range c {
		switch e := e.(type) {
		case string:
			s += len(e)
		case int:
			if e < 0 || e > 255 {
				return 0, fmt.Errorf("contents array has invalid non-byte integer: %d", e)
			}
			s++
		default:
			return 0, fmt.Errorf("contents array as invalid value: %v", e)
		}
	}

	return s, nil
}

func (ti *typeInstance) decodeType() any {
	tst := ti.schemaType

	typ := "bytes"

	if tst.Type != nil {
		tt := tst.Type

		switch {
		case tt.Value != nil:
			typ = *tt.Value
		case tt.SwitchOn != nil:
			// TODO: handle "_"
			// TODO: types int vs int64 etc, ksexpr helper?

			tv := ti.parent.mustEval(tst.ID, "switch-on", tt.SwitchOn)

			for _, ce := range tt.CasesExprs {
				kv := ti.mustEval(tst.ID, "case", &ce.Expr)

				// log.Printf("tv: %#+v\n", tv)
				// log.Printf("kv: %#+v\n", kv)

				if tv == kv {
					typ = ce.Value
					break
				}
			}
		default:
			ti.d.Fatalf("%s: invalid type or switch-on", tst.ID)
		}
	}

	log.Printf("decodeType: typ=%#+v\n", typ)

	if tt, ok := types.Types[typ]; ok {
		log.Printf("  tt: %#+v\n", tt)

		if tt.BitAlign != 0 {
			ti.d.BitEndianAlign()
		}

		switch tt.Encoding {
		case types.Bool:
			return ti.d.FieldBoolE(tst.ID, decodeEndian(ti.d.BitEndian, tt.Endian))

		case types.Bits,
			types.Unsigned:
			var mappers []scalar.UintMapper

			if tst.Enum != "" {
				log.Printf("tst.Enum: %#+v\n", tst.Enum)
				for p := ti; p != nil; p = p.parent {
					if enum, ok := p.schemaType.Enums[tst.Enum]; ok {
						log.Printf("   found enum: %#+v\n", enum)
						mappers = append(mappers, scalar.UintFn(func(s scalar.Uint) (scalar.Uint, error) {
							// log.Printf("s.Actual: %#+v\n", s.Actual)
							// log.Printf("enum.ToID: %#+v\n", enum.ToID)
							if v, ok := enum.ToID[ksexpr.ToValue(s.Actual)]; ok {
								s.Sym = v
							}
							return s, nil
						}))
					}
				}
			}

			e := ti.d.Endian
			if tt.Encoding == types.Bits {
				e = ti.d.BitEndian
			}

			return ti.d.FieldUE(tst.ID, tt.BitSize, decodeEndian(e, tt.Endian), mappers...)

		case types.Signed:
			var mappers []scalar.SintMapper

			if tst.Enum != "" {
				for p := ti; p != nil; p = p.parent {
					if enum, ok := p.schemaType.Enums[tst.Enum]; ok {
						mappers = append(mappers, scalar.SintFn(func(s scalar.Sint) (scalar.Sint, error) {
							// log.Printf("s.Actual: %#+v\n", s.Actual)
							// log.Printf("enum.ToID: %#+v\n", enum.ToID)
							if v, ok := enum.ToID[ksexpr.ToValue(s.Actual)]; ok {
								s.Sym = v
							}
							return s, nil
						}))
					}
				}
			}
			return ti.d.FieldSE(tst.ID, tt.BitSize, decodeEndian(ti.d.Endian, tt.Endian), mappers...)

		case types.Float:
			return ti.d.FieldFE(tst.ID, tt.BitSize, decodeEndian(ti.d.Endian, tt.Endian))

		case types.Bytes:
			switch {
			case tst.Size != nil:
				s := ti.mustEvalInt(tst.ID, "size", tst.Size)
				return ti.d.FieldRawLen(tst.ID, int64(s)*8)
			case tst.SizeEOS:
				// TODO: error not byte aligned?
				return ti.d.FieldRawLen(tst.ID, ti.d.BitsLeft())
			case tst.Contents != nil:
				switch c := tst.Contents.(type) {
				case []any:
					// TODO: move schema parse if constant?
					l, err := contentsByteSize(c)
					if err != nil {
						ti.d.Fatalf("%s: %s", tst.ID, err)
					}

					return ti.d.FieldRawLen(tst.ID, int64(l)*8)
				case string:
					l := len(c)

					return ti.d.FieldRawLen(tst.ID, int64(l)*8)
				default:
					panic("unreachable")
				}
			default:
				panic("unreachable")
			}

		case types.Str:
			// log.Printf("t: %#+v\n", t)

			s := 0
			switch {
			case tst.Size != nil:
				s = ti.mustEvalInt(tst.ID, "size", tst.Size)
			case tst.SizeEOS:
				// TODO: error if not byte aligned?
				s = int(ti.d.BitsLeft() / 8)

				log.Printf("  SIZE-EOS: %#+v\n", s)
			}

			return ti.d.FieldUTF8(tst.ID, s)

		case types.StrTerminated:
			// TODO: config
			// TODO: encoding
			return ti.d.FieldUTF8Null(tst.ID)

		default:
			panic("unreachable")
		}
	}

	for p := ti; p != nil; p = p.parent {
		if t, ok := p.schemaType.Types[typ]; ok {
			// TODO: type always seq?

			log.Printf("  SUBTYPE %s:\n", typ)

			log.Printf("     t: %#+v\n", t)
			log.Printf("     tst: %#+v\n", tst)

			ti.schemaType = t

			d := ti.d

			// TODO: refactor out to byteSize/bitsSize function?
			if tst.Size != nil {
				s := ti.parent.mustEvalInt(tst.ID, "size", tst.Size)
				d = d.FramedLen(int64(s) * 8)
				log.Printf("  FRAMED s: %#+v\n", s)
			}

			ti.parent.seq[tst.ID] = ti

			d.FieldStruct(tst.ID, func(d *decode.D) {
				ti.d = d
				ti.decodeSeq()
			})

			return ti
		}
	}

	ti.d.Errorf("can't find type %s", typ)
	panic("unreachable")
}

func (ti *typeInstance) decodeSeq() {
	tst := ti.schemaType

	// TODO: move

	if tst.Meta != nil {
		// ti.d.FieldStruct("_meta", func(d *decode.D) {
		// 	addStrNonEmpty(d, "id", tst.Meta.ID)
		// 	addStrNonEmpty(d, "title", tst.Meta.Title)
		// 	addStrNonEmpty(d, "endian", tst.Meta.Endian)
		// })

		if tst.Meta.Endian != nil {
			// TODO: switch-on
			ti.d.Endian = decodeEndian(ti.d.Endian, types.Endianess(*tst.Meta.Endian))
		}
		if tst.Meta.BitEndian != nil {
			// TODO: switch-on
			ti.d.BitEndian = decodeEndian(ti.d.Endian, types.Endianess(*tst.Meta.BitEndian))
		}
	}

	for _, t := range tst.Seq {
		log.Printf("decodeSeq t.ID: %#+v\n", t.ID)

		if t.If != nil {
			if !ti.mustEvalBool(t.ID, "if", t.If) {
				continue
			}
		}

		if t.Repeat != "" {
			// TODO: correct? own typeInstance?

			ti.d.FieldArray(t.ID, func(d *decode.D) {
				tti := &typeInstance{
					schemaType: t,
					parent:     ti,
					root:       ti.root,
					d:          d,

					seq: map[string]any{},
				}
				ti.seq[t.ID] = tti

				tti.decodeRepeat()
			})
		} else {
			tti := &typeInstance{
				schemaType: t,
				parent:     ti,
				root:       ti.root,
				d:          ti.d,

				seq: map[string]any{},
			}

			ti.last = tti.decodeType()
		}
	}

	log.Println("decodeSeq Instances")

	for id := range tst.Instances {
		log.Printf("  id: %#+v\n", id)
		if _, err := ti.resolveInstance(id); err != nil {
			ti.d.Fatalf("%s", err)
		}
	}
}

func (ti *typeInstance) decodeRepeat() {
	tst := ti.schemaType

	switch tst.Repeat {
	case "eos":
		log.Printf("  REPEAT-EOS:\n")

		for !ti.d.End() {
			tti := &typeInstance{
				schemaType: tst,
				parent:     ti,
				root:       ti.root,
				d:          ti.d,

				seq: map[string]any{},
			}

			ti.parent.last = tti.decodeType()

			// ti.repeat = append(ti.repeat, v)
		}
	case "until":
		if tst.RepeatUntil == nil {
			ti.d.Fatalf("%s: repeat: %s: without repeat-until", tst.ID, tst.Repeat)
		}

		log.Printf("  REPEAT-UTIL: n: %#+v\n", tst.RepeatUntil.Str)

		for {
			tti := &typeInstance{
				schemaType: tst,
				parent:     ti,
				root:       ti.root,
				d:          ti.d,

				seq: map[string]any{},
			}

			ti.parent.last = tti.decodeType()

			// TODO: skip .parent, no new instance for repeat?
			if ti.parent.mustEvalBool(tst.ID, "repeat-until", tst.RepeatUntil) {
				return
			}

			// ti.repeat = append(ti.repeat, v)
		}
	case "expr":
		if tst.RepeatExpr == nil {
			ti.d.Fatalf("%s: repeat: %s: without repeat-expr", tst.ID, tst.Repeat)
		}

		// TODO: skip .parent, no new instance for repeat?
		n := ti.parent.mustEvalInt(tst.ID, "repeat-expr", tst.RepeatExpr)
		log.Printf("  REPEAT-EXPR: n: %#+v\n", n)

		for i := 0; i < n; i++ {
			tti := &typeInstance{
				schemaType: tst,
				parent:     ti,
				root:       ti.root,
				d:          ti.d,

				seq: map[string]any{},
			}
			ti.parent.last = tti.decodeType()
		}
	default:
		// TODO: add verify in parser
		panic("unreachable")
	}
}

func kaitaiDecode(d *decode.D) any {
	var ki format.KaitaiIn
	if !d.ArgAs(&ki) {
		d.Fatalf("no source option")
	}

	t, err := schema.Parse(strings.NewReader(ki.Source))
	if err != nil {
		log.Fatalf("source: %v", err)
	}

	log.Printf("t: %#+v\n", t)

	ti := &typeInstance{
		schemaType: t,
		parent:     nil,
		d:          d,

		seq: map[string]any{},
	}
	ti.root = ti
	ti.decodeSeq()

	return nil
}
