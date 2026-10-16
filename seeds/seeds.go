// Command seeds implements the generation and verification of the NIST
// pseudorandom prime curves P-192, P-224, P-256, P-384, and P-521 from their
// domain parameter seeds, as specified in [SP 800-186] Appendix C.3.1 and C.3.2.
//
// The same procedures appear in FIPS 186-4 Appendix D.5 and D.6 and in FIPS
// 186-2 Appendix 6.4 and 6.5, and originate from [ANSI X9.62-1998] Annex
// A.3.3.2 and A.3.4.2, which allow seeds longer than 160 bits and apply the
// curve acceptance criteria of Annex A.3.2. All NIST pseudorandom prime curves
// use a = -3, SHA-1, and a 160-bit seed, which this package hardcodes.
//
// Curve orders are computed with the SEA implementation of PARI/GP, which must
// be in PATH as gp, unless they are already known to this package.
//
// The NIST seeds are consecutive 20-byte blocks of (int)(drand48() * 256)
// output after srand48(t) for the timestamps in streams. A run could
// produce several curves: the ANSI X9.62-1998 Annex J.5 sample curves
// prime192v3 and prime192v2 are blocks 5228 and 6013 of the P-192 stream, and
// prime239v1, prime239v2, and prime239v3 are blocks 2185, 2320, and 3048 of
// srand48(863711343). The generator rejected the candidates with n ≥ 2^bits(p)
// (see suitableOrder) and, for a reason not yet identified, about half of the
// remaining prime-order candidates: blocks 2198 and 5405 of P-192, 777, 1485,
// and 5736 of P-224, 876 of P-256, and 36, 1562, 2491, 2682, and 2863 of the
// prime239 stream.
//
// [SP 800-186]: https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-186.pdf
// [ANSI X9.62-1998]: https://safecurves.cr.yp.to/grouper.ieee.org/groups/1363/private/x9-62-09-20-98.pdf
package main

import (
	"bytes"
	"crypto/elliptic"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// streams lists the drand48 seeds t passed to srand48 by the generation runs,
// the prime field of each run, and the published domain parameter seeds found
// in each stream, from FIPS 186-4 Appendix D.1.2 and ANSI X9.62-1998 Annex J.5.
var streams = []struct {
	t         uint32
	p         *big.Int
	published map[string]string // seed → curve name
}{
	{863711343, hexInt("7fffffffffffffffffffffff7fffffffffff8000000000007fffffffffff"), map[string]string{
		"E43BB460F0B80CC0C0B075798E948060F8321B7D": "prime239v1",
		"E8B4011604095303CA3B8099982BE09FCB9AE616": "prime239v2",
		"7D7374168FFE3471B60A857686A19475D3BFA2FF": "prime239v3",
	}},
	{863966734, P192.P, map[string]string{
		"3045AE6FC8422F64ED579528D38120EAE12196D5": "P-192 a.k.a. prime192v1",
		"C469684435DEB378C4B65CA9591E2A5763059A2E": "prime192v3",
		"31A92EE2029FD10D901B113E990710F0D21AC6B6": "prime192v2",
	}},
	{871002724, elliptic.P521().Params().P, map[string]string{
		"D09E8800291CB85396CC6717393284AAA0DA64BA": "P-521",
	}},
	{871696354, elliptic.P256().Params().P, map[string]string{
		"C49D360886E704936A6678E1139D26B7819F7E90": "P-256 a.k.a. prime256v1",
	}},
	{872952608, elliptic.P384().Params().P, map[string]string{
		"A335926AA319A27A1D00896A6773A4827ACDAC73": "P-384",
	}},
	{891607927, elliptic.P224().Params().P, map[string]string{
		"BD71344799D5C7FCDC45B59FA3B9AB8F6A948BC5": "P-224",
	}},
}

func main() {
	log.SetFlags(0)
	for _, s := range streams {
		fmt.Printf("srand48(%d) [%v]:\n", s.t, time.Unix(int64(s.t), 0).UTC())
		for _, a := range Search(s.p, s.t, s.published) {
			name := s.published[strings.ToUpper(hex.EncodeToString(a.seed[:]))]
			if name == "" {
				name = "not published"
			}
			fmt.Printf("  accepted %6s seed %x: %s\n", ordinal(a.n), a.seed, name)
		}
	}
}

// srand48 returns an [io.Reader] producing the top byte of each successive
// 48-bit state of the C library's drand48 generator, specified in [POSIX.1-2017],
// seeded with srand48(seed).
//
// [POSIX.1-2017]: https://pubs.opengroup.org/onlinepubs/9699919799/functions/drand48.html
func srand48(seed uint32) io.Reader {
	// "The initializer function srand48() sets the high-order 32 bits of Xi
	// to the low-order 32 bits contained in its argument. The low-order 16
	// bits of Xi are set to the arbitrary value 330E16."
	return &drand48{x: uint64(seed)<<16 | 0x330E}
}

type drand48 struct {
	x uint64 // 48-bit state
}

func (d *drand48) Read(p []byte) (int, error) {
	for i := range p {
		// Xₙ₊₁ = (aXₙ + c) mod m, with m = 2⁴⁸, a = 0x5DEECE66D, and c = 0xB
		d.x = (d.x*0x5DEECE66D + 0xB) & (1<<48 - 1)
		// "Then the appropriate number of bits, according to the type of
		// data item to be returned, are copied from the high-order (leftmost)
		// bits of Xi and transformed into the returned value."
		//
		// p[i] = drand48() * 256
		p[i] = byte(d.x >> 40)
	}
	return len(p), nil
}

// Search draws 160-bit seeds from drand48 seeded with srand48(t) until all the
// published seeds have been drawn, and returns those that [Generate] accepts
// for p, with their 1-based positions in the sequence.
func Search(p *big.Int, t uint32, published map[string]string) []accepted {
	var out []accepted
	seen := 0
	r := srand48(t)
	for n := 1; seen < len(published); n++ {
		var seed [20]byte
		io.ReadFull(r, seed[:])
		if _, ok := published[strings.ToUpper(hex.EncodeToString(seed[:]))]; ok {
			seen++
		}
		if Generate(p, seed) {
			out = append(out, accepted{n, seed})
		}
	}
	return out
}

type accepted struct {
	n    int
	seed [20]byte
}

// Generate derives the coefficient b of the curve y² = x³ - 3x + b over GF(p)
// from the 160-bit seed, according to SP 800-186 Appendix C.3.1 (ANSI X9.62
// Annex A.3.3.2), and reports whether the curve satisfies the criteria of
// Appendix C.2.1.1 (ANSI X9.62 Annex A.3.2), i.e. whether the seed is accepted.
func Generate(p *big.Int, seed [20]byte) bool {
	c := hashSeed(seed, p.BitLen())

	// Step 8: reject the seed if c = 0 or 4c + 27 ≡ 0 (mod p). With a = -3 and
	// b² ≡ -27/c, 4a³ + 27b² ≡ -27(4c + 27)/c (mod p), so this is equivalent to
	// the non-zero discriminant check of X9.62 Annex A.3.3.2 Step 8.
	if c.Sign() == 0 {
		return false
	}
	d := new(big.Int).Mul(c, big.NewInt(4))
	d.Add(d, big.NewInt(27))
	if d.Mod(d, p).Sign() == 0 {
		return false
	}

	// Step 9: a = -3 and cb² ≡ a³ (mod p), so b² ≡ -27/c (mod p).
	//
	// Either square root may be chosen. If p ≡ 1 (mod 4) the two curves are
	// isomorphic. If p ≡ 3 (mod 4) they are quadratic twists of each other,
	// and all NIST curves with p ≡ 3 (mod 4) use the root computed by X9.62
	// Annex D.1.4 Algorithm 1, (-27/c)^((p+1)/4), even P-384 where both twists
	// have prime order. That is also what ModSqrt computes.
	b := new(big.Int).ModInverse(c, p)
	b.Mul(b, big.NewInt(-27)).Mod(b, p)
	if b.ModSqrt(b, p) == nil {
		return false
	}

	// Step 10: check that the curve has suitable order.
	return suitableOrder(p, order(p, b))
}

// hashSeed computes the integer c of SP 800-186 Appendix C.3.2, Steps 1–6, for
// a prime p of bit length l. This is the integer r of X9.62 Annex A.3.3.2,
// which clears the leftmost bit of a (l - 160v)-bit h₀ instead of taking w bits.
func hashSeed(seed [20]byte, l int) *big.Int {
	v := (l - 1) / 160
	w := l - 160*v - 1

	// h₀ is the w rightmost bits of SHA-1(s).
	h := sha1.Sum(seed[:])
	c := new(big.Int).SetBytes(h[:])
	c.Mod(c, new(big.Int).Lsh(big.NewInt(1), uint(w)))

	// hᵢ = SHA-1(sᵢ), where sᵢ is the 160-bit encoding of (z + i) mod 2¹⁶⁰ and
	// z is the integer encoded by s.
	z := new(big.Int).SetBytes(seed[:])
	for i := 1; i <= v; i++ {
		si := new(big.Int).Add(z, big.NewInt(int64(i)))
		si.Mod(si, new(big.Int).Lsh(big.NewInt(1), 160))
		hi := sha1.Sum(si.FillBytes(make([]byte, 20)))

		// c = h₀ || h₁ || … || hᵥ
		c.Lsh(c, 160)
		c.Or(c, new(big.Int).SetBytes(hi[:]))
	}

	return c
}

// suitableOrder reports whether a curve over GF(p) of order n satisfies the
// criteria of SP 800-186 Appendix C.2.1.1 and C.1: prime order, so h = 1
// (Criterion 2; X9.62 Annex A.3.2 Step 3 only requires a smooth cofactor), not
// anomalous (Criterion 4; X9.62 Annex A.1.2), and embedding degree k ≥ 2¹⁰
// (Criterion 5; X9.62 Annex A.1.1 with B = 2¹⁰ - 1 instead of 20), as well as
// the undocumented criterion that n has at most as many bits as p.
func suitableOrder(p, n *big.Int) bool {
	if !n.ProbablyPrime(20) {
		return false
	}
	// n < 2^bits(p). This is not a documented criterion, but every prime-order
	// candidate with a longer n preceding a published seed was rejected (ten
	// in the P-224 and P-384 streams), while prime239v2 was accepted with
	// p < n < 2^239. It also implies the curve is not anomalous.
	if n.BitLen() > p.BitLen() {
		return false
	}
	// pᵏ ≢ 1 (mod n) for all 1 ≤ k < 2¹⁰
	pk := big.NewInt(1)
	for range 1<<10 - 1 {
		pk.Mul(pk, p).Mod(pk, n)
		if pk.IsInt64() && pk.Int64() == 1 {
			return false
		}
	}
	return true
}

// order computes the order of the curve y² = x³ - 3x + b over GF(p), returning
// zero if the order was found to have a small prime factor before the
// computation completed.
func order(p, b *big.Int) *big.Int {
	// Known results are hardcoded in orders.
	key := fmt.Sprintf("%x %x", p, b)
	if s, ok := orders[key]; ok {
		n, ok := new(big.Int).SetString(s, 16)
		if !ok {
			log.Fatalf("seeds: invalid cached order for %s", key)
		}
		return n
	}

	n := sea(p, b)
	cacheOrder(key, n)
	return n
}

// sea runs the SEA implementation of PARI/GP to compute the order of the curve
// y² = x³ - 3x + b over GF(p), with early abort for non-prime orders.
func sea(p, b *big.Int) *big.Int {
	script := fmt.Sprintf("print(ellsea(ellinit([-3, %d], %d), 1))", b, p)
	cmd := exec.Command("gp", "-q", "-f", "--default", "parisizemax=2000000000")
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("seeds: running gp: %v", err)
	}
	n, ok := new(big.Int).SetString(strings.TrimSpace(stdout.String()), 10)
	if !ok {
		log.Fatalf("seeds: unexpected gp output: %q (%q)", stdout.String(), stderr.String())
	}
	return n
}

// orders caches the results of sea, keyed by p and b in hexadecimal. It is
// loaded from orders.txt in the current directory, which has one "p b n" line
// per entry, and new results are appended to it.
var orders = func() map[string]string {
	m := make(map[string]string)
	data, err := os.ReadFile("orders.txt")
	if err != nil {
		log.Printf("seeds: %v, starting with an empty cache", err)
		return m
	}
	for line := range strings.Lines(string(data)) {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if len(f) != 3 {
			panic("seeds: invalid line in orders.txt: " + line)
		}
		m[f[0]+" "+f[1]] = f[2]
	}
	return m
}()

// cacheOrder adds the order n of the curve identified by key to orders, and
// appends it to orders.txt.
func cacheOrder(key string, n *big.Int) {
	orders[key] = n.Text(16)
	f, err := os.OpenFile("orders.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("seeds: failed to open orders.txt, add manually: %s %s (%v)", key, n.Text(16), err)
		return
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s %s\n", key, n.Text(16)); err != nil {
		log.Printf("seeds: failed to append to orders.txt, add manually: %s %s (%v)", key, n.Text(16), err)
	}
}

var P192 = &elliptic.CurveParams{
	Name:    "P-192",
	BitSize: 192,
	P:       hexInt("fffffffffffffffffffffffffffffffeffffffffffffffff"),
	N:       hexInt("ffffffffffffffffffffffff99def836146bc9b1b4d22831"),
	B:       hexInt("64210519e59c80e70fa7e9ab72243049feb8deecc146b9b1"),
	Gx:      hexInt("188da80eb03090f67cbf20eb43a18800f4ff0afd82ff1012"),
	Gy:      hexInt("07192b95ffc8da78631011ed6b24cdd573f977a11e794811"),
}

// ordinal formats n as an English ordinal number, e.g. "139th".
func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

func hexInt(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("seeds: invalid hex integer " + s)
	}
	return n
}
