package cryptopals

import "testing"
import "bytes"
import "math/rand"
import "time"
import "encoding/binary"

func TestProblem49(t *testing.T) {
	{
		makeTx, receiveTx := newCBCMACOracle("filippo")
		tx := makeTx("bob", 5)
		if v, err := receiveTx(tx); err != nil {
			t.Fatal(err)
		} else if v.Get("to") != "bob" {
			t.Fatal("wrong recipient", v.Get("to"))
		}
		tx[5] += 1
		if _, err := receiveTx(tx); err == nil {
			t.Fatal("modified tx was accepted")
		}

		if v, err := receiveTx(fakeCBCMACTx("bob", makeTx)); err != nil {
			t.Fatal(err)
		} else if v.Get("to") != "filippo" {
			t.Fatal("wrong recipient", v.Get("to"))
		} else if v.Get("from") != "bob" {
			t.Fatal("wrong sender", v.Get("from"))
		}
	}

	{
		targetTx, makeTx, receiveTx := newCBCMACMultiTxOracle("filippo")
		if _, err := receiveTx(targetTx); err != nil {
			t.Fatal(err)
		}
		tx := makeTx([]tx{{"bob", 5}})
		if mtx, err := receiveTx(tx); err != nil {
			t.Fatal(err)
		} else if to := mtx.txList[0].recipient; to != "bob" {
			t.Fatal("wrong recipient", to)
		}
		tx[5] += 1
		if _, err := receiveTx(tx); err == nil {
			t.Fatal("modified tx was accepted")
		}

		fakeTx := fakeCBCMACTxMulti(targetTx, makeTx)
		mtx, err := receiveTx(fakeTx)
		if err != nil {
			t.Fatal(err)
		}
		if mtx.source != "bob" {
			t.Fatal("wrong source", mtx.source)
		}
		var found bool
		for _, tx := range mtx.txList {
			t.Logf("%#v", tx)
			if tx.recipient == "filippo" {
				found = true
			}
		}
		if !found {
			t.Error("no tx for filippo found")
		}
	}
}

func TestProblem50(t *testing.T) {
	hash := decodeHex(t, "296b8d7cb78a243dda4d0a61d33bbdd1")
	key := []byte("YELLOW SUBMARINE")
	msg := []byte(`alert('Ayo, the Wu is back!');`)
	newMsg := fakeCBCMACHash(key, msg, hash)
	if !bytes.HasPrefix(newMsg, msg) {
		t.Error("new message does not include target code")
	}
	if !bytes.Equal(cbcMACHash(key, newMsg), hash) {
		t.Error("hash is not target hash")
	}
}

func TestProblem51(t *testing.T) {
	secret := []byte("TmV2ZXIgcmV2ZWFsIHRoZSBXdS1UYW5nIFNlY3JldCE=")
	oracle := newCompressionOracle(secret)
	newSecret := attackCompressionOracle(oracle)
	if !bytes.Equal(secret, newSecret) {
		t.Fail()
	}
}

func TestProblem52(t *testing.T) {
	bitLen := 32
	coll := findCollisions(16, bitLen)
	hash := shortHash(bitLen)
	var msg1, msg2 []byte
	for _, c := range coll {
		msg1 = append(msg1, c[rand.Intn(2)]...)
		msg2 = append(msg2, c[rand.Intn(2)]...)
	}
	if !bytes.Equal(hash(msg1), hash(msg2)) {
		t.Fatal("random messages don't collide")
	}

	bitLenF, bitLenG := 32, 40
	collConcat := findCollConcat(bitLenF, bitLenG)
	hashF := shortHash(bitLenF)
	if !bytes.Equal(hashF(collConcat[0]), hashF(collConcat[1])) {
		t.Fatal("not a short hash collision")
	}
	hashG := shortHash(bitLenG)
	if !bytes.Equal(hashG(collConcat[0]), hashG(collConcat[1])) {
		t.Fatal("not a long hash collision")
	}
}

func BenchmarkFindCollision(b *testing.B) {
	bitLen := 32
	for i := 0; i < b.N; i++ {
		findCollisions(1, bitLen)
	}
}

func TestProblem53(t *testing.T) {
	k, bitLen := 24, 32
	expMsg, finalState := expandableMessage(k, bitLen)
	hash := shortHash(bitLen)

	dummy := bytes.Repeat([]byte("*"), 16)
	msg := bytes.Repeat(dummy, expMsg[0].longBlockLen-1)
	msg = append(msg, expMsg[0].longLastBlock...)
	if !bytes.Equal(hash(msg), hash(expMsg[0].singleBlock)) {
		t.Fatal("first part is broken")
	}

	h := hash(expandMessage(expMsg, k))
	for i := 0; i < 10; i++ {
		hh := hash(expandMessage(expMsg, rand.Intn(1<<uint(k))+k-1))
		if !bytes.Equal(h, hh) {
			t.Fatal("expanded message hash does not match")
		}
	}

	msg = make([]byte, 1<<uint(k))
	rand.Read(msg)
	coll := preimageWithExpandableMessage(expMsg, finalState, msg, bitLen)
	if len(coll) != len(msg) {
		t.Error("the colliding message is not long the same, so the padding would mismatch")
	}
	if !bytes.Equal(hash(msg), hash(coll)) {
		t.Fatal("not a collision")
	}
	if bytes.Equal(msg, coll) {
		t.Fatal("same message")
	}
}

func TestProblem54(t *testing.T) {
	bitLen := 24
	hash := shortHash(bitLen)
	start := time.Now()
	root := makeNostradamusTree(8, bitLen) // cost: 2^8-1 * 2^bitLen
	t.Log("generated tree (2^8-1 collisions) in:", time.Since(start))

	start = time.Now()
	for i := 0; i < 1<<8; i++ {
		msg := make([]byte, 16*5)
		rand.Read(msg)
		msg = makeNostradamusPrediction(root, msg, bitLen) // cost: 2^(bitLen-8)
		if !bytes.Equal(hash(msg), root.nextState) {
			t.Error("wrong prediction")
		}
	}
	t.Log("generated 2^8 predictions in:", time.Since(start))
}

func TestProblem55(t *testing.T) {
	md4Block := func(p []byte) (a, b, c, d uint32) {
		a = _Init0
		b = _Init1
		c = _Init2
		d = _Init3
		var X [16]uint32

		j := 0
		for i := 0; i < 16; i++ {
			X[i] = uint32(p[j]) | uint32(p[j+1])<<8 | uint32(p[j+2])<<16 | uint32(p[j+3])<<24
			j += 4
		}

		// Round 1.
		for i := uint(0); i < 16; i++ {
			x := i
			s := shift1[i%4]
			f := ((c ^ d) & b) ^ d
			a += f + X[x]
			a = a<<s | a>>(32-s)
			a, b, c, d = d, a, b, c
		}

		return
	}

	msg := make([]byte, 64)
	rand.Read(msg)
	aa, bb, cc, dd := md4Block(msg)
	a, b, c, d := md4Round1(msg,
		func(i int, a, b, c, d, m, s, a0 uint32) uint32 { return a })
	if a != aa || b != bb || c != cc || d != dd {
		t.Fatal("round function is wrong")
	}

	if checkWangConditions(msg) {
		t.Fatal("check always returns true")
	}
	m1 := decodeHex(t, "4d7a9c8356cb927ab9d5a57857a7a5eede748a3"+
		"cdcc366b3b683a0203b2a5d9fc69d71b3f9e99198d79f805ea63bb"+
		"2e845dd8e3197e31fe52794bf08b9e8c3e9")
	reverseUint32Endian(m1) // grrrrrrr....
	h := decodeHex(t, "4d7e6a1defa93d2dde05b45d864c429b")
	md4 := NewMD4()
	md4.Write(m1)
	if !bytes.Equal(md4.checkSum(), h) {
		t.Fatal("wrong hash for paper msg")
	}
	if !checkWangConditions(m1) {
		t.Fatal("check false for paper msg")
	}

	enforceWangConditions(msg)
	if !checkWangConditions(msg) {
		t.Fatal("check false after enforcing")
	}

	m2 := decodeHex(t, "4d7a9c83d6cb927a29d5a57857a7a5eede748a3cdcc3"+
		"66b3b683a0203b2a5d9fc69d71b3f9e99198d79f805ea63bb2e8"+
		"45dc8e3197e31fe52794bf08b9e8c3e9")
	reverseUint32Endian(m2)
	wangSisterMsg(msg, m1)
	if !bytes.Equal(m2, msg) {
		t.Fatal("different sister message")
	}

	coll := searchMD4Collisions()
	t.Logf("m = %x", coll)
	coll1 := make([]byte, 64)
	wangSisterMsg(coll1, coll)
	t.Logf("m1 = %x", coll1)
	md4 = NewMD4()
	md4.Write(coll)
	h = md4.checkSum()
	t.Logf("h = %x", h)
	md4 = NewMD4()
	md4.Write(coll1)
	if !bytes.Equal(md4.checkSum(), h) {
		t.Fatal("wrong hash")
	}
}

func reverseUint32Endian(msg []byte) {
	for i := 0; i < len(msg); i += 4 {
		v := binary.BigEndian.Uint32(msg[i:])
		binary.LittleEndian.PutUint32(msg[i:], v)
	}
}

func TestProblem56(t *testing.T) {
	// mm := rc4Map(31)
	// for i := 0; i < 256; i++ {
	// 	t.Logf("% 4d: %0.6f", i, mm[byte(i)])
	// }
	secret := decodeBase64(t, "QkUgU1VSRSBUTyBEUklOSyBZT1VSIE9WQUxUSU5F")
	oracle := newRC4Oracle(secret)
	res := rc4ExploitBiases(oracle)
	if !bytes.Equal(res, secret) {
		t.Fail()
	}
}
