// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "os";

type Result struct {
	data	[]interface{};
	error	os.Error;
}

func (self *Result) Data() []interface{}	{ return self.data }

func (self *Result) Error() os.Error	{ return self.error }
