package ksexpr

import (
	"fmt"
	"math"
	"math/big"
)

type zeroDivError struct {
	l, r any
}

func (z zeroDivError) Error() string {
	return fmt.Sprintf("division by zero: %s / %s ", z.l, z.r)
}

type PrefixFn func(v any) any

type PrefixOp int

func (op PrefixOp) String() string {
	if s, ok := prefixOpNames[op]; ok {
		return s
	}
	panic(fmt.Sprintf("invalid prefix op %d", op))
}

const (
	PrefixOpNeg = iota
	PrefixOpNot
)

var prefixOpNames = map[PrefixOp]string{
	PrefixOpNeg: "-",
	PrefixOpNot: "not",
}

var prefixOpFn = map[PrefixOp]PrefixFn{
	PrefixOpNeg: PrefixNeg,
	PrefixOpNot: PrefixNot,
}

type InfixFn func(l, r any) any

type InfixOp int

func (op InfixOp) String() string {
	if s, ok := infixOpNames[op]; ok {
		return s
	}
	panic(fmt.Sprintf("invalid infix op %d", op))
}

const (
	InfixOpAdd = iota
	InfixOpSub
	InfixOpDiv
	InfixOpMul
	InfixOpMod
	InfixOpLT
	InfixOpLTEQ
	InfixOpGT
	InfixOpGTEQ
	InfixOpEQ
	InfixOpNotEQ
	InfixOpBSL
	InfixOpBSR
	InfixOpBAnd
	InfixOpBOr
	InfixOpBXor
	InfixOpAnd
	InfixOpOr
)

var infixOpNames = map[InfixOp]string{
	InfixOpAdd:   "+",
	InfixOpSub:   "-",
	InfixOpDiv:   "/",
	InfixOpMul:   "*",
	InfixOpMod:   "%",
	InfixOpLT:    "<",
	InfixOpLTEQ:  "<=",
	InfixOpGT:    ">",
	InfixOpGTEQ:  ">=",
	InfixOpEQ:    "==",
	InfixOpNotEQ: "!=",
	InfixOpBSL:   "<<",
	InfixOpBSR:   ">>",
	InfixOpBAnd:  "&",
	InfixOpBOr:   "|",
	InfixOpBXor:  "^",
	InfixOpAnd:   "and",
	InfixOpOr:    "or",
}

var infixOpFn = map[InfixOp]InfixFn{
	InfixOpAdd:   InfixAdd,
	InfixOpSub:   InfixSub,
	InfixOpDiv:   InfixDiv,
	InfixOpMul:   InfixMul,
	InfixOpMod:   InfixMod,
	InfixOpLT:    InfixLT,
	InfixOpLTEQ:  InfixLTEQ,
	InfixOpGT:    InfixGT,
	InfixOpGTEQ:  InfixGTEQ,
	InfixOpEQ:    InfixEQ,
	InfixOpNotEQ: InfixNotEQ,
	InfixOpBSL:   InfixBSL,
	InfixOpBSR:   InfixBSR,
	InfixOpBAnd:  InfixBAnd,
	InfixOpBOr:   InfixBOr,
	InfixOpBXor:  InfixBXor,
	InfixOpAnd:   InfixAnd,
	InfixOpOr:    InfixOr,
}

func prefixTypeSwitch(
	v any,
	boolFn func(v Boolean) any,
	integerFn func(v Integer) any,
	floatFn func(v Float) any,
	bigIntFn func(v BigInt) any,
	stringFn func(v String) any,
	arrayFn func(v Array) any,
	fallbackFn func(v any) any,
) any {
	switch v := v.(type) {
	case Boolean:
		return boolFn(v)
	case Integer:
		return integerFn(v)
	case Float:
		return floatFn(v)
	case BigInt:
		return bigIntFn(v)
	case String:
		return stringFn(v)
	case Array:
		return arrayFn(v)
	default:
		return fallbackFn(v)
	}
}

func prefixInvalid[T any](op PrefixOp) func(v T) any {
	return func(v T) any {
		str := func(v any) string {
			if s, ok := v.(fmt.Stringer); ok {
				return s.String()
			}
			return fmt.Sprintf("%#v", v)
		}
		return fmt.Errorf("invalid operation %s %s", op, str(v))
	}
}

func PrefixNeg(v any) any {
	return prefixTypeSwitch(
		v,
		prefixInvalid[Boolean](PrefixOpNeg),
		func(v Integer) any { return Integer(-v) },
		func(v Float) any { return Float(-v) },
		func(v BigInt) any { return BigInt{new(big.Int).Neg(v.V)} },
		prefixInvalid[String](PrefixOpNeg),
		prefixInvalid[Array](PrefixOpNeg),
		prefixInvalid[any](PrefixOpNeg),
	)
}

func PrefixNot(v any) any {
	return prefixTypeSwitch(
		v,
		func(v Boolean) any { return Boolean(!v) },
		prefixInvalid[Integer](PrefixOpNot),
		prefixInvalid[Float](PrefixOpNot),
		prefixInvalid[BigInt](PrefixOpNot),
		prefixInvalid[String](PrefixOpNot),
		prefixInvalid[Array](PrefixOpNot),
		prefixInvalid[any](PrefixOpNot),
	)
}

func infixTypeSwitch(
	l any, r any,
	boolFn func(l, r Boolean) any,
	integerFn func(l, r Integer) any,
	floatFn func(l, r Float) any,
	bigIntFn func(l, r BigInt) any,
	stringFn func(l, r String) any,
	arrayFn func(l, r Array) any,
	fallbackFn func(l, r any) any,
) any {
	switch l := l.(type) {
	case Boolean:
		switch r := r.(type) {
		case Boolean:
			return boolFn(l, r)
		}
	case Integer:
		switch r := r.(type) {
		case Integer:
			return integerFn(l, r)
		case Float:
			return floatFn(Float(l), r)
		case BigInt:
			return bigIntFn(NewBigIntFromInteger(l), r)
		}
	case Float:
		switch r := r.(type) {
		case Integer:
			return floatFn(l, Float(r))
		case Float:
			return floatFn(l, Float(r))
		case BigInt:
			return floatFn(l, NewFloatFromBigInt(r))
		}
	case BigInt:
		switch r := r.(type) {
		case Integer:
			return bigIntFn(l, NewBigIntFromInteger(r))
		case Float:
			return floatFn(NewFloatFromBigInt(l), r)
		case BigInt:
			return bigIntFn(l, r)
		}
	case String:
		switch r := r.(type) {
		case String:
			return stringFn(l, r)
		}
	case Array:
		switch r := r.(type) {
		case Array:
			return arrayFn(l, r)
		}
	}
	return fallbackFn(l, r)
}

func infixInvalid[T any](op InfixOp) func(l, r T) any {
	return func(l, r T) any {
		str := func(v any) string {
			if s, ok := v.(fmt.Stringer); ok {
				return s.String()
			}
			return fmt.Sprintf("%#v", v)
		}
		return fmt.Errorf("invalid operation %s %s %s", str(l), op, str(r))
	}
}

func InfixAdd(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpAdd),
		func(l, r Integer) any { return Integer(l + r) }, // TODO: overflow
		func(l, r Float) any { return floatToNumber(float64(l) + float64(r)) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Add(l.V, r.V)} },
		func(l, r String) any { return String(l + r) },
		infixInvalid[Array](InfixOpAdd),
		infixInvalid[any](InfixOpAdd),
	)
}

func InfixSub(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpSub),
		func(l, r Integer) any { return Integer(l - r) }, // TODO: overflow
		func(l, r Float) any { return floatToNumber(float64(l) - float64(r)) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Sub(l.V, r.V)} },
		infixInvalid[String](InfixOpSub),
		infixInvalid[Array](InfixOpSub),
		infixInvalid[any](InfixOpSub),
	)
}

func InfixMul(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpMul),
		func(l, r Integer) any { return Integer(l * r) }, // TODO: overflow
		func(l, r Float) any { return floatToNumber(float64(l) * float64(r)) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Mul(l.V, r.V)} },
		infixInvalid[String](InfixOpMul),
		infixInvalid[Array](InfixOpMul),
		infixInvalid[any](InfixOpMul),
	)
}

func InfixDiv(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpDiv),
		func(l, r Integer) any {
			if r == 0 {
				if l == 0 {
					return math.NaN()
				}
				return zeroDivError{l, r}
			}
			return Integer(l / r)
		},
		func(l, r Float) any {
			if r == 0.0 {
				if l == 0.0 {
					return math.NaN()
				}
				return zeroDivError{l, r}
			}
			return floatToNumber(float64(l) / float64(r))
		},
		func(l, r BigInt) any {
			if bigIntIsZero(r.V) {
				if bigIntIsZero(l.V) {
					return math.NaN()
				}
				return zeroDivError{l, r}
			}
			d, m := new(big.Int).DivMod(l.V, r.V, new(big.Int))
			if bigIntIsZero(m) {
				return BigInt{d}
			}
			return floatToNumber(l.Float() / r.Float())
		},
		infixInvalid[String](InfixOpDiv),
		infixInvalid[Array](InfixOpDiv),
		infixInvalid[any](InfixOpDiv),
	)
}

func InfixMod(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpDiv),
		func(l, r Integer) any {
			if r == 0 {
				return zeroDivError{l, r}
			}
			return Integer(l % r)

		},
		func(l, r Float) any {
			if r == 0 {
				return zeroDivError{l, r}
			}
			return floatToNumber(math.Mod(float64(l), float64(r)))
		},
		func(l, r BigInt) any {
			if bigIntIsZero(r.V) {
				return zeroDivError{l, r}
			}
			return BigInt{new(big.Int).Rem(l.V, r.V)}
		},
		infixInvalid[String](InfixOpDiv),
		infixInvalid[Array](InfixOpDiv),
		infixInvalid[any](InfixOpDiv),
	)
}

func InfixLT(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpLT),
		func(l, r Integer) any { return Boolean(l < r) },
		func(l, r Float) any { return Boolean(l < r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) < 0) },
		func(l, r String) any { return Boolean(l < r) },
		infixInvalid[Array](InfixOpLT),
		infixInvalid[any](InfixOpLT),
	)
}

func InfixLTEQ(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpLTEQ),
		func(l, r Integer) any { return Boolean(l <= r) },
		func(l, r Float) any { return Boolean(l <= r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) <= 0) },
		func(l, r String) any { return Boolean(l <= r) },
		infixInvalid[Array](InfixOpLTEQ),
		infixInvalid[any](InfixOpLTEQ),
	)
}

func InfixGT(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpGT),
		func(l, r Integer) any { return Boolean(l > r) },
		func(l, r Float) any { return Boolean(l > r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) > 0) },
		func(l, r String) any { return Boolean(l > r) },
		infixInvalid[Array](InfixOpGT),
		infixInvalid[any](InfixOpGT),
	)
}

func InfixGTEQ(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpGTEQ),
		func(l, r Integer) any { return Boolean(l >= r) },
		func(l, r Float) any { return Boolean(l >= r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) >= 0) },
		func(l, r String) any { return Boolean(l >= r) },
		infixInvalid[Array](InfixOpGTEQ),
		infixInvalid[any](InfixOpGTEQ),
	)
}

func InfixEQ(l, r any) any {
	return infixTypeSwitch(
		l, r,
		func(l, r Boolean) any { return Boolean(l == r) },
		func(l, r Integer) any { return Boolean(l == r) },
		func(l, r Float) any { return Boolean(l == r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) == 0) },
		func(l, r String) any { return Boolean(l == r) },
		func(l, r Array) any {
			if len(l) != len(r) {
				return Boolean(false)
			}
			for i := range l {
				v := InfixEQ(l[i], r[i])
				if err, ok := v.(error); ok {
					return fmt.Errorf("at index %d: %w", i, err)
				}
				if b, ok := v.(Boolean); !ok {
					panic("element eq eval not a bool")
				} else if !b {
					return Boolean(false)
				}
			}

			return Boolean(true)
		},
		infixInvalid[any](InfixOpEQ),
	)
}

func InfixNotEQ(l, r any) any {
	return infixTypeSwitch(
		l, r,
		func(l, r Boolean) any { return Boolean(l != r) },
		func(l, r Integer) any { return Boolean(l != r) },
		func(l, r Float) any { return Boolean(l != r) },
		func(l, r BigInt) any { return Boolean(l.V.Cmp(r.V) != 0) },
		func(l, r String) any { return Boolean(l != r) },
		func(l, r Array) any {
			v := InfixEQ(l, r)
			if err, ok := v.(error); ok {
				return err
			}
			b, ok := v.(Boolean)
			if !ok {
				panic("neq eval not a bool")
			}
			return Boolean(!b)
		},
		infixInvalid[any](InfixOpNotEQ),
	)
}

func InfixBSL(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpBSL),
		func(l, r Integer) any {
			if v := l << r; v>>r == l {
				return Integer(v)
			}
			return BigInt{new(big.Int).Lsh(big.NewInt(int64(l)), uint(r))}
		},
		func(l, r Float) any {
			li := l.Int()
			ri := r.Int()
			if v := li << ri; v>>ri == li {
				return Integer(v)
			}
			return BigInt{new(big.Int).Lsh(big.NewInt(int64(li)), uint(ri))}
		},
		func(l, r BigInt) any { return BigInt{new(big.Int).Lsh(l.V, uint(r.Int()))} },
		infixInvalid[String](InfixOpBSL),
		infixInvalid[Array](InfixOpBSL),
		infixInvalid[any](InfixOpBSL),
	)
}

func InfixBSR(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpBSR),
		func(l, r Integer) any { return Integer(l >> r) },
		func(l, r Float) any { return Integer(l.Int() >> r.Int()) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Rsh(l.V, uint(r.Int()))} },
		infixInvalid[String](InfixOpBSR),
		infixInvalid[Array](InfixOpBSR),
		infixInvalid[any](InfixOpBSR),
	)
}

func InfixBOr(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpBOr),
		func(l, r Integer) any { return Integer(l | r) },
		func(l, r Float) any { return Integer(l.Int() | r.Int()) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Or(l.V, r.V)} },
		infixInvalid[String](InfixOpBOr),
		infixInvalid[Array](InfixOpBOr),
		infixInvalid[any](InfixOpBOr),
	)
}

func InfixBAnd(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpBAnd),
		func(l, r Integer) any { return Integer(l & r) },
		func(l, r Float) any { return Integer(l.Int() & r.Int()) },
		func(l, r BigInt) any { return BigInt{new(big.Int).And(l.V, r.V)} },
		infixInvalid[String](InfixOpBAnd),
		infixInvalid[Array](InfixOpBAnd),
		infixInvalid[any](InfixOpBAnd),
	)
}

func InfixBXor(l, r any) any {
	return infixTypeSwitch(
		l, r,
		infixInvalid[Boolean](InfixOpBXor),
		func(l, r Integer) any { return Integer(l ^ r) },
		func(l, r Float) any { return Integer(l.Int() ^ r.Int()) },
		func(l, r BigInt) any { return BigInt{new(big.Int).Xor(l.V, r.V)} },
		infixInvalid[String](InfixOpBXor),
		infixInvalid[Array](InfixOpBXor),
		infixInvalid[any](InfixOpBXor),
	)
}

func InfixAnd(l, r any) any {
	return infixTypeSwitch(
		l, r,
		func(l, r Boolean) any { return Boolean(l && r) },
		infixInvalid[Integer](InfixOpAnd),
		infixInvalid[Float](InfixOpAnd),
		infixInvalid[BigInt](InfixOpAnd),
		infixInvalid[String](InfixOpAnd),
		infixInvalid[Array](InfixOpAnd),
		infixInvalid[any](InfixOpAnd),
	)
}

func InfixOr(l, r any) any {
	return infixTypeSwitch(
		l, r,
		func(l, r Boolean) any { return Boolean(l || r) },
		infixInvalid[Integer](InfixOpOr),
		infixInvalid[Float](InfixOpOr),
		infixInvalid[BigInt](InfixOpOr),
		infixInvalid[String](InfixOpOr),
		infixInvalid[Array](InfixOpOr),
		infixInvalid[any](InfixOpOr),
	)
}
