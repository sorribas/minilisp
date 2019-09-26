package main

import (
	"errors"
	"strconv"
	"strings"
)

func GenerateCode(p Program) string {
	cg := codeGenerator{}
	cg.program = p
	cg.globalContext.name = "global"
	cg.globalContext.variables = map[string]bool{}
	cg.globalContext.parent = nil
	cg.currentContext = &cg.globalContext
	return cg.generate()
}

type codeGenerator struct {
	program               Program
	globalContext         context
	mainCode              strings.Builder
	functionsCode         strings.Builder
	structsCode           strings.Builder
	definitionsCode       strings.Builder
	currentContext        *context
	currentFunctionNumber int
}

type context struct {
	parent    *context
	name      string
	variables map[string]bool
}

var runtime string = `
extern void* malloc(unsigned long long);
extern int printf ( const char * format, ... );

typedef enum valuetype {
  NUMBER,
  FUNCTION
} valueType;

typedef struct svalue (*lambdafn)();

typedef struct _lambda {
	void* ctx;
	lambdafn fun;
} lambda;

typedef union uuvalue {
  int number;
	struct _lambda *fun;
} uvalue;

typedef struct svalue {
  valueType type;
  uvalue actualValue;
} value;

struct emptycontext {
	void *parent;
};

/* end types */

value make_number(int n) {
	value* val = malloc(sizeof(value));
	val->actualValue.number = n;
	return *val;
}

value make_lambda(lambdafn fn, void* ctx) {
	lambda* l = (lambda*)malloc(sizeof(lambda));
	l->fun = fn;
	l->ctx = ctx;
	value* val = malloc(sizeof(value));
	val->actualValue.fun = l;
	return *val;
}

void* make_context(unsigned long long size, void* parent) {
	struct emptycontext* ctx = (struct emptycontext*)malloc(size);
	ctx->parent = parent;
	return ctx;
}

value plus(value a, value b) {
	return make_number(a.actualValue.number + b.actualValue.number);
}

void printval(value v) {
	printf("%d\n", v.actualValue.number);
}
`

func (cg *codeGenerator) generate() string {
	// Generate context
	for _, sexp := range cg.program.SExps {
		if len(sexp.List) > 2 && sexp.List[0].Id == "def" {
			if sexp.List[1].Id == "" {
				// ERROR expected identifier
			}
			cg.globalContext.variables[sexp.List[1].Id] = true
		}
	}

	for _, sexp := range cg.program.SExps {
		if sexp.List[0].Id == "def" {
			defcode, err := cg.generateDefCode(*sexp)
			if err != nil {
				// ERROR
			}
			cg.mainCode.WriteString(defcode + ";\n")
		} else {
			sexpcode, err := cg.generateSexpCode(*sexp)
			if err != nil {
				// ERROR
			}
			cg.mainCode.WriteString(sexpcode + ";\n")
		}
	}

	return runtime + "\n\n" +
		cg.definitionsCode.String() + "\n\n" +
		cg.structsCode.String() + "\n\n" +
		cg.functionsCode.String() + "\n\n" +
		cg.globalContextGen() +
		"int main() {\nstruct globalContextStruct c;\nstruct globalContextStruct *ctx = &c;\n" +
		cg.mainCode.String() +
		"\n return 0;\n}"
}

func (cg *codeGenerator) globalContextGen() string {
	var bldr strings.Builder
	bldr.WriteString("struct globalContextStruct {\n")
	for varName, _ := range cg.globalContext.variables {
		bldr.WriteString("  value " + varName + ";\n")
	}
	bldr.WriteString("};\n")
	return bldr.String()
}

func (cg *codeGenerator) generateDefCode(sexp SExp) (string, error) {
	if len(sexp.List) < 3 {
		return "", errors.New("Incomplete definition")
	}

	if sexp.List[1].Id == "" {
		return "", errors.New("Expected identifier after def")
	}

	valueString, err := cg.generateSexpCode(sexp.List[2])
	if err != nil {
		return "", err
	}
	return "ctx->" + sexp.List[1].Id + " = " + valueString, nil
}

func (cg *codeGenerator) generateSexpCode(sexp SExp) (string, error) {
	if sexp.Number != "" {
		return "make_number(" + sexp.Number + ")", nil
	}

	if len(sexp.List) > 0 && sexp.List[0].Id == "fun" { // function declaration
		str, err := cg.generateFunction(sexp)
		if err != nil {
			return "", err
		}
		return str, nil
	}
	if len(sexp.List) > 0 { // function application
		var bldr strings.Builder
		for i, arg := range sexp.List[1:] {
			code, err := cg.generateSexpCode(arg)
			if err != nil {
				return "", nil
			}
			if i == 0 {
				bldr.WriteString(code)
			} else {
				bldr.WriteString("," + code)
			}
		}
		return functionName(sexp.List[0].Id) + bldr.String() + ")", nil
	}

	if sexp.Id != "" { // identifier
		var idtxt strings.Builder
		idtxt.WriteString("ctx")

		ctx := cg.currentContext
		for ctx != nil {
			if ctx.variables[sexp.Id] {
				idtxt.WriteString("->" + sexp.Id)
				break
			}

			ctx = ctx.parent
			idtxt.WriteString("->parent")
		}
		return idtxt.String(), nil
	}

	return "", nil
}

func (cg *codeGenerator) generateFunction(sexp SExp) (string, error) {
	ctx := context{}
	ctx.parent = cg.currentContext
	ctx.variables = map[string]bool{}

	n := cg.currentFunctionNumber
	name := "fn" + strconv.Itoa(n)

	cg.currentFunctionNumber += 1
	cg.currentContext = &ctx
	cg.currentContext.name = name

	for _, arg := range sexp.List[1].List {
		if arg.Id == "" {
			// ERROR
		}

		ctx.variables[arg.Id] = true
	}

	functionBody, err := cg.generateSexpCode(sexp.List[2])
	if err != nil {
		return "", err
	}

	cg.definitionsCode.WriteString("struct " + name + "struct;")
	cg.structsCode.WriteString("\n\ntypedef struct " + name + "struct{\n")
	if ctx.parent != nil {
		cg.structsCode.WriteString("struct " + ctx.parent.name + "struct *parent;\n")
	}

	params := sexp.List[1]
	for _, param := range params.List {
		cg.structsCode.WriteString("value " + param.Id + ";\n")
	}

	cg.structsCode.WriteString("} " + name + "context ;\n")
	cg.functionsCode.WriteString("value " + name + "(" + name + "context* ctx")
	// function arguments
	for _, param := range params.List {
		cg.functionsCode.WriteString(", value " + param.Id)
	}
	cg.functionsCode.WriteString(") {\n")
	for _, param := range params.List {
		cg.functionsCode.WriteString("ctx->" + param.Id + " = " + param.Id + ";\n")
	}
	cg.functionsCode.WriteString("return " + functionBody + ";\n}\n")
	cg.currentContext = ctx.parent

	return `make_lambda(&fn` + strconv.Itoa(n) + `, make_context(sizeof(` + name + `context), ctx));`, nil
}

func functionName(id string) string {
	if id == "+" {
		return "plus("
	}
	if id == "printval" {
		return "printval("
	}
	return "ctx->" + id + ".actualValue.fun->fun(" + "ctx->" + id + ".actualValue.fun->ctx,"
}
