// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import (
	"db";
	"os";
	"reflect";
)

// SQLite connections
type Connection struct {
	handle *sqlConnection;
}

// Fill in a SystemError with information about
// the last error from SQLite.
func (self *Connection) error() (error os.Error) {
	e := new(SystemError);
	// Debian's SQLite 3.5.9 has no sqlite3_extended_errcode.
	// It's not really needed anyway if we ask SQLite to use
	// extended codes for the normal sqlite3_errcode() call;
	// we just have to mask out high bits to turn them back
	// into basic errors. :-D
	e.extended = self.handle.sqlErrorCode();
	e.basic = e.extended & 0xff;
	e.message = self.handle.sqlErrorMessage();
	return e;
}

// Precompile query into Statement.
func (self *Connection) Prepare(query string) (statement db.Statement, error os.Error) {
	s := new(Statement);
	s.connection = self;
	var rc int;
	s.handle, rc = self.handle.sqlPrepare(query)

	if rc != StatusOk {
		error = self.error();
		// did we get a handle anyway? if so we need to
		// finalize it, but that could trigger another,
		// secondary error; for now we ignore that one;
		// note that we shouldn't get a handle if there
		// was an error, that's what the docs say...
		if s.handle != nil {
			_ = s.handle.sqlFinalize();
		}
		return;
	}

	statement = s;
	return;
}

// stolen from fmt package, special-cases interface values
func getField(v *reflect.StructValue, i int) reflect.Value {
	val := v.Field(i);
	if i, ok := val.(*reflect.InterfaceValue); ok {
		if inter := i.Interface(); inter != nil {
			return reflect.NewValue(inter)
		}
	}
	return val;
}

func struct2array(s *reflect.StructValue) (r []interface{}) {
	l := s.NumField();
	r = make([]interface{}, l);
	for i := 0; i < l; i++ {
		r[i] = getField(s, i)
	}
	return;
}

// Execute precompiled statement with given parameters
// (if any). The statement stays valid even if we fail
// to execute with given parameters.
//
// TODO: Figure out parameter stuff, right now all are
// TEXT parameters. :-/
func (self *Connection) ExecuteClassic(statement db.Statement, parameters ...) (rset db.ClassicResultSet, error os.Error) {
	s, ok := statement.(*Statement);
	if !ok {
		error = &DriverError{"Execute: Not an sqlite3 statement!"};
		return;
	}

	p := reflect.NewValue(parameters).(*reflect.StructValue);

	if p.NumField() != s.handle.sqlBindParameterCount() {
		error = &DriverError{"Execute: Number of parameters doesn't match!"};
		return;
	}

	pa := struct2array(p);

	for k, v := range pa {
		q := v.(*reflect.StringValue).Get();
		rc := s.handle.sqlBindText(k, q);

		if rc != StatusOk {
			error = self.error();
			s.clear();
			return;
		}
	}

	rc := s.handle.sqlStep();

	if rc != StatusDone && rc != StatusRow {
		// presumably any other outcome is an error
		error = self.error()
	}

	if rc == StatusRow {
		// statement is producing results, need a cursor
		rs := new(ClassicResultSet);
		rs.statement = s;
		rs.connection = self;
		rs.more = true;
		rset = rs;
	} else {
		// clean up after error or done
		s.clear()
	}

	return;
}


func (self *Connection) Execute(statement db.Statement, parameters ...) (rs db.ResultSet, error os.Error) {
	var crs db.ClassicResultSet;
	crs, error = self.ExecuteClassic(statement, parameters);
	if error != nil {
		return
	}
	mrs := new(ResultSet);
	mrs.init(crs);
	rs = mrs;
	return;
}

func (self *Connection) Close() (error os.Error) {
	// TODO
	rc := self.handle.sqlClose();
	if rc != StatusOk {
		error = self.error()
	}
	return;
}

func (self *Connection) Changes() (changes int, error os.Error) {
	changes = self.handle.sqlChanges();
	return;
}

func (self *Connection) LastId() (id int64, error os.Error) {
	// TODO: really returns sqlite3_int64, what to do?
	id = self.handle.sqlLastInsertRowId();
	return;
}
