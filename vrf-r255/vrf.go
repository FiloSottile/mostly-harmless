package vrf

import (
	"crypto/sha512"

	r255 "github.com/gtank/ristretto255"
)

const (
	C2SP_SUITE_STRING = "\xFFc2sp.org/vrf-r255"

	encode_to_curve_ds         = 0x82
	challenge_generation_front = 0x02
	challenge_generation_back  = 0x00
	proof_to_hash_front        = 0x03
	proof_to_hash_back         = 0x00

	cLen  = 16 // note that c as a Scalar is encoded as 32 bytes
	qLen  = 32
	ptLen = 32
)

// VerificationError indicates that proof verification failed
type VerificationError struct{}

func (e *VerificationError) Error() string {
	return "proof verification failed"
}

// PublicKey for Ristretto VRF
type PublicKey struct {
	y *r255.Element
}

// PrivateKey for Ristretto VRF
type PrivateKey struct {
	Y *PublicKey
	x *r255.Scalar
}

// NewPrivateKey returns a PrivateKey from given bytes
func NewPrivateKey(sk []byte) (*PrivateKey, error) {
	s, err := newScalar(sk)
	if err != nil {
		return nil, err
	}
	y := r255.NewElement()
	y = y.ScalarBaseMult(s)
	return &PrivateKey{
		Y: &PublicKey{y: y},
		x: s,
	}, nil
}

func newElement(in []byte) (*r255.Element, error) {
	x := r255.NewElement()
	x, err := x.SetCanonicalBytes(in)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func newScalar(in []byte) (*r255.Scalar, error) {
	x := r255.NewScalar()
	x, err := x.SetCanonicalBytes(in)
	if err != nil {
		return nil, err
	}
	return x, nil
}

// Challenge is a VRF challenge
type Challenge = r255.Scalar

// newChallenge takes first 16 bytes of in to make Challenge
func newChallenge(in []byte) (*Challenge, error) {
	tmp := make([]byte, 32-cLen)
	tmp = append(in[:cLen], tmp...)
	return newScalar(tmp)
}

// GenerateChallenge generates a Challenge given some elements
//
// Input:
// P1, P2, P3, P4, P5 - EC points
//
// Output:
// c - challenge value, integer between 0 and 2^(8*cLen)-1, as a Scalar
func GenerateChallenge(pk *PublicKey, h, gamma, u, v *r255.Element) (*Challenge, error) {
	c_string := hashToChallenge(pk, h, gamma, u, v)

	// 7. truncated_c_string = c_string[0]...c_string[cLen-1]
	// 8. c = string_to_int(truncated_c_string)
	// you can use SetCanonicalBytes since you have a small enough number
	c, err := newChallenge(c_string)
	if err != nil {
		return nil, err
	}

	// 9. Output c
	return c, nil
}

// expose this step of generateChallenge for the c_string test vector
func hashToChallenge(pk *PublicKey, h, gamma, u, v *r255.Element) []byte {
	h1 := sha512.New()

	// 2. Initialize str = suite_string || challenge_generation_domain_separator_front
	h1.Write([]byte(C2SP_SUITE_STRING))
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

// Proof is a VRF proof
type Proof struct {
	g *r255.Element
	c *Challenge
	s *r255.Scalar
}

// NewProof returns a Proof from given bytes
//
// Input:
// pi_string - VRF proof, octet string (ptLen+cLen+qLen octets)
//
// Output:
// "INVALID", or
// Gamma - a point on E
// c - integer between 0 and 2^(8*cLen)-1
// s - integer between 0 and q-1
func NewProof(pi []byte) (*Proof, error) {
	if len(pi) != ptLen+cLen+qLen {
		return nil, &VerificationError{}
	}

	// gamma_string = pi_string[0]...pi_string[ptLen-1]
	// make copies
	var gs = make([]byte, ptLen)
	copy(gs, pi[:ptLen])

	// c_string = pi_string[ptLen]...pi_string[ptLen+cLen-1]
	var cs = make([]byte, cLen)
	copy(cs, pi[ptLen:ptLen+cLen])

	// s_string = pi_string[ptLen+cLen]...pi_string[ptLen+cLen+qLen-1]
	var ss = make([]byte, qLen)
	copy(ss, pi[ptLen+cLen:ptLen+cLen+qLen])

	// Gamma = string_to_point(gamma_string)
	// if Gamma = "INVALID" output "INVALID" and stop
	g, err := newElement(gs)
	if err != nil {
		return nil, err
	}

	// c = string_to_int(c_string)
	c, err := newChallenge(cs)
	if err != nil {
		return nil, err
	}

	// s = string_to_int(s_string)
	// if s >= q output "INVALID" and stop
	s, err := newScalar(ss)
	if err != nil {
		return nil, err
	}

	// Output Gamma, c, and s
	p := Proof{g, c, s}
	return &p, nil
}

// Bytes returns encoding of Proof
func (p *Proof) Bytes() (pi []byte) {
	pi = append(pi, p.g.Bytes()...)
	pi = append(pi, p.c.Bytes()[:cLen]...)
	pi = append(pi, p.s.Bytes()[:qLen]...)
	return pi
}

// Hash returns hash of proof
func (p *Proof) Hash() (beta []byte, err error) {
	// 1. D = ECVRF_decode_proof(pi_string) (see Section 5.4.4)
	// 2. If D is "INVALID", output "INVALID" and stop
	// 3. (Gamma, c, s) = D

	// 6. beta_string = Hash(suite_string || proof_to_hash_domain_separator_front || point_to_string(cofactor * Gamma) || proof_to_hash_domain_separator_back)
	h := sha512.New()
	h.Write([]byte(C2SP_SUITE_STRING))
	h.Write([]byte{proof_to_hash_front})
	h.Write(p.g.Bytes())
	h.Write([]byte{proof_to_hash_back})
	return h.Sum(nil), nil
}

// Prove returns proof pi that beta is the correct hash output.
//
// Input:
// sk - VRF private key
// alpha - input alpha, an octet string
//
// Output:
// pi - VRF proof, octet string of length ptLen+cLen+qLen
func (x *PrivateKey) Prove(alpha []byte) (pi *Proof, err error) {
	// 1. Use SK to derive the VRF secret scalar x and the VRF public key Y = x*B
	// 2. H = ECVRF_encode_to_curve(encode_to_curve_salt, alpha_string) (see Section 5.4.1)
	salt := x.Y.y.Bytes()
	h, err := encodeToCurve(salt, alpha)
	if err != nil {
		return nil, err
	}

	// 3. h_string = point_to_string(H)
	hs := h.Bytes() // 32 bytes

	// 4. Gamma = x*H
	g := r255.NewElement()
	g = g.ScalarMult(x.x, h)

	// 5. k = ECVRF_nonce_generation(SK, h_string) (see Section 5.4.2)
	k, err := x.GenerateNonce(hs)
	if err != nil {
		return nil, err
	}

	// 6. c = ECVRF_challenge_generation(Y, H, Gamma, k*B, k*H) (see Section 5.4.3)
	u := r255.NewElement()
	u = u.ScalarBaseMult(k)
	v := r255.NewElement()
	v = v.ScalarMult(k, h)
	c, err := GenerateChallenge(x.Y, h, g, u, v)
	if err != nil {
		return nil, err
	}

	// 7. s = (k + c*x) mod q
	s := r255.NewScalar()
	s = s.Multiply(c, x.x)
	s = s.Add(k, s)

	// 8. pi_string = point_to_string(Gamma) || int_to_string(c, cLen) || int_to_string(s, qLen)
	p := Proof{g, c, s}
	return &p, nil
}

// generates a deterministic nonce from given private key and input
// https://github.com/C2SP/C2SP/blob/filippo/vrf-r255/vrf-r255.md#nonce-generation
func (x *PrivateKey) GenerateNonce(h []byte) (k *r255.Scalar, err error) {
	// 1. nonce_generation_domain_separator = 0x81
	DS := "\x81"

	// 2. k_string = Hash(suite_string || nonce_generation_domain_separator || int_to_string(SK, 32) || h_string)
	ks := sha512.New()
	ks.Write([]byte(C2SP_SUITE_STRING))
	ks.Write([]byte(DS))
	ks.Write(x.x.Bytes())
	ks.Write(h)

	// 3. k = string_to_int(k_string) mod q
	out := ks.Sum(nil)
	k = r255.NewScalar()
	k, err = k.SetUniformBytes(out)
	if err != nil {
		return nil, err
	}
	return k, nil
}

// Verify verifies a given proof for a given alpha and public key
func (y *PublicKey) Verify(p *Proof, alpha []byte) (beta []byte, err error) {
	// Not needed - 1. Y = string_to_point(PK_string)
	// Not needed - 2. If Y is "INVALID", output "INVALID" and stop
	// SetCanonicalBytes validates - 3. If validate_key, run ECVRF_validate_key(Y) (Section 5.4.5); if it outputs "INVALID", output "INVALID" and stop
	// Not needed - 4. D = ECVRF_decode_proof(pi_string) (see Section 5.4.4)
	// 5. If D is "INVALID", output "INVALID" and stop
	// 6. (Gamma, c, s) = D

	// 7. H = ECVRF_encode_to_curve(encode_to_curve_salt, alpha_string) (see Section 5.4.1)
	salt := y.y.Bytes()
	h, err := encodeToCurve(salt, alpha)
	if err != nil {
		return nil, err
	}

	// 8. U = s*B - c*Y
	u1 := r255.NewElement()
	u1 = u1.ScalarBaseMult(p.s)
	u2 := r255.NewElement()
	u2 = u2.ScalarMult(p.c, y.y)
	u := r255.NewElement()
	u = u.Subtract(u1, u2)

	// 9. V = s*H - c*Gamma
	v1 := r255.NewElement()
	v1 = v1.ScalarMult(p.s, h)
	v2 := r255.NewElement()
	v2 = v2.ScalarMult(p.c, p.g)
	v := r255.NewElement()
	v = v.Subtract(v1, v2)

	// 7. c' = ECVRF_challenge_generation(Y, H, Gamma, U, V) (see Section 5.4.3)
	cc, err := GenerateChallenge(y, h, p.g, u, v)
	if err != nil {
		return nil, err
	}

	// 8. If c and c' are equal, output ("VALID", ECVRF_proof_to_hash(pi_string)); else output "INVALID"
	if p.c.Equal(cc) == 0 {
		return nil, &VerificationError{}
	}
	return p.Hash()
}

// https://github.com/C2SP/C2SP/blob/filippo/vrf-r255/vrf-r255.md#encode-to-curve
func encodeToCurve(salt []byte, alpha []byte) (h *r255.Element, err error) {
	x := toUniformBytes(salt, alpha)
	// 3. H = ristretto255_one_way_map(hash_string)
	h = r255.NewElement()
	h, err = h.SetUniformBytes(x)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// expose this step of encodeToCurve for the hash_string test vector
func toUniformBytes(salt []byte, alpha []byte) []byte {
	// 2. hash_string = Hash(suite_string || encode_to_curve_domain_separator || encode_to_curve_salt || alpha_string)
	u := sha512.New()
	u.Write([]byte(C2SP_SUITE_STRING))
	u.Write([]byte{encode_to_curve_ds})
	u.Write(salt)
	u.Write(alpha)
	x := u.Sum(nil)
	return x
}
