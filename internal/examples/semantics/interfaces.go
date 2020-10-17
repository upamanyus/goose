package semantics

import "fmt"

// Interface with method set
type testInterface interface {
	Square() uint64
}

// Empty interface
type emptyInterface interface{}

func measure(t testInterface) {
	fmt.Println(t.Square())
}

type TestStruct struct {
	Number uint64
}

func (t TestStruct) Square() uint64 {
	return t.Number * t.Number
}

func test() {
	s := TestStruct{
		Number: 2,
	}

	measure(s)
}
