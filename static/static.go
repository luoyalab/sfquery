package static

import (
	"io/fs"
	"sfquery"
)

// FS is the embedded static file system
var FS, _ = fs.Sub(sfquery.StaticFS, "web/static")
