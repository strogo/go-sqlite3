/*
	THIS IS NOT DONE AT ALL! USE AT YOUR OWN RISK!
*/
package sqlite3

/*
	Error in the database interface itself, *not*
	the database system we talk to.
*/
type InterfaceError struct {
	message string;
}

/*
	Textual description of the error.
	Implements os.Error interface.
*/
func (self *InterfaceError) String() string {
	return self.message;
}

/*
	Error in the database system we talk to.
	SQLite has basic and extended error codes
	in addition to textual messages.
*/
type DatabaseError struct {
	message string;
	basic int;
	extended int;
}

/*
	Textual description of the error.
	Implements os.Error interface.
*/
func (self *DatabaseError) String() string {
	return self.message;
}

/*
	Basic SQLite error code. These are plain
	integers.
*/
func (self *DatabaseError) Basic() int {
	return self.basic;
}

/*
	Extended SQLite error code. These are OR'd
	together from various bits and pieces on top
	of basic error codes.
*/
func (self *DatabaseError) Extended() int {
	return self.extended;
}
