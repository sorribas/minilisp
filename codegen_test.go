package main

import (
	"fmt"
	"testing"

	libtcc "github.com/sorribas/tcc"
)

func TestBasic(t *testing.T) {
	programString := `
	  (def a 3)
	  (printval a)
	`

	p := Program{}
	parser.ParseString(programString, &p)
	cprogram := GenerateCode(p)
	//t.Log(cprogram)
	runCProgram(cprogram, t)
}

func TestClosure(t *testing.T) {
	programString := `
	  (def a (fun(x) 
			(fun (y) 
				(+ x y))))
		(def b (a 3))
		(printval (b 2))
	`

	// (a 3) -> a(3)

	p := Program{}
	parser.ParseString(programString, &p)
	cprogram := GenerateCode(p)
	//t.Log("PROG", cprogram)
	runCProgram(cprogram, t)
}

func TestClosureMoreVars(t *testing.T) {
	programString := `
	  (def a (fun(x y) 
			(fun (z) 
				(+ x (+ y z)))))
		(def b (a 3 2))
		(printval (b 2))
		(printval (b 4))
	`

	p := Program{}
	parser.ParseString(programString, &p)
	cprogram := GenerateCode(p)
	// t.Log("PROG", cprogram)
	runCProgram(cprogram, t)
}

func TestFirstClassFunction(t *testing.T) {
	programString := `
	  (def a (fun(x)
			(+ x 1)))
		(def b (fun(x f)
			(+ 1 (f x))))
		(printval (b 2 a))
	`

	p := Program{}
	err := parser.ParseString(programString, &p)
	fmt.Println("err", err)
	cprogram := GenerateCode(p)
	//t.Log("PROG", cprogram)
	runCProgram(cprogram, t)
}

func runCProgram(cprogram string, t *testing.T) {
	tcc := libtcc.NewTcc()
	defer tcc.Delete()
	tcc.SetOutputType(1)

	err := tcc.CompileString(cprogram)
	if err != nil {
		t.Fatalf("Should compile. %s\n\n%s", cprogram, err)
	}
	// run the program
	i, err := tcc.Run([]string{})
	if i != 0 {
		t.Fatalf("Should return 0, got %d", i)
	}
	if err != nil {
		t.Fatalf("Shouldn't error at running")
	}
}
