# mostly copied from Eden Li's mysql interface
# "Who is supposed to grok this mess?" --- phf

include $(GOROOT)/src/Make.$(GOARCH)

TARG=db/sqlite3
CGOFILES=low.go
GOFILES=core.go error.go util.go connection.go statement.go result.go classic.go set.go doc.go
CGO_LDFLAGS=-lsqlite3
CLEANFILES+=example test.db

include $(GOROOT)/src/Make.pkg

example: install test.db example.go
	$(GC) example.go
	$(LD) -o $@ example.$O

test.db:
	sqlite3 test.db <create_db.sql
