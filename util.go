// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "fmt"

// FlagsURL() is a helper to turn the various OpenXYZ option
// flags into the "flags=123456789" notation required for
// the URL passed to Open(). It's a shame that we have to
// go from int to string and back to int, but thus is the
// price of generality.
func FlagsURL(options int) string	{ return fmt.Sprintf("flags=%d", options) }
