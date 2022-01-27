package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wader/fq/format/kaitai/ksexpr"
)

// used to make json numbers into int/float64
func normalizeNumbers(a any) any {
	switch a := a.(type) {
	case map[string]any:
		for k, v := range a {
			a[k] = normalizeNumbers(v)
		}
		return a
	case []any:
		for k, v := range a {
			a[k] = normalizeNumbers(v)
		}
		return a
	case json.Number:
		if strings.Contains(a.String(), ".") {
			f, _ := a.Float64()
			return f
		}
		// TODO: truncates
		i, _ := a.Int64()
		return int(i)
	default:
		return a
	}
}

type JSONVar struct {
	V any
}

func (v *JSONVar) String() string {
	if v.V != nil {
		b, _ := json.Marshal(v.V)
		return string(b)
	}
	return ""
}

func (v *JSONVar) Set(s string) error {
	jd := json.NewDecoder(bytes.NewBufferString(s))
	jd.UseNumber()
	if err := jd.Decode(&v.V); err != nil {
		return err
	}
	v.V = normalizeNumbers(v.V)
	return nil
}

func lookup(input any, ns string, name string) (any, error) {
	switch ns {
	case "":
		switch i := input.(type) {
		case map[string]any:
			v := i[name]
			return v, nil
		}
	default:
		return "with_namespace", nil
	}

	return nil, fmt.Errorf("failed to lookup ident %s", name)
}

func main() {
	lexFlag := flag.Bool("lex", false, "Lex expression")
	parseFlag := flag.Bool("parse", false, "Parse expression")
	evalFlag := flag.Bool("eval", false, "Eval expression")
	var inputValue JSONVar
	flag.Var(&inputValue, "input", "Input JSON")

	flag.Parse()
	exprStr := flag.Arg(0)

	// eval by default if no other flag is given
	*evalFlag = *evalFlag || (!*lexFlag && !*parseFlag)

	if exprStr == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTS] EXPR\n", os.Args[0])
		flag.Usage()
		os.Exit(1)
	}

	if *lexFlag {
		for _, t := range ksexpr.Lex(exprStr) {
			fmt.Fprintf(
				os.Stderr,
				"%s %s (%d-%d) %v\n",
				t.Name, t.Token.Str, t.Token.Span.Start, t.Token.Span.Stop, t.Err,
			)
		}
	}
	if *parseFlag || *evalFlag {
		r, err := ksexpr.Parse(exprStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse: %s\n", err)
			os.Exit(1)
		}

		if *parseFlag {
			je := json.NewEncoder(os.Stderr)
			je.SetIndent("", "  ")
			if err := je.Encode(&r); err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
		}
		if *evalFlag {
			v, err := r.Eval(ksexpr.ToValue(inputValue.V))
			if err != nil {
				fmt.Fprintf(os.Stderr, "eval: %s\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stdout, "%#v\n", v)
		}
	}
}
