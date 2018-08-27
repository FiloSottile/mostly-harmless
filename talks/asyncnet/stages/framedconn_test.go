//+build ignore

package stages

import (
	"io/ioutil"
	"net"
	"testing"

	"golang.org/x/net/nettest"
)

// Note: synchronous, different from real TCP.
// TODO: actual localhost pipe?

func TestFramedConnection(t *testing.T) {
	t.Run("SimplePipe", func(t *testing.T) {
		p1, p2 := net.Pipe()
		c1, c2 := &framedConn{Conn: p1}, &framedConn{Conn: p2}
		done := make(chan struct{})
		go func() {
			_, err := c2.Write([]byte("Hello, Conn!"))
			if err != nil {
				t.Errorf("Write failed: %v", err)
			}
			err = c2.Close()
			if err != nil {
				t.Errorf("Close failed: %v", err)
			}
			close(done)
		}()
		buf, err := ioutil.ReadAll(c1)
		if err != nil {
			t.Errorf("ReadAll failed: %v", err)
		}
		if string(buf) != "Hello, Conn!" {
			t.Errorf("Expected %q; got %q", "Hello, Conn!", string(buf))
		}
		<-done
	})
	t.Run("nettest", func(t *testing.T) {
		nettest.TestConn(t, func() (c1, c2 net.Conn, stop func(), err error) {
			p1, p2 := net.Pipe()
			c1, c2 = &framedConn{Conn: p1}, &framedConn{Conn: p2}
			stop = func() {
				c1.Close()
				c2.Close()
			}
			return
		})
	})
}
