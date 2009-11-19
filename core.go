/*
	THIS IS NOT DONE AT ALL! USE AT YOUR OWN RISK!

	- it would be nice if cgo could grok several .go files,
	so far it can't; so all the C interface stuff has to be
	in one file; bummer that
*/

package sqlite3

/*
#include <stdlib.h>
#include "wrapper.h"
*/
import "C"
import "unsafe"

import "fmt"
import "os"

/* these are not exported yet since I am not sure they are needed */
const (
	sqliteOk = iota; /* Successful result */
	sqliteError; /* SQL error or missing database */
	sqliteInternal; /* Internal logic error in SQLite */
	sqlitePerm; /* Access permission denied */
	sqliteAbort; /* Callback routine requested an abort */
	sqliteBusy; /* The database file is locked */
	sqliteLocked; /* A table in the database is locked */
	sqliteNomem; /* A malloc() failed */
	sqliteReadonly; /* Attempt to write a readonly database */
	sqliteInterrupt; /* Operation terminated by sqlite3_interrupt()*/
	sqliteIoerr; /* Some kind of disk I/O error occurred */
	sqliteCorrupt; /* The database disk image is malformed */
	sqlite_Notfound; /* NOT USED. Table or record not found */
	sqliteFull; /* Insertion failed because database is full */
	sqliteCantopen; /* Unable to open the database file */
	sqliteProtocol; /* NOT USED. Database lock protocol error */
	sqliteEmpty; /* Database is empty */
	sqliteSchema; /* The database schema changed */
	sqliteToobig; /* String or BLOB exceeds size limit */
	sqliteConstraint; /* Abort due to constraint violation */
	sqliteMismatch; /* Data type mismatch */
	sqliteMisuse; /* Library used incorrectly */
	sqliteNolfs; /* Uses OS features not supported on host */
	sqliteAuth; /* Authorization denied */
	sqliteFormat; /* Auxiliary database format error */
	sqliteRange; /* 2nd parameter to sqlite3_bind out of range */
	sqliteNotadb; /* File opened that is not a database file */
	sqliteRow = 100; /* sqlite3_step() has another row ready */
	sqliteDone = 101;  /* sqlite3_step() has finished executing */
)

type Connection struct {
	handle C.wsq_db;
}

type Cursor struct {
	handle C.wsq_st;
	connection *Connection;
	result bool;
}

type Any interface{};
type ConnectionInfo map[string] Any;

func Open(info ConnectionInfo) (conn *Connection, error os.Error) {
	name, ok := info["name"];
	if !ok {
		error = &Error{0,0,"No name in connection info."};
		return;
	}
	flags, ok := info["sqlite.flags"];
	if !ok {
		flags = Any(0);
	}
	vfs, _ := info["sqlite.vfs"];

	conn = new(Connection);

	p := C.CString(name.(string));

	rc := sqliteOk;

	if vfs != nil {
		q := C.CString(vfs.(string));
		rc = int(C.wsq_open(p, &conn.handle, C.int(flags.(int)), q));
		C.free(unsafe.Pointer(q));
	}
	else {
		rc = int(C.wsq_open(p, &conn.handle, C.int(flags.(int)), nil));
	}

	C.free(unsafe.Pointer(p));

	if rc != sqliteOk {
		error = conn.error();
	}

	return;
}

func (self *Connection) error() (error os.Error) {
	e := new(Error);
	e.basic = int(C.wsq_errcode(self.handle));
	e.extended = int(C.wsq_extended_errcode(self.handle));
	e.message = C.GoString(C.wsq_errmsg(self.handle));
	return e;
}

func (self *Connection) Cursor() (cursor *Cursor, error os.Error) {
	cursor = new(Cursor);
	cursor.connection = self;
	return;
}

func (self *Connection) Close() (error os.Error) {
	rc := C.wsq_close(self.handle);
	if rc != 0 {
		error = self.error();
	}
	return;
}

func (self *Cursor) Execute(query string, parameters ...) (error os.Error) {
	query = fmt.Sprintf(query, parameters);

	q := C.CString(query);

	rc := C.wsq_prepare(self.connection.handle, q, -1, &self.handle, nil);
	if rc != sqliteOk {
		error = self.connection.error();
		if self.handle != nil {
			// TODO: finalize
		}
		return;
	}

	rc = C.wsq_step(self.handle);
	switch rc {
		case sqliteDone:
			self.result = false;
			// TODO: finalize
		case sqliteRow:
			self.result = true;
			// TODO: obtain results somehow? or later call?
		default:
			error = self.connection.error();
			// TODO: finalize
			return;
	}

	C.free(unsafe.Pointer(q));
	return;
}

func (self *Cursor) FetchOne() (data []interface{}, error os.Error) {
	if !self.result {
		error = &Error{0, 0, "No results to fetch!"};
		return;
	}

	nColumns := int(C.wsq_column_count(self.handle));
	if nColumns <= 0 {
		error = &Error{0, 0, "No columns!"};
		return;
	}

	data = make([]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		text := C.wsq_column_text(self.handle, C.int(i));
		data[i] = C.GoString(text);
	}

	rc := C.wsq_step(self.handle);
	switch rc {
		case sqliteDone:
			self.result = false;
			// TODO: finalize
		case sqliteRow:
			self.result = true;
		default:
			error = self.connection.error();
			// TODO: finalize
			return;
	}

	return;
}
func (self *Cursor) FetchRow() (data map[string]interface{}, error os.Error) {
	if !self.result {
		error = &Error{0, 0, "No results to fetch!"};
		return;
	}

	nColumns := int(C.wsq_column_count(self.handle));
	if nColumns <= 0 {
		error = &Error{0, 0, "No columns!"};
		return;
	}

	data = make(map[string]interface{}, nColumns);
	for i := 0; i < nColumns; i++ {
		text := C.wsq_column_text(self.handle, C.int(i));
		name := C.wsq_column_name(self.handle, C.int(i));
		data[C.GoString(name)] = C.GoString(text);
	}

	rc := C.wsq_step(self.handle);
	switch rc {
		case sqliteDone:
			self.result = false;
			// TODO: finalize
		case sqliteRow:
			self.result = true;
		default:
			error = self.connection.error();
			// TODO: finalize
			return;
	}

	return;
}

func (self *Cursor) Close() (error os.Error) {
	if self.handle != nil {
		rc := C.wsq_finalize(self.handle);
		if rc != sqliteOk {
			error = self.connection.error();
		}
	}
	return;
}
