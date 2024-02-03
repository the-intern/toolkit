package toolkit

import "crypto/rand"

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Tools is the type used to instantiate this module.  Any variable of this type has access to the methods defined with the receiver *Tools
type Tools struct {
}

// RandomString returns a random string of characters of length n, using as its source randomStringSource for the possible characters to comprise the return value
func (t *Tools) RandomString(n int) string {

	s, r := make([]rune, n), []rune(randomStringSource)

	for i := range s {

		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))

		s[i] = r[x%y]
	}

	return string(s)
}

/********************************************
* Notes
func rand.Prime(rand io.Reader, bits int) (*big.Int, error)
Prime returns
	- a number of the given bit length that is prime with high probability.
	-  error for any error returned by rand.Read or if bits < 2.

func (*big.Int).Uint64() uint64
Uint64 returns the uint64 representation of x. If x cannot be represented in a uint64, the result is undefined.

uint64 is the set of all unsigned 64-bit integers. Range: 0 through 18446744073709551615.
*/
/*********************************************/
