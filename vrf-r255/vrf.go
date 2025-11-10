// Package vrf implements the [ECVRF-RISTRETTO255-SHA512] ciphersuite for
// [RFC 9381] Verifiable Random Functions.
//
// [ECVRF-RISTRETTO255-SHA512]: https://c2sp.org/vrf-r255
// [RFC 9381]: https://rfc-editor.org/rfc/rfc9381.html
package vrf

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"

	r255 "github.com/gtank/ristretto255"
)

const (
	suite_string = "\xFFc2sp.org/vrf-r255"

	encode_to_curve_ds         = 0x82
	challenge_generation_front = 0x02
	challenge_generation_back  = 0x00
	proof_to_hash_front        = 0x03
	proof_to_hash_back         = 0x00

	cLen  = 16 // note that c as a Scalar is encoded as 32 bytes
	qLen  = 32
	ptLen = 32
)

// ErrFailedVerification indicates that proof verification failed.
var ErrFailedVerification = errors.New("proof verification failed")

// PublicKey can be used to verify a [Proof] that the corresponding [Proof.Hash]
// is the correct VRF hash of an input.
type PublicKey struct {
	y *r255.Element
}

// PrivateKey can be used to compute a VRF hash of an input and a [Proof] that
// it is correct.
type PrivateKey struct {
	y *PublicKey
	x *r255.Scalar
}

// GenerateKey generates a new random [PrivateKey].
func GenerateKey() *PrivateKey {
	r := make([]byte, 64)
	rand.Read(r)
	s := must(r255.NewScalar().SetUniformBytes(r))
	y := r255.NewIdentityElement().ScalarBaseMult(s)
	return &PrivateKey{
		y: &PublicKey{y: y},
		x: s,
	}
}

// NewPrivateKey returns a [PrivateKey] from its byte encoding.
func NewPrivateKey(sk []byte) (*PrivateKey, error) {
	s, err := r255.NewScalar().SetCanonicalBytes(sk)
	if err != nil {
		return nil, err
	}
	y := r255.NewIdentityElement().ScalarBaseMult(s)
	return &PrivateKey{
		y: &PublicKey{y: y},
		x: s,
	}, nil
}

// PublicKey returns the [PublicKey] for x.
func (x *PrivateKey) PublicKey() *PublicKey {
	return x.y
}

// Bytes returns the byte encoding of y.
func (y *PublicKey) Bytes() []byte {
	return y.y.Bytes()
}

// Bytes returns the byte encoding of x.
func (x *PrivateKey) Bytes() []byte {
	return x.x.Bytes()
}

func NewPublicKey(pk []byte) (*PublicKey, error) {
	y, err := r255.NewIdentityElement().SetCanonicalBytes(pk)
	if err != nil {
		return nil, err
	}
	return &PublicKey{y: y}, nil
}

// challenge takes first 16 bytes of in to make Challenge
func challenge(in []byte) *r255.Scalar {
	tmp := make([]byte, 32)
	copy(tmp, in[:cLen])
	// SetCanonicalBytes can't return an error, because the most significant
	// bytes are zero, guaranteeing the value is less than l.
	return must(r255.NewScalar().SetCanonicalBytes(tmp))
}

// generateChallenge generates a Challenge given some elements
//
// Input:
// P1, P2, P3, P4, P5 - EC points
//
// Output:
// c - challenge value, integer between 0 and 2^(8*cLen)-1, as a Scalar
func generateChallenge(pk *PublicKey, h, gamma, u, v *r255.Element) *r255.Scalar {
	c_string := hashToChallenge(pk, h, gamma, u, v)

	// 7. truncated_c_string = c_string[0]...c_string[cLen-1]
	// 8. c = string_to_int(truncated_c_string)
	// you can use SetCanonicalBytes since you have a small enough number
	// 9. Output c
	return challenge(c_string)
}

// expose this step of generateChallenge for the c_string test vector
func hashToChallenge(pk *PublicKey, h, gamma, u, v *r255.Element) []byte {
	h1 := sha512.New()

	// 2. Initialize str = suite_string || challenge_generation_domain_separator_front
	h1.Write([]byte(suite_string))
	h1.Write([]byte{challenge_generation_front})

	// 3. for PJ in [P1, P2, P3, P4, P5]:
	// str = str || point_to_string(PJ)
	for _, pj := range []*r255.Element{pk.y, h, gamma, u, v} {
		h1.Write(pj.Bytes())
	}

	// 5. str = str || challenge_generation_domain_separator_back
	h1.Write([]byte{challenge_generation_back})

	// 6. c_string = Hash(str)
	return h1.Sum(nil)
}

// Proof is a proof that a VRF hash was computed correctly.
//
// The actual hash can be computed from the proof with [Proof.Hash].
type Proof struct {
	g *r255.Element
	c *r255.Scalar
	s *r255.Scalar
}

// NewProof returns a [Proof] from its byte encoding.
func NewProof(pi []byte) (*Proof, error) {
	if len(pi) != ptLen+cLen+qLen {
		return nil, ErrFailedVerification
	}

	// gamma_string = pi_string[0]...pi_string[ptLen-1]
	gs := pi[:ptLen]

	// c_string = pi_string[ptLen]...pi_string[ptLen+cLen-1]
	cs := pi[ptLen : ptLen+cLen]

	// s_string = pi_string[ptLen+cLen]...pi_string[ptLen+cLen+qLen-1]
	ss := pi[ptLen+cLen : ptLen+cLen+qLen]

	// Gamma = string_to_point(gamma_string)
	// if Gamma = "INVALID" output "INVALID" and stop
	g, err := r255.NewIdentityElement().SetCanonicalBytes(gs)
	if err != nil {
		return nil, err
	}

	// c = string_to_int(c_string)
	c := challenge(cs)

	// s = string_to_int(s_string)
	// if s >= q output "INVALID" and stop
	s, err := r255.NewScalar().SetCanonicalBytes(ss)
	if err != nil {
		return nil, err
	}

	// Output Gamma, c, and s
	return &Proof{g, c, s}, nil
}

// Bytes returns the byte encoding of p.
func (p *Proof) Bytes() (pi []byte) {
	pi = make([]byte, 0, ptLen+cLen+qLen)
	pi = append(pi, p.g.Bytes()...)
	pi = append(pi, p.c.Bytes()[:cLen]...)
	pi = append(pi, p.s.Bytes()...)
	return pi
}

// Hash returns the actual VRF hash proven by p.
func (p *Proof) Hash() (beta []byte) {
	// 6. beta_string = Hash(suite_string || proof_to_hash_domain_separator_front || point_to_string(cofactor * Gamma) || proof_to_hash_domain_separator_back)
	h := sha512.New()
	h.Write([]byte(suite_string))
	h.Write([]byte{proof_to_hash_front})
	h.Write(p.g.Bytes())
	h.Write([]byte{proof_to_hash_back})
	return h.Sum(nil)
}

// Prove computes a VRF hash of the input alpha, and returns a proof that it was
// computed correctly.
//
// The actual hash can be retrieved with [Proof.Hash].
func (x *PrivateKey) Prove(alpha []byte) (pi *Proof) {
	// 1. Use SK to derive the VRF secret scalar x and the VRF public key Y = x*B
	// 2. H = ECVRF_encode_to_curve(encode_to_curve_salt, alpha_string) (see Section 5.4.1)
	salt := x.y.y.Bytes()
	h := encodeToCurve(salt, alpha)

	// 3. h_string = point_to_string(H)
	hs := h.Bytes() // 32 bytes

	// 4. Gamma = x*H
	g := r255.NewIdentityElement().ScalarMult(x.x, h)

	// 5. k = ECVRF_nonce_generation(SK, h_string) (see Section 5.4.2)
	k := x.generateNonce(hs)

	// 6. c = ECVRF_challenge_generation(Y, H, Gamma, k*B, k*H) (see Section 5.4.3)
	u := r255.NewIdentityElement().ScalarBaseMult(k)
	v := r255.NewIdentityElement().ScalarMult(k, h)
	c := generateChallenge(x.y, h, g, u, v)

	// 7. s = (k + c*x) mod q
	s := r255.NewScalar()
	s = s.Multiply(c, x.x)
	s = s.Add(k, s)

	// 8. pi_string = point_to_string(Gamma) || int_to_string(c, cLen) || int_to_string(s, qLen)
	return &Proof{g, c, s}
}

// generates a deterministic nonce from given private key and input
// https://github.com/C2SP/C2SP/blob/filippo/vrf-r255/vrf-r255.md#nonce-generation
func (x *PrivateKey) generateNonce(h []byte) *r255.Scalar {
	// 1. nonce_generation_domain_separator = 0x81
	DS := "\x81"

	// 2. k_string = Hash(suite_string || nonce_generation_domain_separator || int_to_string(SK, 32) || h_string)
	ks := sha512.New()
	ks.Write([]byte(suite_string))
	ks.Write([]byte(DS))
	ks.Write(x.x.Bytes())
	ks.Write(h)

	// 3. k = string_to_int(k_string) mod q
	return must(r255.NewScalar().SetUniformBytes(ks.Sum(nil)))
}

// Verify verifies that p is a valid proof for the generation of its associated
// VRF hash of the input alpha.
func (y *PublicKey) Verify(p *Proof, alpha []byte) (beta []byte, err error) {
	// Not needed - 1. Y = string_to_point(PK_string)
	// Not needed - 2. If Y is "INVALID", output "INVALID" and stop
	// SetCanonicalBytes validates - 3. If validate_key, run ECVRF_validate_key(Y) (Section 5.4.5); if it outputs "INVALID", output "INVALID" and stop
	// Not needed - 4. D = ECVRF_decode_proof(pi_string) (see Section 5.4.4)
	// 5. If D is "INVALID", output "INVALID" and stop
	// 6. (Gamma, c, s) = D

	// 7. H = ECVRF_encode_to_curve(encode_to_curve_salt, alpha_string) (see Section 5.4.1)
	salt := y.y.Bytes()
	h := encodeToCurve(salt, alpha)

	// 8. U = s*B - c*Y
	u1 := r255.NewIdentityElement().ScalarBaseMult(p.s)
	u2 := r255.NewIdentityElement().ScalarMult(p.c, y.y)
	u := r255.NewIdentityElement().Subtract(u1, u2)

	// 9. V = s*H - c*Gamma
	v1 := r255.NewIdentityElement().ScalarMult(p.s, h)
	v2 := r255.NewIdentityElement().ScalarMult(p.c, p.g)
	v := r255.NewIdentityElement().Subtract(v1, v2)

	// 7. c' = ECVRF_challenge_generation(Y, H, Gamma, U, V) (see Section 5.4.3)
	cc := generateChallenge(y, h, p.g, u, v)

	// 8. If c and c' are equal, output ("VALID", ECVRF_proof_to_hash(pi_string)); else output "INVALID"
	if p.c.Equal(cc) == 0 {
		return nil, ErrFailedVerification
	}
	return p.Hash(), nil
}

// https://github.com/C2SP/C2SP/blob/filippo/vrf-r255/vrf-r255.md#encode-to-curve
func encodeToCurve(salt []byte, alpha []byte) *r255.Element {
	x := toUniformBytes(salt, alpha)
	// 3. H = ristretto255_one_way_map(hash_string)
	return must(r255.NewIdentityElement().SetUniformBytes(x))
}

func toUniformBytes(salt []byte, alpha []byte) []byte {
	// 2. hash_string = Hash(suite_string || encode_to_curve_domain_separator || encode_to_curve_salt || alpha_string)
	u := sha512.New()
	u.Write([]byte(suite_string))
	u.Write([]byte{encode_to_curve_ds})
	u.Write(salt)
	u.Write(alpha)
	return u.Sum(nil)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
