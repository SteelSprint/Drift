Below is a small Go program that implements a basic calculator. Your task is to add driftpin spec coverage to this EXISTING codebase — do not rewrite the logic. You should:

1. Create spec files (`*.pin.xml`) that describe what each function does.
2. Place `D! id=<markerid> range-start` and `D! id=<markerid> range-end` markers in the code, wrapping the region that implements each spec.
3. Link markers to specs.
4. Run `drift todo` and resolve any drift.

Write the following code to `main.go` in your project root, then add driftpin coverage:

```go
package main

import (
	"fmt"
	"os"
	"strconv"
)

func add(a, b float64) float64 { return a + b }
func sub(a, b float64) float64 { return a - b }
func mul(a, b float64) float64 { return a * b }
func div(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: calc <a> <op> <b>")
		os.Exit(1)
	}
	a, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil {
		fmt.Println("invalid a:", err)
		os.Exit(1)
	}
	b, err := strconv.ParseFloat(os.Args[3], 64)
	if err != nil {
		fmt.Println("invalid b:", err)
		os.Exit(1)
	}
	op := os.Args[2]
	switch op {
	case "+":
		fmt.Println(add(a, b))
	case "-":
		fmt.Println(sub(a, b))
	case "*":
		fmt.Println(mul(a, b))
	case "/":
		result, err := div(a, b)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(result)
	default:
		fmt.Println("unknown operator:", op)
		os.Exit(1)
	}
}
```
