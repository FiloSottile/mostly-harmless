package benchstatter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_formatGroup(t *testing.T) {
	input := "pkg:encoding/gob goos:darwin note:hw acceleration enabled foo:bar"
	want := `pkg: encoding/gob
goos: darwin
note: hw acceleration enabled
foo: bar`
	got := formatGroup(input)
	require.Equal(t, want, got)
}

func Test_csv2markdown(t *testing.T) {
	for _, td := range []struct {
		name string
		csv  string
		want []string
	}{
		{
			name: "basic",
			csv: `
foo,bar
baz,qux
`,
			want: []string{`
| foo | bar |
|-----|-----|
| baz | qux |
`},
		},
		{
			name: "double",
			csv: `
foo,bar
baz,qux

a,b
c,d
`,
			want: []string{
				`
| foo | bar |
|-----|-----|
| baz | qux |
`,
				`
| a | b |
|---|---|
| c | d |
`,
			},
		},
		{
			name: "escaped",
			csv: `
foo,bar
"b,q",foo
`,
			want: []string{`
| foo | bar |
|-----|-----|
| b,q | foo |
`},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			got, err := csv2Markdown([]byte(td.csv))
			require.NoError(t, err)
			for i, s := range td.want {
				td.want[i] = strings.TrimSpace(s)
			}
			for i, s := range got {
				got[i] = strings.TrimSpace(s)
			}
			require.Equal(t, td.want, got)
		})
	}
}
