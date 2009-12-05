package main

import "db"
import "db/sqlite3"
import "fmt"

func main() {
	fmt.Printf("About to query version\n");
	version, e := sqlite3.Version();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	for k, v := range version {
		fmt.Printf("version[%s] == %s\n", k, v);
	}

	info := sqlite3.ConnectionInfo{"name": "test.db"};

	fmt.Printf("About to connect\n");
	c, e := sqlite3.Open(info);
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("connection: %s\n", c);

	fmt.Printf("About to prepare statement\n");
	s, e := c.Prepare("SELECT rowid, * FROM users WHERE password=?");
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("statement: %s\n", s);

	fmt.Printf("About to execute query\n");
	cc, e := c.Execute(s, "somepassword");
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);

	fmt.Printf("About to fetch all results\n");
	d, e := cc.FetchAll();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);
	fmt.Printf("data: %s\n", d);

	fmt.Printf("About to close cursor\n");
	e = cc.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("About to re-execute query\n");
	cc, e = c.Execute(s, "somepassword");
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);

	fmt.Printf("About to fetch one row\n");
	r, e := cc.FetchOne();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);
	fmt.Printf("data: %s\n", r);

	fmt.Printf("About to fetch another row\n");
	f, e := cc.FetchOne();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);
	fmt.Printf("data: %s\n", f);

	fmt.Printf("About to fetch the rest\n");
	g, e := cc.FetchMany(10);
	fmt.Printf("%s\n", g);
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("cursor: %s\n", cc);
	for _, y := range g {
		fmt.Printf("data: %s\n", y);
	}

	fmt.Printf("About to close cursor\n");
	e = cc.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("About to close statement\n");
	e = s.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("About to execute directly\n");
	d, e = db.ExecuteDirectly(c, "SELECT * FROM users");
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
	fmt.Printf("data: %s\n", d);

	fmt.Printf("About to close connection\n");
	e = c.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
}
