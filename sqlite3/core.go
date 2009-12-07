// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// SQLite database driver for Go.
//
// Binding Query Parameters:
//
// We only support the simply "?" parameter slots in queries.
// SQLite has many more variations, but supporting them all
// would complicate the API immensly for very little gain.
// The parameter slots are matched to the arguments given in
// Execute() in order of appearance.
//
// Concurrency:
//
// We still need to address concurrency issues in detail, for
// now we simply force SQLite into "serialized" threading mode
// (see http://www.sqlite.org/threadsafe.html for details).
//
// XXX: it would be nice if cgo could grok several .go files,
// so far it can't; so all the C interface stuff has to be
// in one file; bummer that; an alternative would be to move
// all the high-level stuff out and keep a very low-level,
// mostly procedural API here; hmm...
//
// TODO: rename to "sqlite" instead of "sqlite3"?
package sqlite3

/*
#include <stdlib.h>
#include <sqlite3.h>

// needed since sqlite3_column_text() returns const unsigned char*
// for some wack-a-doodle reason
const char *wsq_column_text(sqlite3_stmt *statement, int column)
{
	return (const char *) sqlite3_column_text(statement, column);
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

import "db"

import "os"
import "strconv"
import "container/vector"
import "reflect"

// These constants can be or'd together and passed as the
// "sqlite3.flags" argument to Open(). Some of them only
// apply if "sqlite3.vfs" is also passed. See the SQLite
// documentation for details. Note that we always force
// OpenFullMutex, so passing OpenNoMutex has no effect.
const (
	OpenReadOnly		= 0x00000001;
	OpenReadWrite		= 0x00000002;
	OpenCreate		= 0x00000004;
	OpenDeleteOnClose	= 0x00000008;	// VFS only
	OpenExclusive		= 0x00000010;	// VFS only
	OpenMainDb		= 0x00000100;	// VFS only
	OpenTempDb		= 0x00000200;	// VFS only
	OpenTransientDb		= 0x00000400;	// VFS only
	OpenMainJournal		= 0x00000800;	// VFS only
	OpenTempJournal		= 0x00001000;	// VFS only
	OpenSubJournal		= 0x00002000;	// VFS only
	OpenMasterJournal	= 0x00004000;	// VFS only
	OpenNoMutex		= 0x00008000;
	OpenFullMutex		= 0x00010000;
	OpenSharedCache		= 0x00020000;
	OpenPrivateCache	= 0x00040000;
)

// Constants for sqlite3_config() used only internally.
// In fact only *one* is used. See SQLite documentation
// for details.
const (
	_	= iota;
	configSingleThread;
	configMultiThread;
	configSerialized;
	configMalloc;
	configGetMalloc;
	configScratch;
	configPageCache;
	configHeap;
	configMemStatus;
	configMutex;
	configGetMutex;
	_;
	configLookAside;
	configPCache;
	configGetPCache;
)

// after we run into a locked database/table,
// we'll retry for this long
const defaultTimeoutMilliseconds = 16 * 1000

// SQLite connections
type Connection struct {
	handle *C.sqlite3;
}

// SQLite statements
type Statement struct {
	handle		*C.sqlite3_stmt;
	connection	*Connection;
}

// SQLite cursors, will be renamed/refactored soon
type Cursor struct {
	statement	*Statement;
	connection	*Connection;
	result		bool;	// still have results left
}

// SQLite version information
var Version db.VersionSignature
// SQLite connection factory
var Open db.OpenSignature

func init() {
	Version = version;
	Open = open;

	// Supposedly serialized mode is the default,
	// but let's make sure...
	rc := C.wsq_config(configSerialized);
	if rc != StatusOk {
		panic("db/sqlite3 fatal error: can't switch to serialized mode")
	}
}

/*
	The SQLite database interface returns keys "version",
	"sqlite3.sourceid", and "sqlite3.versionnumber"; the
	latter are specific to SQLite.
*/
func version() (data map[string]string, error os.Error) {
	data = make(map[string]string);

	cp := C.sqlite3_libversion();
	if cp == nil {
		error = &DriverError{"Version: couldn't get library version!"};
		return;
	}
	data["version"] = C.GoString(cp);
	// TODO: fake client and server keys?

	i := C.sqlite3_libversion_number();
	data["sqlite3.versionnumber"] = strconv.Itob(int(i), 10);

	/*
		Debian's SQLite 3.5.9 has no sqlite3_sourceid,
		still need to find the correct version to cut
		off on.
	*/
	if i > 3005009 {
		cp = C.sqlite3_sourceid();
		if cp != nil {
			data["sqlite3.sourceid"] = C.GoString(cp)
		}
	}

	return;
}

type Any interface{}
type ConnectionInfo map[string]Any

func parseConnInfo(info ConnectionInfo) (name string, flags int, vfs *string, error os.Error) {
	ok := false;
	any := Any(nil);

	any, ok = info["name"];
	if !ok {
		error = &DriverError{"Open: No \"name\" in arguments map."};
		return;
	}
	name, ok = any.(string);
	if !ok {
		error = &DriverError{"Open: \"name\" argument not a string."};
		return;
	}

	any, ok = info["sqlite.flags"];
	if ok {
		flags, ok = any.(int);
		if !ok {
			error = &DriverError{"Open: \"flags\" argument not an int."};
			return;
		}
	}

	any, ok = info["sqlite.vfs"];
	if ok {
		vfs = new(string);
		*vfs, ok = any.(string);
		if !ok {
			error = &DriverError{"Open: \"vfs\" argument not a string."};
			return;
		}
	}

	return;
}

/* TODO: use URIs instead? http://golang.org/pkg/http/#URL */
func open(info ConnectionInfo) (connection db.Connection, error os.Error) {
	name, flags, vfs, error := parseConnInfo(info);
	if error != nil {
		return
	}

	// We want all connections to be in serialized threading
	// mode, so we fiddle with the flags to make sure.
	flags &^= OpenNoMutex;
	flags |= OpenFullMutex;

	conn := new(Connection);

	rc := StatusOk;
	p := C.CString(name);

	if vfs != nil {
		q := C.CString(*vfs);
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), q));
		C.free(unsafe.Pointer(q));
	} else {
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), nil))
	}

	connection = conn;

	C.free(unsafe.Pointer(p));
	if rc != StatusOk {
		error = conn.error();
		return;
	}

	rc = int(C.sqlite3_busy_timeout(conn.handle, defaultTimeoutMilliseconds));
	if rc != StatusOk {
		error = conn.error();
		return;
	}

	rc = int(C.sqlite3_extended_result_codes(conn.handle, C.int(1)));
	if rc != StatusOk {
		error = conn.error();
		return;
	}

	return;
}

/* === Connection === */

/*
	Fill in a DatabaseError with information about
	the last error from SQLite.
*/
func (self *Connection) error() (error os.Error) {
	e := new(DatabaseError);
	/*
		Debian's SQLite 3.5.9 has no sqlite3_extended_errcode.
		It's not really needed anyway if we ask SQLite to use
		extended codes for the normal sqlite3_errcode() call;
		we just have to mask out high bits to turn them back
		into basic errors. :-D
	*/
	e.extended = int(C.sqlite3_errcode(self.handle));
	e.basic = e.extended & 0xff;
	e.message = C.GoString(C.sqlite3_errmsg(self.handle));
	return e;
}

/*
	Precompile query into Statement.
*/
func (self *Connection) Prepare(query string) (statement db.Statement, error os.Error) {
	q := C.CString(query);
	s := new(Statement);
	s.connection = self;

	/* -1: process q until 0 byte, nil: don't return tail pointer */
	/* TODO: may need tail to process statement sequence? */
	rc := C.sqlite3_prepare_v2(self.handle, q, -1, &s.handle, nil);

	if rc != StatusOk {
		error = self.error();
		/*
			did we get a handle anyway? if so we need to
			finalize it, but that could trigger another,
			secondary error; for now we ignore that one;
			note that we shouldn't get a handle if there
			was an error, that's what the docs say...
		*/
		if s.handle != nil {
			_ = C.sqlite3_finalize(s.handle)
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

/*
	Execute precompiled statement with given parameters
	(if any). The statement stays valid even if we fail
	to execute with given parameters.

	TODO: Figure out parameter stuff, right now all are
	TEXT parameters. :-/
*/
func (self *Connection) Execute(statement db.Statement, parameters ...) (cursor db.Cursor, error os.Error) {
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
		/* presumably any other outcome is an error */
		error = self.error()
	}

	if rc == StatusRow {
		/* statement is producing results, need a cursor */
		c := new(Cursor);
		c.statement = s;
		c.connection = self;
		c.result = true;
		cursor = c;
	} else {
		/* clean up after error or done */
		s.clear()
	}

	return;
}

func iterate(cursor db.Cursor, channel chan<- db.Result) {
	var err os.Error;
	var data []interface{}
	var res db.Result;

	for cursor.MoreResults() {
		data, err = cursor.FetchOne();
		res.Data = data;
		res.Error = err;
		channel <- res;
	}

	cursor.Close();
	close(channel);
}

func (self *Connection) Iterate(statement db.Statement, parameters ...) (channel <-chan db.Result, error os.Error) {
	ch := make(chan db.Result);

	cur, error := self.Execute(statement, parameters);
	if error != nil {
		return
	}

	go iterate(cur, ch);

	channel = ch;
	return;
}

func (self *Connection) Close() (error os.Error) {
	/* TODO */
	rc := C.sqlite3_close(self.handle);
	if rc != StatusOk {
		error = self.error()
	}
	return;
}

/* === Statement === */

func (self *Statement) String() string {
	sql := C.sqlite3_sql(self.handle);
	return C.GoString(sql);
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

/* === Cursor === */

func (self *Cursor) MoreResults() bool	{ return self.result }

/*
	Fetch another result. Once results are exhausted, the
	the statement that produced them will be reset and
	ready for another execution.
*/
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

/*
	Fetch at most count results. If we get no results at
	all, an error will be returned; otherwise it probably
	still occurred but will be hidden.
*/
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

func (self *Cursor) Close() os.Error {
	/*
		Hmmm... There's really nothing to do since
		we want the statement to stay around. Should
		we reset it here?
	*/
	return nil
}

/*
func (self *Cursor) FetchRow() (data map[string]interface{}, error os.Error) {
	if !self.result {
		error = &DriverError{"FetchRow: No results to fetch!"};
		return;
	}

	nColumns := int(C.sqlite3_column_count(self.handle));
	if nColumns <= 0 {
		error = &DriverError{"FetchRow: No columns in result!"};
		return;
	}

	data = make(map[string]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		text := C.sqlite3_column_text(self.handle, C.int(i));
		name := C.sqlite3_column_name(self.handle, C.int(i));
		data[C.GoString(name)] = C.GoString(text);
	}

	rc := C.sqlite3_step(self.handle);
	switch rc {
		case StatusDone:
			self.result = false;
			// TODO: finalize
		case StatusRow:
			self.result = true;
		default:
			error = self.connection.error();
			// TODO: finalize
			return;
	}

	return;
}
*/
