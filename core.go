// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// SQLite database driver for Go.
//
// Please see http://www.sqlite.org/c3ref/intro.html for all
// the missing details. Sorry, our documentation is focused
// on driver details, not on SQLite in general.
//
// Restrictions on Types:
//
// For now, we treat *all* values as strings. This is less of
// an issue for SQLite since it's typed dynamically anyway.
// Accepting/returning appropriate Go types is a high priority
// goal though.
//
// Binding Query Parameters:
//
// SQL queries can contain "?" parameter slots that are bound
// to values in Execute(). SQLite has more variations that we
// don't support for now: they would make the API more complex
// for very little gain. Parameter slots are matched to values
// in order of appearance.
//
// Concurrency:
//
// We still need to address concurrency issues in detail, for
// now we simply force SQLite into "serialized" threading mode
// (see http://www.sqlite.org/threadsafe.html for details).
//
//
// Low-Level API:
//
// The file low.go contains the low-level API for SQLite. The
// API is not exposed, and you should only have to worry about
// it if you're hunting for bugs in the database driver. Note
// that the technical reason for the low-level API is that cgo
// can't process multiple files at once (a factoid that really
// doesn't have any bearing whatsoever on applications).
package sqlite3

import (
	"db";
	"fmt";
	"http";
	"os";
	"reflect";
	"strconv";
)

// These constants can be or'd together and passed as the
// "flags" option to Open(). Some of them only apply if
// the "vfs" option is also passed. See SQLite documentation
// for details. Note that we always force OpenFullMutex,
// so passing OpenNoMutex has no effect. See also FlagURL().
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
	handle *sqlConnection;
}

// SQLite statements
type Statement struct {
	handle		*sqlStatement;
	connection	*Connection;
}

// SQLite version information
var Version db.VersionSignature
// SQLite connection factory
var Open db.OpenSignature

func init() {
	// Idiom to ensure that we actually conform
	// to the database API for functions as well.
	Version = version;
	Open = open;

	// Supposedly serialized mode is the default,
	// but let's make sure...
	rc := sqlConfig(configSerialized);
	if rc != StatusOk {
		panic("sqlite3 fatal error: can't switch to serialized mode")
	}
}

// The SQLite database interface returns keys "version",
// "sqlite3.sourceid", and "sqlite3.versionnumber"; the
// latter are specific to SQLite.
func version() (data map[string]string, error os.Error) {
	// TODO: fake client and server keys?
	data = make(map[string]string);
	data["version"] = sqlVersion();
	i := sqlVersionNumber();
	data["sqlite3.versionnumber"] = strconv.Itob(i, 10);
	data["sqlite3.sourceid"] = sqlSourceId();
	return;
}

func parseConnInfo(str string) (name string, flags int, vfs string, error os.Error) {
	var url *http.URL;

	url, error = http.ParseURL(str);
	if error != nil {
		return	// XXX really return error from ParseURL?
	}

	if len(url.Scheme) > 0 {
		if url.Scheme != "sqlite3" {
			error = &DriverError{fmt.Sprintf("Open: unknown scheme %s expected sqlite3", url.Scheme)};
			return;
		}
	}

	if len(url.Path) == 0 {
		error = &DriverError{"Open: no path or database name"};
		return;
	} else {
		name = url.Path
	}

	if len(url.RawQuery) > 0 {
		options, e := db.ParseQueryURL(url.RawQuery);
		if e != nil {
			error = e;
			return	// XXX really return error from ParseQueryURL?
		}
		rflags, ok := options["flags"];
		if ok {
			flags, error = strconv.Atoi(rflags);
			if error != nil {
				return	// XXX really return error from Atoi?
			}
		}
		vfs, ok = options["vfs"];
	}

	return;
}

func open(url string) (connection db.Connection, error os.Error) {
	var name string;
	var flags int;
	var vfs string;

	name, flags, vfs, error = parseConnInfo(url);
	if error != nil {
		return
	}

	// We want all connections to be in serialized threading
	// mode, so we fiddle with the flags to make sure.
	flags &^= OpenNoMutex;
	flags |= OpenFullMutex;

	conn := new(Connection);
	var rc int;
	conn.handle, rc = sqlOpen(name, flags, vfs);

	if rc != StatusOk {
		error = conn.error();
		// did we get a handle anyway? if so we need to
		// close it, but that could trigger another,
		// secondary error; for now we ignore that one
		if conn.handle != nil {
			_ = conn.Close();
		}
		return;
	}

	rc = conn.handle.sqlBusyTimeout(defaultTimeoutMilliseconds);
	if rc != StatusOk {
		error = conn.error();
		// ignore potential secondary error
		_ = conn.Close();
		return;
	}

	rc = conn.handle.sqlExtendedResultCodes(true);
	if rc != StatusOk {
		error = conn.error();
		// ignore potential secondary error
		_ = conn.Close();
		return;
	}

	connection = conn;
	return;
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

type Result struct {
	data	[]interface{};
	error	os.Error;
}

func (self *Result) Data() []interface{}	{ return self.data }

func (self *Result) Error() os.Error	{ return self.error }

func iterate(rset db.ClassicResultSet, channel chan<- db.Result) {
	for rset.More() {
		channel <- rset.Fetch();
	}
	rset.Close();
	close(channel);
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

/*
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






type ResultSet struct {
	// we implement everything in terms of classic stuff
	classic db.ClassicResultSet;
	// channel to send results through
	results chan db.Result;
	// channel to check for termination
	stops chan bool;
}

func (self *ResultSet) init(crs db.ClassicResultSet) {
	self.classic = crs;
	self.results = make(chan db.Result);
	self.stops = make(chan bool);
}

func (self *ResultSet) kill() {
	self.classic.Close();
	self.classic = nil;
	close(self.results);
	self.results = nil;
	close(self.stops);
	self.stops = nil;
}

// goroutine implementing the iterator
func (self *ResultSet) iterate() {
	for self.classic.More() {
		// block until either send or receive
		select {
		case self.results <- self.classic.Fetch():
//			fmt.Printf("sent to %s", self.results);
		case _ = <-self.stops:
//			fmt.Printf("received from %s", self.stops);
			break;
//		default:
//			fmt.Printf("debugging: no communication");
		}
	}
	self.kill();
}

func (self *ResultSet) Iter() <-chan db.Result {
	go self.iterate();
	return self.results;
}

func (self *ResultSet) Close() os.Error {
	if self.stops != nil {
		self.stops <- true;
	}
	return nil;
}

func (self *ResultSet) Names() []string {
	return nil;
}

func (self *ResultSet) Types() []string {
	return nil;
}
