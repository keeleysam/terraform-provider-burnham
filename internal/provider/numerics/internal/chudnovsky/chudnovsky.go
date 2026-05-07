/*
Package chudnovsky computes π to arbitrary precision using the Chudnovsky brothers' series with binary splitting in pure integer arithmetic.

This package is internal: it's used by sibling cmd/genpi to produce the DPD-packed pi_packed.bin shipped with terraform-burnham, and by the numerics package's cross-validation test. Production code does not import it — Burnham's runtime path serves digits from the embedded packed table, not from a runtime computation.

References:
  - Chudnovsky, D. & Chudnovsky, G. (1988). Approximations and complex multiplication according to Ramanujan.
  - Wikipedia: https://en.wikipedia.org/wiki/Chudnovsky_algorithm
  - Binary splitting: https://en.wikipedia.org/wiki/Binary_splitting
  - Reference Python implementation: https://www.craig-wood.com/nick/articles/pi-chudnovsky/
  - Reference Go implementation: https://github.com/mgomes/chudnovsky (MIT)

Why this is hand-rolled rather than imported.

We evaluated github.com/ericlagergren/decimal — a maintained BSD-3 library (v3.3.1 tagged 2019, master last touched April 2024) whose Context.Pi method implements the same Chudnovsky binary-splitting algorithm coded above. Performance head-to-head was a tie at our 1M-digit target (~3 s, both ultimately bottoming out on math/big.Int.Mul, which is what their decimal.Big calls under the hood anyway). The deciding factor was memory: at 10M digits their implementation peaked at 3.5 GB allocated vs. 250 MB for this one — a 14× difference — driven by the per-call closure pattern in their Pi helpers (`var tmp Big; ... return &tmp`) and decimal.Big's per-instance Context overhead. The bloat is structural, not a one-line fix.

We don't actually hit that memory ceiling at our 1M cap, so the practical difference at our workload is negligible — but it tipped the choice to "100 lines we own" over "depend on a multi-thousand-line decimal arithmetic library to call the same Chudnovsky math we already have."

Follow-up worth doing eventually: contribute the memory fix upstream (rework the BinarySplit helpers to take pre-allocated state, drop per-call allocations from getPiA/P/Q). If that lands and tags a release, the calculus flips — we'd shed 100 lines of math we have to maintain in exchange for an upstream dep that does the same thing equally well. Until then, this stays.

The series is

    1/π = 12 · Σ_{k=0..∞} [(-1)^k · (6k)! · (13591409 + 545140134k)]
                          ─────────────────────────────────────────
                          [(3k)! · (k!)^3 · 640320^(3k+3/2)]

Each term contributes about 14.181647 additional decimal digits.

Binary splitting evaluates the partial sum Σ_{k=a..b-1} via three integer quantities P(a,b), Q(a,b), T(a,b) satisfying

    P(a,b)·P(b,c) = P(a,c)
    Q(a,b)·Q(b,c) = Q(a,c)
    T(a,c)        = Q(b,c)·T(a,b) + P(a,b)·T(b,c)

with the partial sum equal to T(a,b)/Q(a,b). The base case (b - a == 1) plugs in the closed-form term values; the recursive case combines two halves. After combining, we recover π via the closed form

    π = (Q · 426880 · √10005) / T

evaluated entirely in *big.Int by scaling: compute floor(π · 10^digits) using big.Int.Sqrt over (10005 · 10^(2·digits)).
*/

package chudnovsky

import (
	"fmt"
	"math/big"
	"strings"
)

// PiDigits returns the first `digits` decimal digits of π *following* the decimal point (i.e. with the leading "3." stripped). The result is always exactly `digits` characters long and contains only ASCII '0'-'9'.
//
// digits must be >= 1.
func PiDigits(digits int) string {
	if digits < 1 {
		panic(fmt.Sprintf("chudnovsky.PiDigits: digits must be >= 1, got %d", digits))
	}

	// Number of Chudnovsky iterations needed. Each term contributes log10(640320^3 / (24·6·2·6)) ≈ 14.181647 digits; using 14 gives a safety buffer.
	n := int64(digits/14) + 2

	P, Q, T := binarySplit(0, n)
	_ = P // P is unused at the top level; only Q and T feed the closing formula.

	// argSqrt = 10005 · 10^(2·digits)
	// sqrt(argSqrt) = √10005 · 10^digits   (integer floor of)
	tenPow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(2*digits)), nil)
	argSqrt := new(big.Int).Mul(big.NewInt(10005), tenPow)
	sqrtC := new(big.Int).Sqrt(argSqrt)

	// num = Q · 426880 · sqrtC
	num := new(big.Int).Mul(Q, big.NewInt(426880))
	num.Mul(num, sqrtC)

	// piScaled = floor(num / T) = floor(π · 10^digits)
	piScaled := new(big.Int).Quo(num, T)

	// piScaled prints as "3<digits>" (a (digits+1)-character integer). We want only the digits after the decimal point.
	s := piScaled.String()
	if len(s) < digits+1 {
		// Shouldn't happen given the formulation, but be defensive: pad with leading zeros so the slicing below is safe.
		s = strings.Repeat("0", digits+1-len(s)) + s
	}
	if s[0] != '3' {
		panic(fmt.Sprintf("chudnovsky.PiDigits: expected leading '3', got %q (digits=%d)", s[0:1], digits))
	}
	return s[1 : 1+digits]
}

// constC3Over24 = 640320^3 / 24 = 10939058860032000.
//
// 640320^3 = 262537412640768000; dividing by 24 gives the integer constant that appears in Q's base case (a^3 · C3/24).
var constC3Over24 = big.NewInt(10939058860032000)

// binarySplit computes (P, Q, T) for the half-open interval [a, b) of Chudnovsky terms. Caller invokes with a=0 to capture the k=0 term.
//
// Base case (b - a == 1):
//
//	a == 0:  P = 1, Q = 1
//	a >= 1:  P = (6a-5)(2a-1)(6a-1)
//	         Q = a^3 · (640320^3 / 24)
//	T = P · (13591409 + 545140134·a)
//	if a is odd, T = -T  (encodes the (-1)^k sign).
//
// Recursive case: split at m = (a+b)/2; combine.
func binarySplit(a, b int64) (P, Q, T *big.Int) {
	if b-a == 1 {
		if a == 0 {
			P = big.NewInt(1)
			Q = big.NewInt(1)
		} else {
			// P = (6a-5)(2a-1)(6a-1)
			P = new(big.Int).Mul(big.NewInt(6*a-5), big.NewInt(2*a-1))
			P.Mul(P, big.NewInt(6*a-1))

			// Q = a^3 · C3/24
			aBig := big.NewInt(a)
			Q = new(big.Int).Mul(aBig, aBig)
			Q.Mul(Q, aBig)
			Q.Mul(Q, constC3Over24)
		}
		// T = P · (13591409 + 545140134·a)
		coef := new(big.Int).Mul(big.NewInt(545140134), big.NewInt(a))
		coef.Add(coef, big.NewInt(13591409))
		T = new(big.Int).Mul(P, coef)
		if a%2 == 1 {
			T.Neg(T)
		}
		return
	}

	m := (a + b) / 2
	Pam, Qam, Tam := binarySplit(a, m)
	Pmb, Qmb, Tmb := binarySplit(m, b)

	P = new(big.Int).Mul(Pam, Pmb)
	Q = new(big.Int).Mul(Qam, Qmb)

	// T = Q(m,b)·T(a,m) + P(a,m)·T(m,b)
	T = new(big.Int).Mul(Qmb, Tam)
	rhs := new(big.Int).Mul(Pam, Tmb)
	T.Add(T, rhs)
	return
}
