/*
	THIS IS NOT DONE AT ALL! USE AT YOUR OWN RISK!

	- it would be nice if cgo could grok several .go files,
	so far it can't; so all the C interface stuff has to be
	in one file; bummer that
*/

package sqlite3

/*
#include <stdlib.h>
#include <sqlite3.h>
const char *wsq_column_text(sqlite3_stmt *statement, int column)
{
	return (const char *) sqlite3_column_text(statement, column);
}
*/
import "C"
import "unsafe"

import "db"

import "os"
import "strconv"

/*
	These constants can be or'd together and passed as the
	"sqlite3.flags" argument to Open(). Some of them only
	apply if "sqlite3.vfs" is also passed. See the SQLite
	documentation for details.
*/
const (
	OpenReadOnly = 0x00000001;
	OpenReadWrite = 0x00000002;
	OpenCreate = 0x00000004;
	OpenDeleteOnClose = 0x00000008; /* VFS only */
	OpenExclusive = 0x00000010; /* VFS only */
	OpenMainDb = 0x00000100; /* VFS only */
	OpenTempDb = 0x00000200; /* VFS only */
	OpenTransientDb = 0x00000400; /* VFS only */
	OpenMainJournal = 0x00000800; /* VFS only */
	OpenTempJournal = 0x00001000; /* VFS only */
	OpenSubJournal = 0x00002000; /* VFS only */
	OpenMasterJournal = 0x00004000; /* VFS only */
	OpenNoMutex = 0x00008000;
	OpenFullMutex = 0x00010000;
	OpenSharedCache = 0x00020000;
	OpenPrivateCache = 0x00040000;
)

/* after we run into a lock, we'll retry for this long */
const defaultTimeoutMilliseconds = 16*1000;

/* SQLite connections */
type Connection struct {
	/* pointer to struct sqlite3 */
	handle *C.sqlite3;
}

/* SQLite statements */
type Statement struct {
	/* pointer to struct sqlite3_stmt */
	handle *C.sqlite3_stmt;
	/* connection we were created on */
	connection *Connection;
}

/* SQLite cursors, will be renamed/refactored soon */
type Cursor struct {
	/* statement we were created for */
	statement *Statement;
	/* connection we were created on */
	connection *Connection;
	/* the last query yielded results */
	result bool;
}

/* idiom to ensure that signatures are exactly as specified in db */
var Version db.VersionSignature;
var Open db.OpenSignature;
func init() {
	Version = version;
	Open = open;
}

/*
	The SQLite database interface returns keys "version",
	"sqlite3.sourceid", and "sqlite3.versionnumber"; the
	latter are specific to SQLite.
*/
func version() (data map[string]string, error os.Error) {
	data = make(map[string]string);

	cp := C.sqlite3_libversion();
	if (cp == nil) {
		error = &InterfaceError{"Version: couldn't get library version!"};
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
		if (cp != nil) {
			data["sqlite3.sourceid"] = C.GoString(cp);
		}
	}

	return;
}

type Any interface{};
type ConnectionInfo map[string] Any;

func parseConnInfo(info ConnectionInfo) (name string, flags int, vfs *string, error os.Error) {
	ok := false;
	any := Any(nil);

	any, ok = info["name"];
	if !ok {
		error = &InterfaceError{"Open: No \"name\" in arguments map."};
		return;
	}
	name, ok = any.(string);
	if !ok {
		error = &InterfaceError{"Open: \"name\" argument not a string."};
		return;
	}

	any, ok = info["sqlite.flags"];
	if ok {
		flags, ok = any.(int);
		if !ok {
			error = &InterfaceError{"Open: \"flags\" argument not an int."};
			return;
		}
	}

	any, ok = info["sqlite.vfs"];
	if ok {
		vfs = new(string);
		*vfs, ok = any.(string);
		if !ok {
			error = &InterfaceError{"Open: \"vfs\" argument not a string."};
			return;
		}
	}

	return;
}

/* TODO: use URIs instead? http://golang.org/pkg/http/#URL */
func open(info ConnectionInfo) (connection db.Connection, error os.Error)
{
	name, flags, vfs, error := parseConnInfo(info);
	if error != nil {
		return;
	}

	conn := new(Connection);

	rc := StatusOk;
	p := C.CString(name);

	if vfs != nil {
		q := C.CString(*vfs);
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), q));
		C.free(unsafe.Pointer(q));
	}
	else {
		rc = int(C.sqlite3_open_v2(p, &conn.handle, C.int(flags), nil));
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
func (self *Connection) Prepare(query string) (statement db.Statement, error os.Error)
{
	q := C.CString(query);
	s := new(Statement);
	s.connection = self;

	/* -1: process q until 0 byte, nil: don't return tail pointer */
	/* TODO: may need tail to process statement sequence? */
	rc := C.sqlite3_prepare(self.handle, q, -1, &s.handle, nil);

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
			_ = C.sqlite3_finalize(s.handle);
		}
		return;
	}

	statement = s;
	return;
}

/*
	Execute precompiled Statement with given parameters
	(if any). The statement stays valid even if we fail
	to execute with given parameters.

	TODO: Figure out parameter stuff, right now all are
	TEXT parameters. :-/
*/
func (self *Connection) Execute(statement db.Statement, parameters ...) (cursor db.Cursor, error os.Error)
{
	s, ok := statement.(*Statement);
	if !ok {
		error = &InterfaceError{"Execute: Not an sqlite3 statement!"};
		return;
	}

	/* TODO: bind parameters! */

	rc := C.sqlite3_step(s.handle);

	if rc != StatusDone && rc != StatusRow {
		/* presumably any other outcome is an error */
		error = self.error();
	}

	if rc == StatusRow {
		/* statement is producing results, need a cursor */
		c := new(Cursor);
		c.statement = s;
		c.connection = self;
		c.result = true;
		cursor = c;
	}
	else {
		/* clean up after error or done */
		C.sqlite3_reset(s.handle);
		C.sqlite3_clear_bindings(s.handle);
	}

	return;
}

func (self *Connection) Close() (error os.Error) {
	/* TODO */
	rc := C.sqlite3_close(self.handle);
	if rc != StatusOk {
		error = self.error();
	}
	return;
}

/* === Statement === */

func (self *Statement) Close() (error os.Error) {
	rc := C.sqlite3_finalize(self.handle);
	if rc != StatusOk {
		error = self.connection.error();
	}
	return;
}

/* === Cursor === */

func (self *Cursor) MoreResults() bool
{
	return self.result;
}

/*
	Fetch another result. Once results are exhausted, the
	the statement that produced them will be reset and
	ready for another execution.
*/
func (self *Cursor) FetchOne() (data []interface {}, error os.Error)
{
	if !self.result {
		error = &InterfaceError{"FetchOne: No results to fetch!"};
		return;
	}

	// assemble results from current row
	nColumns := int(C.sqlite3_column_count(self.statement.handle));
	if nColumns <= 0 {
		error = &InterfaceError{"FetchOne: No columns in result!"};
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
		error = self.connection.error();
	}

	if rc == StatusDone {
		self.result = false;
		// clean up when done
		C.sqlite3_reset(self.statement.handle);
		C.sqlite3_clear_bindings(self.statement.handle);
	}

	return;
}

/*
	Fetch at most count results. If we get no results at
	all, an error will be returned; otherwise it probably
	still occurred but will be hidden.
*/
func (self *Cursor) FetchMany(count int) (data [][]interface {}, error os.Error) {
	d := make([][]interface{}, count);
	i := 0;
	var e os.Error;

	// grab at most count results
	for i < count {
		d[i], e = self.FetchOne();
		if e == nil {
			i += 1;
		}
		else {
			break;
		}
	}

	if i > 0 {
		// there were results
		if i < count {
			// but fewer than expected, need fresh copy
			data = make([][]interface{}, i);
			j := 0;
			for j < i {
				data[j] = d[j];
				j++;
			}
		}
		else {
			data = d;
		}
	}
	else {
		// no results at all, return the error
		error = e;
	}

	return;
}

func (self *Cursor) FetchAll() ([][]interface {}, os.Error)
{
	return nil, nil;
}

func (self *Cursor) Close() os.Error
{
	/*
		Hmmm... There's really nothing to do since
		we want the statement to stay around. Should
		we reset it here?
	*/
	return nil;
}

/*
func (self *Cursor) FetchRow() (data map[string]interface{}, error os.Error) {
	if !self.result {
		error = &InterfaceError{"FetchRow: No results to fetch!"};
		return;
	}

	nColumns := int(C.sqlite3_column_count(self.handle));
	if nColumns <= 0 {
		error = &InterfaceError{"FetchRow: No columns in result!"};
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
