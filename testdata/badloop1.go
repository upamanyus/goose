package example

func Skip() {}

func BadLoop(i uint64) {
	if i == 0 {
		return
	} else { //@ diag("else with early return")
		Skip()
	}
	Skip()
}
