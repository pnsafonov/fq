//go:generate sh -c "jq -nrf types.jq | gofmt > types_gen.go"

package types

type Encoding int

const (
	Bytes Encoding = iota
	Bool
	Bits
	Signed
	Unsigned
	Float
	Str
	StrTerminated
)

type Endianess int

const (
	CurrentEndian Endianess = iota
	LE
	BE
)

type Type struct {
	Encoding Encoding
	BitSize  int
	BitAlign int
	Endian   Endianess
}
