// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

/*
#include <stdlib.h>
#include <sqlite3.h>

// needed since sqlite3_column_text() and sqlite3_column_name()
// return const unsigned char* for some wack-a-doodle reason
const char *wsq_column_text(sqlite3_stmt *statement, int column)
{
	return (const char *) sqlite3_column_text(statement, column);
}
const char *wsq_column_name(sqlite3_stmt *statement, int column)
{
        return (const char *) sqlite3_column_name(statement, column);
}

// needed to work around the void(*)(void*) callback that is the
// last argument to sqlite3_bind_text(); SQLITE_TRANSIENT forces
// SQLite to make a private copy of the data
int wsq_bind_text(sqlite3_stmt *statement, int i, const char* text, int n)
{
	return sqlite3_bind_text(statement, i, text, n, SQLITE_TRANSIENT);
}

// needed to work around the ... argument of sqlite3_config(); if
// we ever require an option with parameters, we'll have to add more
// wrappers
int wsq_config(int option)
{
	return sqlite3_config(option);
}
*/
import "C"
import "unsafe"

// If something goes wrong on this level, we simply bomb
// out, there's no use trying to recover; note that most
// calls to sqlPanic() are for things that can never,
// ever, ever happen anyway. For regular "errors" status
// codes are returned.

func sqlPanic(str string) {
	panic("sqlite3 fatal error: " + str + "!")
}

// Wrappers around the most important SQLite types.

type sqlConnection struct {
	handle *C.sqlite3;
}

type sqlStatement struct {
	handle *C.sqlite3_stmt;
}

type sqlValue struct {
	handle *C.sqlite3_value;
}

type sqlBlob struct {
	handle *C.sqlite3_blob;
}

// Wrappers around the most important SQLite functions.

func sqlConfig(option int) int {
	return int(C.wsq_config(C.int(option)));
}

func sqlVersion() string {
	cp := C.sqlite3_libversion();
	if cp == nil {
		// The call can't really fail since it returns
		// a string constant, but let's be safe...
		sqlPanic("can't get library version");
	}
	return C.GoString(cp);
}

func sqlVersionNumber() int {
	return int(C.sqlite3_libversion_number());
}

func sqlSourceId() string {
	// SQLite 3.6.18 introduced sqlite3_sourceid(), see
	// http://www.hwaci.com/sw/sqlite/changes.html for
	// details; we can't expect wide availability yet,
	// for example Debian Lenny ships SQLite 3.5.9 only.
	if sqlVersionNumber() < 3006018 {
		return "unknown source id";
	}

	cp := C.sqlite3_sourceid();
	if cp == nil {
		// The call can't really fail since it returns
		// a string constant, but let's be safe...
		sqlPanic("can't get library sourceid");
	}
	return C.GoString(cp);
}

func sqlOpen(name string, flags int, vfs string) (conn *sqlConnection, rc int) {
	conn = new(sqlConnection);

	p := C.CString(name);
	if len(vfs) > 0 {
		q := C.CString(vfs);
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), q));
		C.free(unsafe.Pointer(q));
	} else {
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), nil))
	}
	C.free(unsafe.Pointer(p));

	// We could get a handle even if there's an error, see
	// http://www.sqlite.org/c3ref/open.html for details.
	// But we don't want to return a connection on error.
	if rc != StatusOk && conn.handle != nil {
		_ = conn.sqlClose();
		conn = nil;
	}

	return;
}

func (self *sqlConnection) sqlClose() int {
	return int(C.sqlite3_close(self.handle));
}

func (self *sqlConnection) sqlChanges() int {
	return int(C.sqlite3_changes(self.handle));
}

func (self *sqlConnection) sqlLastInsertRowId() int64 {
	return int64(C.sqlite3_last_insert_rowid(self.handle));
}

func (self *sqlConnection) sqlBusyTimeout(milliseconds int) int {
	return int(C.sqlite3_busy_timeout(self.handle, C.int(milliseconds)));
}

func (self *sqlConnection) sqlExtendedResultCodes(on bool) int {
	v := map[bool]int{true: 1, false: 0}[on];
	return int(C.sqlite3_extended_result_codes(conn.handle, C.int(v)));
}

func (self *sqlConnection) sqlErrorMessage() string {
	cp := C.sqlite3_errmsg();
	if cp == nil {
		// The call can't really fail since it returns
		// a string constant, but let's be safe...
		sqlPanic("can't get error message");
	}
	return C.GoString(cp);
}

func (self *sqlConnection) sqlErrorCode() int {
	return int(C.sqlite3_errcode(self.handle));
}

func (self *sqlConnection) sqlExtendedErrorCode() int {
	// SQLite 3.6.5 introduced sqlite3_extended_errcode(),
	// see http://www.hwaci.com/sw/sqlite/changes.html for
	// details; we can't expect wide availability yet, for
	// example Debian Lenny ships SQLite 3.5.9 only.
	if sqlVersionNumber() < 3006005 {
		// just return the regular error code...
		return self.sqlErrorCode();
	}
	return int(C.sqlite3_extended_errcode(self.handle));
}

func (self *sqlConnection) sqlPrepare(query string) (stat sqlStatement, rc int) {
	stat = new(sqlStatement);

	p := C.CString(query);
	// TODO: may need tail to process statement sequence? or at
	// least to generate an error that we missed some SQL?
	//
	// -1: process query until 0 byte
	// nil: don't return tail pointer
	rc = C.sqlite3_prepare_v2(self.handle, p, -1, &stat.handle, nil);
	C.free(unsafe.Pointer(p));

	// We are not supposed to get a handle on error. Since
	// sqlite3_open() follows a different rule, however, we
	// indulge in paranoia and check to make sure. We really
	// don't want to return a statement on error.
	if rc != StatusOk && stat.handle != nil {
		_ = stat.sqlFinalize();
		stat = nil;
	}

	return;
}

/*
func (self *sqliteConnection) prepare(query string) (statement sqliteStatement, error os.Error) {
	s := new(sqliteStatement);

	p := C.CString(query);
	rc := C.sqlite3_prepare_v2(self.handle, p, -1, &s.handle, nil);
	C.free(unsafe.Pointer(p));

	if rc != StatusOk {
		error = self.error();
		// did we get a handle anyway? if so we need to
		// finalize it, but that could trigger another,
		// secondary error; for now we ignore that one;
		// note that we shouldn't get a handle if there
		// was an error, that's what the docs say...
		if s.handle != nil {
			_ = C.sqlite3_finalize(s.handle)
		}
		return;
	}

	statement = s;
	return;
}


// Execute precompiled statement with given parameters
// (if any). The statement stays valid even if we fail
// to execute with given parameters.
//
// TODO: Figure out parameter stuff, right now all are
// TEXT parameters. :-/
func (self *Connection) ExecuteClassic(statement db.Statement, parameters ...) (cursor db.Cursor, error os.Error) {
	s, ok := statement.(*Statement);
	if !ok {
		error = &DriverError{"Execute: Not an sqlite3 statement!"};
		return;
	}

	p := reflect.NewValue(parameters).(*reflect.StructValue);

	if p.NumField() != int(C.sqlite3_bind_parameter_count(s.handle)) {
		error = &DriverError{"Execute: Number of parameters doesn't match!"};
		return;
	}

	pa := struct2array(p);

	for k, v := range pa {
		q := C.CString(v.(*reflect.StringValue).Get());
		rc := C.wsq_bind_text(s.handle, C.int(k+1), q, C.int(-1));
		C.free(unsafe.Pointer(q));

		if rc != StatusOk {
			error = self.error();
			s.clear();
			return;
		}
	}

	rc := C.sqlite3_step(s.handle);

	if rc != StatusDone && rc != StatusRow {
		// presumably any other outcome is an error
		error = self.error()
	}

	if rc == StatusRow {
		// statement is producing results, need a cursor
		c := new(Cursor);
		c.statement = s;
		c.connection = self;
		c.result = true;
		cursor = c;
	} else {
		// clean up after error or done
		s.clear()
	}

	return;
}



func (self *sqliteStatement) String() string {
	sql := C.sqlite3_sql(self.handle);
	return C.GoString(sql);
}

func (self *sqliteStatement) bindParameterCount() (int) {
	return int(C.sqlite3_bind_parameter_count(self.handle));
}

func (self *Statement) Close() (error os.Error) {
	rc := C.sqlite3_finalize(self.handle);
	if rc != StatusOk {
		error = self.connection.error()
	}
	return;
}

func (self *Statement) clear() (error os.Error) {
	rc := C.sqlite3_reset(self.handle);
	if rc == StatusOk {
		rc := C.sqlite3_clear_bindings(self.handle);
		if rc == StatusOk {
			return
		}
	}
	error = self.connection.error();
	return;
}

func (self *Cursor) MoreResults() bool	{ return self.result }

// Fetch another result. Once results are exhausted, the
// the statement that produced them will be reset and
// ready for another execution.
func (self *Cursor) FetchOne() (data []interface{}, error os.Error) {
	if !self.result {
		error = &DriverError{"FetchOne: No results to fetch!"};
		return;
	}

	// assemble results from current row
	nColumns := int(C.sqlite3_column_count(self.statement.handle));
	if nColumns <= 0 {
		error = &DriverError{"FetchOne: No columns in result!"};
		return;
	}
	data = make([]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		text := C.wsq_column_text(self.statement.handle, C.int(i));
		data[i] = C.GoString(text);
	}

	// try to get another row
	rc := C.sqlite3_step(self.statement.handle);

	if rc != StatusDone && rc != StatusRow {
		// presumably any other outcome is an error
		error = self.connection.error()
	}

	if rc == StatusDone {
		self.result = false;
		// clean up when done
		self.statement.clear();
	}

	return;
}

// Fetch at most count results. If we get no results at
// all, an error will be returned; otherwise it probably
// still occurred but will be hidden.
func (self *Cursor) FetchMany(count int) (data [][]interface{}, error os.Error) {
	d := make([][]interface{}, count);
	l := 0;
	var e os.Error;

	// grab at most count results
	for l < count {
		d[l], e = self.FetchOne();
		if e == nil {
			l += 1
		} else {
			break
		}
	}

	if l > 0 {
		// there were results
		if l < count {
			// but fewer than expected, need fresh copy
			data = make([][]interface{}, l);
			for i := 0; i < l; i++ {
				data[i] = d[i]
			}
		} else {
			data = d
		}
	} else {
		// no results at all, return the error
		error = e
	}

	return;
}

func (self *Cursor) FetchAll() (data [][]interface{}, error os.Error) {
	var v vector.Vector;
	var d interface{}
	var e os.Error;

	// grab results until error
	for {
		d, e = self.FetchOne();
		if e != nil {
			break
		}
		v.Push(d);
	}

	l := v.Len();

	if l > 0 {
		// TODO: how can this be done better?
		data = make([][]interface{}, l);
		for i := 0; i < l; i++ {
			data[i] = v.At(i).([]interface{})
		}
	} else {
		// no results at all, return the error
		error = e
	}

	return;
}

func (self *Cursor) FetchRow() (data map[string]interface{}, error os.Error) {
	if !self.result {
		error = &DriverError{"FetchRow: No results to fetch!"};
		return;
	}

	nColumns := int(C.sqlite3_column_count(self.statement.handle));
	if nColumns <= 0 {
		error = &DriverError{"FetchRow: No columns in result!"};
		return;
	}

	data = make(map[string]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		text := C.wsq_column_text(self.statement.handle, C.int(i));
		name := C.wsq_column_name(self.statement.handle, C.int(i));
		data[C.GoString(name)] = C.GoString(text);
	}

	// try to get another row
	rc := C.sqlite3_step(self.statement.handle);

	if rc != StatusDone && rc != StatusRow {
		// presumably any other outcome is an error
		error = self.connection.error()
	}

	if rc == StatusDone {
		self.result = false;
		// clean up when done
		self.statement.clear();
	}

	return;
}

*/
