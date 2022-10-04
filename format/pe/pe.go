package pe

// https://osandamalith.com/2020/07/19/exploring-the-ms-dos-stub/

import (
	"github.com/wader/fq/format"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
)

// TODO: probe?
// TODO: not pe_ prefix for format names?

var peMSDosStubGroup decode.Group
var peCOFFGroup decode.Group

func init() {
	interp.RegisterFormat(
		format.Pe,
		&decode.Format{
			Description: "Portable Executable",
			Groups:      []*decode.Group{format.Probe},
			Dependencies: []decode.Dependency{
				{Groups: []*decode.Group{format.PeMsdosStub}, Out: &peMSDosStubGroup},
				{Groups: []*decode.Group{format.PeCoff}, Out: &peCOFFGroup},
			},
			DecodeFn: peDecode,
		})
}

func peDecode(d *decode.D) any {

	d.FieldFormat("ms_dos_stub", &peMSDosStubGroup, nil)
	d.FieldFormat("coff", &peCOFFGroup, nil)

	return nil
}
