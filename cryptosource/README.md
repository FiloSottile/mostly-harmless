`math/rand` outputs are predictable, which can cause unexpected security
issues in a number of scenarios. Because of that, it's best practice to
default to `crypto/rand` unless a strong reason to use `math/rand` is present.

However, `crypto/rand` doesn't provide the same helper functions as `math/rand`.

Package `cryptosource` provides a `crypto/rand`-backed `math/rand.Source`, so
the convenient `math/rand` methods can be used with the security of `crypto/rand`.

```go
package main

import (
	"fmt"
	"math/rand"

	"github.com/FiloSottile/mostly-harmless/cryptosource"
)

func main() {
	r := rand.New(cryptosource.New())
	fmt.Println(r.Float32())
}
```

[Try this on the Go Playground](https://play.golang.org/p/QurtNQbNHRs).

The code is released in the Public Domain (and comes with a
fallback license with no requirements) so you can just copy-paste
it if you want. It's short.
