package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
	"github.com/alecthomas/participle/lexer/ebnf"
	"github.com/sorribas/minilisp/libtccbins"
	"github.com/sorribas/tcc"
)

type Program struct {
	SExps []*SExp `{ @@ }`
}

type SExp struct {
	Id     string `  @Id`
	Number string `| @Number`
	List   []SExp `| "(" { @@ } ")"`
}

var lispLexer lexer.Definition = lexer.Must(ebnf.New(`
  	Id = (idchar) {idchar} .
		Whitespace = " " | "\t" | "\n" | "\r" .
		RParen = "(" .
		LParen = ")" .
		Number = ("0"…"9") | {"0"…"9"} .
		idchar = "a"…"z" | "A"…"Z" | "+" | "-" | "/" | "*" .
  `))

var parser *participle.Parser = participle.MustBuild(&Program{},
	participle.Lexer(lispLexer),
	participle.Elide("Whitespace"),
)

func main() {
	p := Program{}
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "error: no input files\n")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not open file: %s\n", err)
		os.Exit(2)
	}
	defer file.Close()

	err = parser.Parse(file, &p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(3)
	}
	cprogram := GenerateCode(p)
	err = compileCProgram(cprogram, os.Args[1][:strings.IndexByte(os.Args[1], '.')])
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(4)
	}
}

func compileCProgram(cprogram string, outfile string) error {
	cc := tcc.NewTcc()
	defer cc.Delete()
	cc.SetOutputType(tcc.OUTPUT_EXE)

	libfolder, err := tmplibfolder()
	if err != nil {
		return err
	}

	cc.SetLibPath(libfolder)
	cc.AddIncludePath(libfolder)

	err = cc.CompileString(cprogram)
	if err != nil {
		return err
	}
	cc.OutputFile(outfile)
	err = os.RemoveAll(libfolder)
	return nil
}

func tmplibfolder() (string, error) {
	var err error
	slash := string(os.PathSeparator)
	tmpDir, err := ioutil.TempDir("", "minilisp-tcclibs")

	libtcca, err := libtccbins.Asset("libtcc.a")
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(tmpDir+slash+"libtcc.a", libtcca, 0755)
	if err != nil {
		return "", err
	}

	libtcc1a, err := libtccbins.Asset("libtcc1.a")
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(tmpDir+slash+"libtcc1.a", libtcc1a, 0755)
	if err != nil {
		return "", err
	}

	libtcch, err := libtccbins.Asset("tcclib.h")
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(tmpDir+slash+"tcclib.h", libtcch, 0755)
	if err != nil {
		return "", err
	}

	return tmpDir, err
}
