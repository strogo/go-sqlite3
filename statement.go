// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "os";

// SQLite statements
type Statement struct {
	handle		*sqlStatement;
	connection	*Connection;
}

func (self *Statement) String() string {
	sql := self.handle.sqlSql();
	return sql;
}

func (self *Statement) Close() (error os.Error) {
	rc := self.handle.sqlFinalize();
	if rc != StatusOk {
		error = self.connection.error()
	}
	return;
}

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
