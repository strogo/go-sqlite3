/*
	BRAINSTORMING ONLY! DON'T RELY ON THIS YET!
*/

package future_database_api_for_go

import "os"

/*
	Terminology: Database systems are pieces of software (usually
	outside of Go) that allow storage and retrieval of data. Note
	that we don't need to assume "relational" on the API level.
	Database interfaces are pieces of software (written in and for
	Go) that allow Go programs to interact with database systems
	through the use of some query language. Note that we don't
	need to assume "SQL" on the API level.
*/

/*
	Each database interface must provide an Open() function. The
	Open() function is used to establish a connection to a database
	system. The signature of Open() must be as follows:
*/

type OpenSignature func (args Arguments) (connection Connection, error os.Error)

/*
	Different database systems require a wide variety of parameters
	for the initial connection, which is why the Arguments type is
	as generic as possible:
*/

type Any interface {}
type Arguments map[string] Any

/*
	Client applications have to create suitable map and pass it
	to Open(). Each entry consists of a string key and a generic
	value. There are a number of well-known keys that apply to a
	wide variety of database systems:

	Name		Type	Description

	name		string	the database to connect to
	host		string	the host to connect to
	port		int	the port to connect to
	username	string	the user to connect as
	password	string	the password for that user

	For example, the following piece of code tries to connect to
	a MySQL database on the local machine at the default port:

	c, e := mysql.Open(Arguments{
		"name": "mydb",
		"username": "phf",
		"password": "somepassword"}
	)

	Note that defaults for all keys are specific to the database
	interface in question and should be documented there.

	The Open() function is free to ignore entries that it has no
	use for. For example, the sqlite3 interface only understands
	"name" and ignores the other well-known keys.

	A database interface is free to introduce additional keys if
	necessary, however those keys have to start with the package
	name of the database interface in question. For example, the
	sqlite3 interface supports the key "sqlite3.vfs".

	A successful call to Open() results in a Connection to the
	database system (there are several variations of this, but
	Connection is the most basic one):
*/

type Connection interface {
	Prepare(query string) (Statement, os.Error);
	Execute(statement Statement, parameters ...) (*Cursor, os.Error);
	Changes() (int, os.Error);
	Close() os.Error
}

type FancyConnection interface {
	Connection;
	ExecuteDirectly(query string, parameters ...) (*Cursor, os.Error)
}

type TransactionalConnection interface {
	Connection;
	Commit() os.Error;
	Rollback() os.Error
}

type Statement interface {
}

/*
	TODO
	Connections are use to execute queries. If a query has results
	that should be processed, Execute() returns a Cursor, otherwise
	it returns nil. If a query has no results but somehow affected
	the database, Changes() returns the number of changes. Note that
	the database interface has to explain what exactly constitutes
	a change for a given database system and query. Close() should
	be called if we are done communicating with the database system;
	after a connection has been closed, none of the other operations
	are allowed anymore.

	Queries that produced results return a Cursor to allow clients
	to iterate through the results (there are several variations of
	this, but Cursor is the most basic one):
*/

type Cursor interface {
	FetchOne() ([]interface {}, os.Error);
	FetchMany(count int) ([][]interface {}, os.Error);
	FetchAll() ([][]interface {}, os.Error);
	Close() os.Error
}

type InformativeCursor interface {
	Cursor;
	Description() (map[string]string, os.Error);
	Results() int;
};

type PythonicCursor interface {
	Cursor;
        FetchDict() (data map[string]interface{}, error os.Error);
        FetchManyDicts(count int) (data []map[string]interface{}, error os.Error);
        FetchAllDicts() (data []map[string]interface{}, error os.Error)
};

/*
	TODO
	Each result consists of a number of fields (in relational
	terminology, a result is a row and the fields are entries
	in each column).

	Description() returns a map from (the name of) a field to
	(the name of) its type. The exact format of field and type
	names is specified by the database interface in question.

	The Fetch() methods are used to returns results. You can mix
	and match, but if you want to know how many results you got
	in total you need to keep a running tally yourself.
	TODO
*/
