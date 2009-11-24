# mostly copied from Eden Li's mysql interface
# "Who is supposed to grok this mess?" --- phf

include $(GOROOT)/src/Make.$(GOARCH)

TARG=db/sqlite3
CGOFILES=core.go
GOFILES=error.go
CGO_LDFLAGS=-lsqlite3
CLEANFILES+=example test.db

include $(GOROOT)/src/Make.pkg

sqlite3_core.so: core.cgo4.o
	gcc $(_CGO_CFLAGS_$(GOARCH)) $(_CGO_LDFLAGS_$(GOOS)) -o $@ core.cgo4.o $(CGO_LDFLAGS)

example: install test.db example.go
	$(GC) example.go
	$(LD) -o $@ example.$O

test.db:
	sqlite3 test.db <create_db.sql
