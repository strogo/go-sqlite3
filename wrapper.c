#include <sqlite3.h>
#include "wrapper.h"

int wsq_open(const char *name, wsq_db *database, int flags, const char *vfs)
{
	return sqlite3_open_v2(name, (sqlite3**) database, flags, vfs);
}

int wsq_prepare(wsq_db database, const char *sql, int length, wsq_st *statement, const char **tail)
{
	return sqlite3_prepare_v2(database, sql, length, (sqlite3_stmt**) statement, tail);
}

int wsq_step(wsq_st statement)
{
	return sqlite3_step(statement);
}

int wsq_column_count(wsq_st statement)
{
	return sqlite3_column_count(statement);
}

int wsq_column_type(wsq_st statement, int column)
{
	return sqlite3_column_type(statement, column);
}

const char *wsq_column_name(wsq_st statement, int column)
{
	return sqlite3_column_name(statement, column);
}

const char *wsq_column_text(wsq_st statement, int column)
{
	return sqlite3_column_text(statement, column);
}

int wsq_finalize(wsq_st statement)
{
	return sqlite3_finalize(statement);
}

int wsq_close(wsq_db database)
{
	return sqlite3_close(database);
}

int wsq_errcode(wsq_db database)
{
	return sqlite3_errcode(database);
}

int wsq_extended_errcode(wsq_db database)
{
	return sqlite3_extended_errcode(database);
}

const char *wsq_errmsg(wsq_db database)
{
	return sqlite3_errmsg(database);
}

const char *wsq_libversion(void)
{
	return sqlite3_libversion();
}

const char *wsq_sourceid(void)
{
	return sqlite3_sourceid();
}

int wsq_libversion_number(void)
{
	return sqlite3_libversion_number();
}

int wsq_busy_timeout(wsq_db database, int milliseconds)
{
	return sqlite3_busy_timeout(database, milliseconds);
}
