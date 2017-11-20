package cryptopals

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func makeCBCMAC(msg, iv []byte, b cipher.Block) []byte {
	msg = padPKCS7(msg, 16)
	return encryptCBC(msg, b, iv)[len(msg)-16:]
}

func checkCBCMAC(msg, iv []byte, b cipher.Block, mac []byte) bool {
	expected := makeCBCMAC(msg, iv, b)
	return hmac.Equal(expected, mac)

}

func newCBCMACOracle(ownerID string) (
	makeTx func(recipient string, amount int) []byte,
	receiveTx func(tx []byte) (url.Values, error),
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	makeTx = func(recipient string, amount int) []byte {
		v := url.Values{}
		v.Set("from", ownerID)
		v.Set("to", recipient)
		v.Set("xamount", strconv.Itoa(amount))
		msg := []byte(v.Encode())
		iv := make([]byte, 16)
		rand.Read(iv)
		mac := makeCBCMAC(msg, iv, b)
		msg = append(msg, iv...)
		msg = append(msg, mac...)
		return msg
	}
	receiveTx = func(tx []byte) (url.Values, error) {
		mac := tx[len(tx)-16:]
		iv := tx[len(tx)-32 : len(tx)-16]
		msg := tx[:len(tx)-32]
		if !checkCBCMAC(msg, iv, b, mac) {
			return nil, errors.New("wrong MAC")
		}
		v, err := url.ParseQuery(string(msg))
		if err != nil {
			return nil, err
		}
		return v, nil
	}
	return
}

func fakeCBCMACTx(target string, makeTx func(recipient string, amount int) []byte) []byte {
	tx := makeTx("foo", 1)
	msg := tx[:len(tx)-32]
	v, err := url.ParseQuery(string(msg))
	if err != nil {
		return nil
	}
	us := []byte(v.Get("from"))
	if len(us) < len(target) {
		return nil
	}
	tgt := append([]byte(target),
		bytes.Repeat([]byte("&"), len(us)-len(target))...)

	tx = makeTx(string(us), 10000000)
	iv := tx[len(tx)-32 : len(tx)-16]

	// from=filippo&to=filippo&amount=1000000
	// from=bob&&&&&to=filippo&amount=1000000
	tweak := xor(tx[:16], bytes.Replace(tx[:16], us, tgt, 1))
	copy(iv, xor(iv, tweak))
	copy(tx, xor(tx[:16], tweak))

	return tx
}

type tx struct {
	recipient string
	amount    int
}

type multiTx struct {
	source string
	txList []tx
}

func encodeMultiTx(mtx *multiTx) string {
	var txList string
	for i, tx := range mtx.txList {
		if strings.ContainsAny(tx.recipient, ";:") {
			panic("invalid tx")
		}
		if i != 0 {
			txList += ";"
		}
		txList += tx.recipient
		txList += ":"
		txList += strconv.Itoa(tx.amount)
	}
	v := url.Values{}
	v.Set("from", mtx.source)
	v.Set("tx_list", txList)
	return v.Encode()
}

func parseMultiTx(s string) *multiTx {
	v, err := url.ParseQuery(s)
	if err != nil {
		return nil
	}
	mtx := &multiTx{
		source: v.Get("from"),
	}
	for _, t := range strings.Split(v.Get("tx_list"), ";") {
		parts := strings.Split(t, ":")
		if len(parts) != 2 {
			continue
		}
		amount, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		mtx.txList = append(mtx.txList, tx{
			recipient: parts[0],
			amount:    amount,
		})
	}
	return mtx
}

func newCBCMACMultiTxOracle(ownerID string) (
	targetTx []byte, // intercepted via MitM
	makeTx func(txList []tx) []byte,
	receiveTx func(tx []byte) (*multiTx, error),
) {
	key := make([]byte, 16)
	rand.Read(key)
	b, _ := aes.NewCipher(key)
	iv := make([]byte, 16)

	targetTx = []byte(encodeMultiTx(&multiTx{
		source: "bob",
		txList: []tx{{"alice", 13}},
	}))
	targetTx = append(targetTx, makeCBCMAC(targetTx, iv, b)...)

	makeTx = func(txList []tx) []byte {
		msg := []byte(encodeMultiTx(&multiTx{
			source: ownerID,
			txList: txList,
		}))
		mac := makeCBCMAC(msg, iv, b)
		msg = append(msg, mac...)
		return msg
	}

	receiveTx = func(tx []byte) (*multiTx, error) {
		mac := tx[len(tx)-16:]
		msg := tx[:len(tx)-16]
		if !checkCBCMAC(msg, iv, b, mac) {
			return nil, errors.New("wrong MAC")
		}
		return parseMultiTx(string(msg)), nil
	}
	return
}

func fakeCBCMACTxMulti(txTgt []byte, makeTx func(txList []tx) []byte) []byte {
	m1 := txTgt[len(txTgt)-16:]

	// from=filippo&tx_ list=filippo%3A1 %3Bfilippo%3A999 9999999
	txUs := makeTx([]tx{
		{"filippo", 1},
		{"filippo", 9999999999},
	})
	if parseMultiTx(string(txUs[:len(txUs)-16])).source != "filippo" {
		panic("our account is hardcoded to filippo")
	}

	glue := xor(m1, txUs[:16])
	var newTx []byte
	newTx = append(newTx, padPKCS7(txTgt[:len(txTgt)-16], 16)...)
	newTx = append(newTx, glue...)
	newTx = append(newTx, txUs[16:]...)

	return newTx
}

func cbcMACHash(key, msg []byte) []byte {
	b, _ := aes.NewCipher(key)
	return makeCBCMAC(msg, make([]byte, 16), b)
}

func fakeCBCMACHash(key, msg, hash []byte) []byte {
	b, _ := aes.NewCipher(key)
	c1 := makeCBCMAC(msg, make([]byte, 16), b)

	buf := make([]byte, 16)
	copy(buf, hash)
	b.Decrypt(buf, buf)
	buf = xor(buf, []byte("/*************/\x01"))
	b.Decrypt(buf, buf)
	buf = xor(buf, c1)

	var res []byte
	res = append(res, padPKCS7(msg, 16)...)
	res = append(res, buf...)
	res = append(res, []byte("/*************/")...)
	return res
}

func newCompressionOracle(secret []byte) func(body []byte) []byte {
	return func(body []byte) []byte {
		request := []byte(fmt.Sprintf(`POST / HTTP/1.1
Host: hapless.com
Cookie: sessionid=%s
Content-Length: %d

%s
`, secret, len(body), body))

		var buf bytes.Buffer
		z := zlib.NewWriter(&buf)
		z.Write(request)
		z.Close()
		compressed := buf.Bytes()

		key := make([]byte, 16)
		rand.Read(key)
		b, _ := aes.NewCipher(key)
		iv := make([]byte, 16)
		stream := cipher.NewCTR(b, iv)
		stream.XORKeyStream(compressed, compressed)

		return append(iv, compressed...)
	}
}

func attackCompressionOracle(oracle func(body []byte) []byte) []byte {
	secret := []byte("sessionid=")
	n := len(oracle(secret))
	for len(secret) < len("sessionid=")+43 {
		secret = append(secret, '*')
		for i := 0; i < 256; i++ {
			secret[len(secret)-1] = byte(i)
			printProgress(secret, false)
			if len(oracle(secret)) == n {
				break
			}
			if i == 255 {
				panic("not found")
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\r%s=\n", secret)
	return append(secret[len("sessionid="):], '=')
}

func hashCore(prevState []byte, bitLen int) func(nextState, msgBlock []byte) {
	if len(prevState) != bitLen/8 {
		panic("wrong prevState length")
	}
	buf := make([]byte, 16)
	copy(buf, prevState)
	b, _ := aes.NewCipher(buf)
	return func(nextState, msgBlock []byte) {
		if len(nextState) != bitLen/8 {
			panic("wrong nextState length")
		}
		if len(msgBlock) != 16 {
			panic("wrong msgBlock length")
		}
		b.Encrypt(buf, msgBlock)
		copy(nextState, buf)
	}
}

func hashCoreToInt(prevState []byte, bitLen int) func(msgBlock []byte) uint64 {
	if len(prevState) != bitLen/8 {
		panic("wrong prevState length")
	}
	buf := make([]byte, 16)
	copy(buf, prevState)
	b, _ := aes.NewCipher(buf)
	mask := uint64(1)<<uint(bitLen) - 1
	return func(msgBlock []byte) uint64 {
		if len(msgBlock) != 16 {
			panic("wrong msgBlock length")
		}
		b.Encrypt(buf, msgBlock)
		return binary.LittleEndian.Uint64(buf) & mask
	}
}

func shortHash(bitLen int) func([]byte) []byte {
	if bitLen%8 != 0 {
		panic("bitLen must be a multiple of 8")
	}
	return func(msg []byte) []byte {
		state := bytes.Repeat([]byte("*"), bitLen/8)

		for len(msg) >= 16 {
			hashCore(state, bitLen)(state, msg[:16])
			msg = msg[16:]
		}

		finalMsg := make([]byte, 16)
		copy(finalMsg, msg)
		hashCore(state, bitLen)(state, finalMsg)

		return state
	}
}

func findCollisions(num, bitLen int) [][2][]byte {
	if bitLen > 64 {
		panic("bitLen too high")
	}
	var res [][2][]byte
	state := bytes.Repeat([]byte("*"), bitLen/8)
	for i := 0; i < num; i++ {
		outputs := make(map[uint64]string)
		block := make([]byte, 16)
		var nextState uint64

		b := hashCoreToInt(state, bitLen)
		randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))

		for {
			randAES.Encrypt(block, block)
			nextState = b(block)

			if _, ok := outputs[nextState]; ok {
				break
			}
			outputs[nextState] = string(block)
		}

		res = append(res, [2][]byte{block, []byte(outputs[nextState])})
		hashCore(state, bitLen)(state, block)
	}
	return res
}

func findCollConcat(bitLenF, bitLenG int) [2][]byte {
	if bitLenF > bitLenG {
		panic("f should be the shorter function")
	}

	collisions := findCollisions(bitLenG/2, bitLenF)
	fromSelector := func(selector uint64, msg []byte) {
		for j := 0; j < bitLenG/2; j++ {
			bit := (selector >> uint(j)) & 1
			copy(msg[16*j:], collisions[j][bit])
		}
	}

	hashG := shortHash(bitLenG)
	outputs := make(map[string]uint64)
	msg := make([]byte, 16*bitLenG/2)
	var found bool
	for selector := uint64(0); selector < 1<<uint(bitLenG/2); selector++ {
		fromSelector(selector, msg)
		hh := hashG(msg)
		h := string(hh)
		if _, ok := outputs[h]; ok {
			found = true
			break
		}
		outputs[h] = selector
	}
	if !found {
		println("retry")
		return findCollConcat(bitLenF, bitLenG)
	}

	msg2 := make([]byte, 16*bitLenG/2)
	fromSelector(outputs[string(hashG(msg))], msg2)
	return [2][]byte{msg2, msg}
}

type expMsgPiece struct {
	longBlockLen  int
	singleBlock   []byte
	longLastBlock []byte
}

func expandableMessage(k, bitLen int) ([]expMsgPiece, []byte) {
	state := bytes.Repeat([]byte("*"), bitLen/8)
	dummy := bytes.Repeat([]byte("*"), 16)

	var res []expMsgPiece
	for k := k; k > 0; k-- {
		outputs := make(map[uint64]string)
		var nextState uint64
		block := make([]byte, 16)
		randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))

		// Make a pool of single block message outputs.
		b := hashCoreToInt(state, bitLen)
		for i := 0; i < 1<<uint(bitLen/2); i++ {
			randAES.Encrypt(block, block)
			nextState = b(block)
			outputs[nextState] = string(block)
		}

		// Hash the first 2^(k-1) blocks into the long message.
		longMsgState := make([]byte, bitLen/8)
		copy(longMsgState, state)
		for i := 0; i < 1<<uint(k-1); i++ {
			hashCore(longMsgState, bitLen)(longMsgState, dummy)
		}

		// Find the last block of the long message.
		b = hashCoreToInt(longMsgState, bitLen)
		for {
			randAES.Encrypt(block, block)
			nextState = b(block)
			if _, ok := outputs[nextState]; ok {
				break
			}
		}

		hashCore(longMsgState, bitLen)(state, block)
		res = append(res, expMsgPiece{
			longBlockLen:  1<<uint(k-1) + 1,
			longLastBlock: block,
			singleBlock:   []byte(outputs[nextState]),
		})
	}
	return res, state
}

func expandMessage(pieces []expMsgPiece, blockLen int) []byte {
	k := len(pieces)
	if k > blockLen || blockLen > k+(1<<uint(k))-1 {
		panic("uncompatible blockLen")
	}
	msg := make([]byte, 0, blockLen*8)
	remaining := blockLen
	remaining -= k // collision blocks (single or last)
	for i := 0; i < k; i++ {
		extraBlocks := pieces[i].longBlockLen - 1
		if remaining >= extraBlocks {
			// Use the long message.
			dummy := bytes.Repeat([]byte("*"), 16)
			msg = append(msg, bytes.Repeat(dummy, extraBlocks)...)
			msg = append(msg, pieces[i].longLastBlock...)
			remaining -= extraBlocks
		} else {
			// Use the short message.
			msg = append(msg, pieces[i].singleBlock...)
		}
	}
	if len(msg) != blockLen*16 {
		panic("counted wrong!")
	}
	return msg
}

func preimageWithExpandableMessage(pieces []expMsgPiece, finalState []byte, msg []byte, bitLen int) []byte {
	intermediates := make(map[uint64]int)

	m := msg
	state := bytes.Repeat([]byte("*"), bitLen/8)
	for len(m) >= 16 {
		nextState := hashCoreToInt(state, bitLen)(m[:16])
		hashCore(state, bitLen)(state, m[:16])
		m = m[16:]
		intermediates[nextState] = len(msg) - len(m)
	}

	var nextState uint64
	block := make([]byte, 16)
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	b := hashCoreToInt(finalState, bitLen)
	for {
		randAES.Encrypt(block, block)
		nextState = b(block)
		if _, ok := intermediates[nextState]; ok {
			break
		}
	}

	coll := expandMessage(pieces, intermediates[nextState]/16-1)
	coll = append(coll, block...)
	coll = append(coll, msg[intermediates[nextState]:]...)
	return coll
}

func collideStates(a, b []byte, bitLen int) []byte {
	ba, bb := hashCoreToInt(a, bitLen), hashCoreToInt(b, bitLen)
	block := make([]byte, 16)
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	for {
		randAES.Encrypt(block, block)
		if ba(block) == bb(block) {
			break
		}
	}
	return block
}

type nostradamusNode struct {
	nextState []byte
	block     []byte
	children  [2]*nostradamusNode
}

func makeNostradamusTree(k, bitLen int) *nostradamusNode {
	var leaves []*nostradamusNode
	for i := 0; i < 1<<uint(k); i++ {
		state := make([]byte, bitLen/8)
		rand.Read(state)
		leaves = append(leaves, &nostradamusNode{
			nextState: state,
		})
	}
	for round := 0; round < k; round++ {
		var newLeaves []*nostradamusNode
		for i := 0; i < len(leaves)/2; i++ {
			a, b := leaves[i*2], leaves[i*2+1]
			block := collideStates(a.nextState, b.nextState, bitLen)
			state := make([]byte, bitLen/8)
			hashCore(a.nextState, bitLen)(state, block)
			newLeaves = append(newLeaves, &nostradamusNode{
				nextState: state,
				block:     block,
				children:  [2]*nostradamusNode{a, b},
			})
		}
		leaves = newLeaves
	}
	if len(leaves) != 1 {
		println(len(leaves))
		panic("I can't trees")
	}
	root := leaves[0]

	// Hash the padding into the root node final status.
	hashCore(root.nextState, bitLen)(root.nextState, make([]byte, 16))

	return root
}

func stateToUint64(state []byte) uint64 {
	buf := make([]byte, 16)
	copy(buf, state)
	return binary.LittleEndian.Uint64(buf)
}

func uint64ToState(n uint64, bitLen int) []byte {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf, n)
	state := make([]byte, bitLen/8)
	copy(state, buf)
	return state
}

func makeNostradamusPrediction(root *nostradamusNode, msg []byte, bitLen int) []byte {
	intermediates := make(map[uint64][][]byte)

	var exploreTree func(node *nostradamusNode, blocks [][]byte)
	exploreTree = func(node *nostradamusNode, blocks [][]byte) {
		if node.children[0] != nil {
			exploreTree(node.children[0], append(blocks, node.block))
			exploreTree(node.children[1], append(blocks, node.block))
		} else {
			blocksCopy := make([][]byte, len(blocks))
			copy(blocksCopy, blocks)
			intermediates[stateToUint64(node.nextState)] = blocksCopy
		}
	}
	exploreTree(root, nil)

	m := msg
	state := bytes.Repeat([]byte("*"), bitLen/8)
	for len(m) >= 16 {
		hashCore(state, bitLen)(state, m[:16])
		m = m[16:]
	}

	var nextState uint64
	block := make([]byte, 16)
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	b := hashCoreToInt(state, bitLen)
	for {
		randAES.Encrypt(block, block)
		nextState = b(block)
		if _, ok := intermediates[nextState]; ok {
			break
		}
	}

	msg = append(msg, block...)
	blocks := intermediates[nextState]
	for i := range blocks {
		msg = append(msg, blocks[len(blocks)-i-1]...)
	}
	return msg
}

func md4Round1(msg []byte, callback func(i int, a, b, c, d, m, s, a0 uint32) uint32) (a, b, c, d uint32) {
	a = 0x67452301
	b = 0xefcdab89
	c = 0x98badcfe
	d = 0x10325476

	var X [16]uint32
	for i := 0; i < 16; i++ {
		X[i] = uint32(msg[i*4]) | uint32(msg[i*4+1])<<8 |
			uint32(msg[i*4+2])<<16 | uint32(msg[i*4+3])<<24
	}

	f := func(a, b, c, d, m, s uint32) uint32 {
		f := (b & c) | (^b & d)
		a = a + f + m
		a = a<<s | a>>(32-s)
		return a
	}

	for i := 0; i < 4; i++ {
		aa := a
		a = f(a, b, c, d, X[i*4], 3)
		a = callback(i*4, a, b, c, d, X[i*4], 3, aa)
		dd := d
		d = f(d, a, b, c, X[i*4+1], 7)
		d = callback(i*4+1, d, a, b, c, X[i*4+1], 7, dd)
		cc := c
		c = f(c, d, a, b, X[i*4+2], 11)
		c = callback(i*4+2, c, d, a, b, X[i*4+2], 11, cc)
		bb := b
		b = f(b, c, d, a, X[i*4+3], 19)
		b = callback(i*4+3, b, c, d, a, X[i*4+3], 19, bb)
	}

	return
}

type wangConditionType int

const (
	w0 wangConditionType = iota // set to 0
	w1                          // set to 1
	wP                          // set to match previous variable
)

var wangConditions = [16][]struct {
	bitIdx uint // 1-INDEXED
	_type  wangConditionType
}{
	{{7, wP}},
	{{7, w0}, {8, wP}, {11, wP}},
	{{7, w1}, {8, w1}, {11, w0}, {26, wP}},
	{{7, w1}, {8, w0}, {11, w0}, {26, w0}},
	{{8, w1}, {11, w1}, {26, w0}, {14, wP}},
	{{14, w0}, {19, wP}, {20, wP}, {21, wP}, {22, wP}, {26, w1}},
	{{13, wP}, {14, w0}, {15, wP}, {19, w0}, {20, w0}, {21, w1}, {22, w0}},
	{{13, w1}, {14, w1}, {15, w0}, {17, wP}, {19, w0}, {20, w0}, {21, w0}, {22, w0}},
	{{13, w1}, {14, w1}, {15, w1}, {17, w0}, {19, w0}, {20, w0}, {21, w0}, {23, wP}, {22, w1}, {26, wP}},
	{{13, w1}, {14, w1}, {15, w1}, {17, w0}, {20, w0}, {21, w1}, {22, w1}, {23, w0}, {26, w1}, {30, wP}},
	{{17, w1}, {20, w0}, {21, w0}, {22, w0}, {23, w0}, {26, w0}, {30, w1}, {32, wP}},
	{{20, w0}, {21, w1}, {22, w1}, {23, wP}, {26, w1}, {30, w0}, {32, w0}},
	{{23, w0}, {26, w0}, {27, wP}, {29, wP}, {30, w1}, {32, w0}},
	{{23, w0}, {26, w0}, {27, w1}, {29, w1}, {30, w0}, {32, w1}},
	{{19, wP}, {23, w1}, {26, w1}, {27, w0}, {29, w0}, {30, w0}},
	{{19, w0}, {26, wP}, {27, w1}, {29, w1}, {30, w0}},
}

func checkWangConditions(msg []byte) bool {
	allValid := true
	check := func(i int, a, b, c, d, m, s, a0 uint32) uint32 {
		for _, cond := range wangConditions[i] {
			bit0 := (b >> (cond.bitIdx - 1)) & 1
			bit1 := (a >> (cond.bitIdx - 1)) & 1
			switch cond._type {
			case w0:
				if bit1 != 0 {
					allValid = false
				}
			case w1:
				if bit1 != 1 {
					allValid = false
				}
			case wP:
				if bit1 != bit0 {
					allValid = false
				}
			default:
				panic("invalid condition")
			}
		}
		return a
	}
	md4Round1(msg, check)
	return allValid
}

func enforceWangConditions(msg []byte) {
	enforce := func(i int, a, b, c, d, m, s, a0 uint32) uint32 {
		for _, cond := range wangConditions[i] {
			bit0 := (b >> (cond.bitIdx - 1)) & 1
			bit1 := (a >> (cond.bitIdx - 1)) & 1
			switch cond._type {
			case w0:
				if bit1 != 0 {
					a ^= 1 << (cond.bitIdx - 1)
				}
			case w1:
				if bit1 != 1 {
					a ^= 1 << (cond.bitIdx - 1)
				}
			case wP:
				if bit1 != bit0 {
					a ^= 1 << (cond.bitIdx - 1)
				}
			default:
				panic("invalid condition")
			}
		}
		m1 := a>>s | a<<(32-s)
		m1 -= a0
		m1 -= (b & c) | (^b & d)
		binary.LittleEndian.PutUint32(msg[i*4:], m1)
		return a
	}
	md4Round1(msg, enforce)
}

func wangSisterMsg(msg1, msg []byte) {
	// 4.1 The Collision Differential for MD4
	for i := 0; i < 16; i++ {
		v := binary.LittleEndian.Uint32(msg[i*4:])
		switch i {
		case 1:
			v += 1 << 31
		case 2:
			v += 1 << 31
			v -= 1 << 28
		case 12:
			v -= 1 << 16
		}
		binary.LittleEndian.PutUint32(msg1[i*4:], v)
	}
}

func searchMD4Collisions() []byte {
	msg, msg1 := make([]byte, 64), make([]byte, 64)
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	md4 := NewMD4()
	for {
		for i := 0; i < 4; i++ {
			randAES.Encrypt(msg[16*i:], msg[16*i:])
		}

		enforceWangConditions(msg)
		wangSisterMsg(msg1, msg)

		md4.Reset()
		md4.Write(msg)
		h0 := md4.checkSum()
		md4.Reset()
		md4.Write(msg1)
		h1 := md4.checkSum()
		if bytes.Equal(h0, h1) {
			return msg
		}
	}
}

func rc4Map(z int) map[byte]float64 {
	m := make(map[byte]uint32)
	buf, zero := make([]byte, z+1), make([]byte, z+1)
	key := make([]byte, 16)
	rand.Read(key)
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	total := 1 << 24
	for i := 0; i < total; i++ {
		randAES.Encrypt(key, key)
		c, _ := rc4.NewCipher(key)
		c.XORKeyStream(buf, zero)
		m[buf[z]]++
	}
	mm := make(map[byte]float64)
	for b, n := range m {
		mm[b] = float64(n) / float64(total)
	}
	return mm
}

func newRC4Oracle(secret []byte) func(req []byte) []byte {
	randAES, _ := aes.NewCipher([]byte("YELLOW SUBMARINE"))
	key := make([]byte, 16)
	rand.Read(key)
	return func(req []byte) []byte {
		data := []byte("/")
		data = append(data, req...)
		data = append(data, '\n')
		data = append(data, secret...)
		randAES.Encrypt(key, key)
		c, _ := rc4.NewCipher(key)
		c.XORKeyStream(data, data)
		return data
	}
}

func rc4ExploitBiases(oracle func(req []byte) []byte) []byte {
	total := 1 << 24
	z1 := 31
	var bias1 byte = 224
	z2 := 15
	var bias2 byte = 240

	res := make([]byte, 30)
	printProgress(res, false)
	var req []byte
	for i := 0; i < 16; i++ {

		var mm1, mm2 []map[byte]uint32
		var wg sync.WaitGroup
		for j := 0; j < runtime.GOMAXPROCS(0); j++ {
			wg.Add(1)
			m1 := make(map[byte]uint32)
			m2 := make(map[byte]uint32)
			mm1 = append(mm1, m1)
			mm2 = append(mm2, m2)
			go func(m1, m2 map[byte]uint32) {
				runtime.LockOSThread()
				total := total / runtime.GOMAXPROCS(0)
				for i := 0; i < total; i++ {
					buf := oracle(req)
					m1[buf[z1]]++
					m2[buf[z2]]++
				}
				wg.Done()
			}(m1, m2)
		}
		wg.Wait()
		m1 := make(map[byte]uint32)
		m2 := make(map[byte]uint32)
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			for b := range mm1[i] {
				m1[b] += mm1[i][b]
			}
			for b := range mm2[i] {
				m2[b] += mm2[i][b]
			}
		}

		var x byte
		var max uint32
		for b, n := range m1 {
			if n > max {
				x = b
				max = n
			}
		}
		res[z1-2-len(req)] = x ^ bias1
		for b, n := range m2 {
			if n > max {
				x = b
				max = n
			}
		}
		if z2-2-len(req) >= 0 {
			res[z2-2-len(req)] = x ^ bias2
		}

		printProgress(res, false)
		req = append(req, 'A')
	}
	printProgress(res, true)
	return res
}
