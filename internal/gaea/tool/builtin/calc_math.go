package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(calcMath{}) }

// calcMath evaluates mathematical expressions using Go's AST parser.
// Supports + - * / ( ) and functions: sqrt, sin, cos, tan, pow, abs, log, exp, floor, ceil.
type calcMath struct{}

func (calcMath) Name() string { return "calc_math" }

func (calcMath) Description() string {
	return "计算数学表达式（支持+,-,*,/,(,)和常见函数：sqrt,sin,cos,tan,pow,abs,log,exp,floor,ceil）。基于Go AST解析，纯安全计算。"
}

func (calcMath) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "expression":{"type":"string","description":"数学表达式字符串，如 \"sqrt(16) + 3*sin(pi/2)\""}
},
"required":["expression"]
}`)
}

func (calcMath) ReadOnly() bool { return true }

func (calcMath) CompactDescription() string     { return compactDesc["calc_math"] }
func (calcMath) CompactSchema() json.RawMessage { return compactSchema["calc_math"] }

func (calcMath) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Expression == "" {
		return "", fmt.Errorf("expression is required")
	}

	// Preprocess: replace common constants
	expr := p.Expression
	expr = strings.ReplaceAll(expr, "π", "pi")
	expr = strings.ReplaceAll(expr, "Pi", "pi")
	expr = strings.ReplaceAll(expr, "PI", "pi")

	// Wrap expression to be parseable as an expression
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	result, err := eval(node)
	if err != nil {
		return "", fmt.Errorf("evaluation error: %w", err)
	}

	// Format result
	formatted := strconv.FormatFloat(result, 'g', -1, 64)
	return fmt.Sprintf("%s = %s", expr, formatted), nil
}

// eval evaluates an AST expression node.
func eval(node ast.Node) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.FLOAT || n.Kind == token.INT {
			return strconv.ParseFloat(n.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal: %s", n.Value)

	case *ast.ParenExpr:
		return eval(n.X)

	case *ast.UnaryExpr:
		val, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -val, nil
		case token.ADD:
			return val, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", n.Op)
		}

	case *ast.BinaryExpr:
		left, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		right, err := eval(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			return float64(int64(left) % int64(right)), nil
		default:
			return 0, fmt.Errorf("unsupported binary operator: %s", n.Op)
		}

	case *ast.CallExpr:
		fnName, ok := n.Fun.(*ast.Ident)
		if !ok {
			return 0, fmt.Errorf("unsupported function call")
		}
		name := strings.ToLower(fnName.Name)

		// Evaluate arguments
		args := make([]float64, len(n.Args))
		for i, arg := range n.Args {
			v, err := eval(arg)
			if err != nil {
				return 0, fmt.Errorf("arg %d: %w", i+1, err)
			}
			args[i] = v
		}

		switch name {
		case "sqrt":
			if len(args) != 1 {
				return 0, fmt.Errorf("sqrt requires 1 argument")
			}
			if args[0] < 0 {
				return 0, fmt.Errorf("sqrt of negative number")
			}
			return math.Sqrt(args[0]), nil

		case "sin":
			if len(args) != 1 {
				return 0, fmt.Errorf("sin requires 1 argument")
			}
			return math.Sin(args[0]), nil

		case "cos":
			if len(args) != 1 {
				return 0, fmt.Errorf("cos requires 1 argument")
			}
			return math.Cos(args[0]), nil

		case "tan":
			if len(args) != 1 {
				return 0, fmt.Errorf("tan requires 1 argument")
			}
			return math.Tan(args[0]), nil

		case "pow":
			if len(args) != 2 {
				return 0, fmt.Errorf("pow requires 2 arguments")
			}
			return math.Pow(args[0], args[1]), nil

		case "abs":
			if len(args) != 1 {
				return 0, fmt.Errorf("abs requires 1 argument")
			}
			return math.Abs(args[0]), nil

		case "log":
			if len(args) != 1 {
				return 0, fmt.Errorf("log requires 1 argument")
			}
			if args[0] <= 0 {
				return 0, fmt.Errorf("log of non-positive number")
			}
			return math.Log(args[0]), nil

		case "log10":
			if len(args) != 1 {
				return 0, fmt.Errorf("log10 requires 1 argument")
			}
			if args[0] <= 0 {
				return 0, fmt.Errorf("log10 of non-positive number")
			}
			return math.Log10(args[0]), nil

		case "exp":
			if len(args) != 1 {
				return 0, fmt.Errorf("exp requires 1 argument")
			}
			return math.Exp(args[0]), nil

		case "floor":
			if len(args) != 1 {
				return 0, fmt.Errorf("floor requires 1 argument")
			}
			return math.Floor(args[0]), nil

		case "ceil":
			if len(args) != 1 {
				return 0, fmt.Errorf("ceil requires 1 argument")
			}
			return math.Ceil(args[0]), nil

		case "sinh":
			if len(args) != 1 {
				return 0, fmt.Errorf("sinh requires 1 argument")
			}
			return math.Sinh(args[0]), nil

		case "cosh":
			if len(args) != 1 {
				return 0, fmt.Errorf("cosh requires 1 argument")
			}
			return math.Cosh(args[0]), nil

		case "tanh":
			if len(args) != 1 {
				return 0, fmt.Errorf("tanh requires 1 argument")
			}
			return math.Tanh(args[0]), nil

		case "asin":
			if len(args) != 1 {
				return 0, fmt.Errorf("asin requires 1 argument")
			}
			return math.Asin(args[0]), nil

		case "acos":
			if len(args) != 1 {
				return 0, fmt.Errorf("acos requires 1 argument")
			}
			return math.Acos(args[0]), nil

		case "atan":
			if len(args) != 1 {
				return 0, fmt.Errorf("atan requires 1 argument")
			}
			return math.Atan(args[0]), nil

		case "atan2":
			if len(args) != 2 {
				return 0, fmt.Errorf("atan2 requires 2 arguments")
			}
			return math.Atan2(args[0], args[1]), nil

		case "max":
			if len(args) < 2 {
				return 0, fmt.Errorf("max requires at least 2 arguments")
			}
			v := args[0]
			for _, a := range args[1:] {
				if a > v {
					v = a
				}
			}
			return v, nil

		case "min":
			if len(args) < 2 {
				return 0, fmt.Errorf("min requires at least 2 arguments")
			}
			v := args[0]
			for _, a := range args[1:] {
				if a < v {
					v = a
				}
			}
			return v, nil

		case "round":
			if len(args) != 1 {
				return 0, fmt.Errorf("round requires 1 argument")
			}
			return math.Round(args[0]), nil

		default:
			return 0, fmt.Errorf("unknown function: %s", name)
		}

	case *ast.Ident:
		switch n.Name {
		case "pi", "Pi", "PI":
			return math.Pi, nil
		case "e", "E":
			return math.E, nil
		default:
			return 0, fmt.Errorf("unknown identifier: %s", n.Name)
		}

	default:
		return 0, fmt.Errorf("unsupported expression type: %T", node)
	}
}
