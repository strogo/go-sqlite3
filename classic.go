// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

// We use the "classic" stuff without channels to implement
// the nicer, more Go-like channel-based stuff. Officially
// the "classic" API is optional, but we really need it. :-D

import (
	"db";
	"os";
	"reflect";
)

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

// TODO
type ClassicResultSet struct {
	statement	*Statement;
	connection	*Connection;
	more		bool;	// still have results left
}

// TODO
func (self *ClassicResultSet) More() bool {
	return self.more;
}

// Fetch another result. Once results are exhausted, the
// the statement that produced them will be reset and
// ready for another execution.
func (self *ClassicResultSet) Fetch() (result db.Result) {
	res := new(Result);
	result = res;

	if !self.more {
		res.error = &DriverError{"Fetch: No result to fetch!"};
		return;
	}

	// assemble results from current row
	nColumns := self.statement.handle.sqlColumnCount();
	if nColumns <= 0 {
		res.error = &DriverError{"Fetch: No columns in result!"};
		return;
	}
	res.data = make([]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		res.data[i] = self.statement.handle.sqlColumnText(i);
	}

	// try to get another row
	rc := self.statement.handle.sqlStep();

	if rc != StatusDone && rc != StatusRow {
		// presumably any other outcome is an error
		// TODO: is res.error the right place?
		res.error = self.connection.error()
	}

	if rc == StatusDone {
		self.more = false;
		// clean up when done
		self.statement.clear();
	}

	return;
}

// TODO
// TODO: reset statement here as well, just like in Fetch
func (self *ClassicResultSet) Close() os.Error {
	return nil;
}
