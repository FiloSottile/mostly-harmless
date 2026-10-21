package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitMessageRepoHandler serves over smart HTTP a repository with a single
// commit, containing a README with msg. msg is also sent as a progress message
// during the clone, which git clients print as "remote: <msg>".
//
// Only fresh clones are supported: the upload-pack request is ignored and the
// whole repository is always sent. Clients requesting protocol v2 fall back to
// v0, which is the only one implemented.
func GitMessageRepoHandler(msg string) http.Handler {
	commit, pack := gitMessageRepo(msg)
	refs := pktLine("# service=git-upload-pack\n") + "0000" +
		pktLine(commit+" HEAD\x00side-band-64k symref=HEAD:refs/heads/main "+
			"object-format=sha1 agent=filippo.fly.dev\n") +
		pktLine(commit+" refs/heads/main\n") + "0000"

	result := pktLine("NAK\n") + pktLine("\x02"+msg+"\n")
	for len(pack) > 0 {
		n := min(len(pack), 65515)
		result += pktLine("\x01" + string(pack[:n]))
		pack = pack[n:]
	}
	result += "0000"

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Cache-Control", "no-cache")
		if strings.HasSuffix(r.URL.Path, "/git-upload-pack") {
			rw.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			io.WriteString(rw, result)
			return
		}
		if r.URL.Query().Get("service") != "git-upload-pack" {
			http.Error(rw, "only git-upload-pack is supported", http.StatusForbidden)
			return
		}
		rw.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		io.WriteString(rw, refs)
	})
}

func pktLine(s string) string {
	return fmt.Sprintf("%04x", len(s)+4) + s
}

// gitMessageRepo returns the hex ID of a commit containing a README with msg,
// and a packfile with the commit, its tree, and the README blob.
func gitMessageRepo(msg string) (commitID string, pack []byte) {
	const (
		objCommit = 1
		objTree   = 2
		objBlob   = 3
	)
	type object struct {
		typ  int
		data []byte
	}
	id := func(o object) []byte {
		name := map[int]string{objCommit: "commit", objTree: "tree", objBlob: "blob"}[o.typ]
		h := sha1.New()
		fmt.Fprintf(h, "%s %d\x00", name, len(o.data))
		h.Write(o.data)
		return h.Sum(nil)
	}

	blob := object{objBlob, []byte(msg + "\n")}
	tree := object{objTree, append([]byte("100644 README\x00"), id(blob)...)}
	commit := object{objCommit, []byte("tree " + hex.EncodeToString(id(tree)) + "\n" +
		"author Filippo Valsorda <filippo@example.com> 1760000000 +0000\n" +
		"committer Filippo Valsorda <filippo@example.com> 1760000000 +0000\n" +
		"\n" + msg + "\n")}

	buf := &bytes.Buffer{}
	buf.WriteString("PACK")
	binary.Write(buf, binary.BigEndian, uint32(2))
	binary.Write(buf, binary.BigEndian, uint32(3))
	for _, o := range []object{commit, tree, blob} {
		size := len(o.data)
		b := byte(o.typ<<4) | byte(size&0x0f)
		for size >>= 4; size > 0; size >>= 7 {
			buf.WriteByte(b | 0x80)
			b = byte(size & 0x7f)
		}
		buf.WriteByte(b)
		z := zlib.NewWriter(buf)
		z.Write(o.data)
		z.Close()
	}
	sum := sha1.Sum(buf.Bytes())
	buf.Write(sum[:])

	return hex.EncodeToString(id(commit)), buf.Bytes()
}
