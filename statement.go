// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "os";

// SQLite prepared statements.
type Statement struct {
	handle		*sqlStatement;
	connection	*Connection;
}

// Original query language string.
func (self *Statement) String() string {
	return self.handle.sqlSql();
}

// Free all associated resources. After a call to
// Close() the statement can not be used anymore.
// If results from the statement are still being
// processed, bad things will happen.
func (self *Statement) Close() (error os.Error) {
	rc := self.handle.sqlFinalize();
	if rc != StatusOk {
		error = self.connection.error()
	}
	self.handle = nil;
	self.connection = nil;
	return;
}

// Make the statement ready for re-binding parameters
// and re-execution.
func (self *Statement) clear() (error os.Error) {
	rc := self.handle.sqlReset();
	if rc == StatusOk {
		rc := self.handle.sqlClearBindings();
		if rc == StatusOk {
			return
		}
	}
	error = self.connection.error();
	return;
}
