package toolkit

import "testing"

/*
* See
https://pkg.go.dev/testing
*/

func Test_Tools_RandomString(t *testing.T) {

	var testTools Tools

	ln := 10
	s := testTools.RandomString(ln)

	if len(s) != ln {
		t.Error("incorrect length of random string returned")
	}

}
