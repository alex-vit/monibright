//go:generate go run gen_icon.go

package icon

import _ "embed"

//go:embed brightness.ico
var Data []byte
