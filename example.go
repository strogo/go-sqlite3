package main

import "sqlite3"
import "fmt"

func main() {
	info := sqlite3.ConnectionInfo{"name": "test.db"};

	fmt.Printf("About to connect\n");
	c, e := sqlite3.Open(info);
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("connection: %s\n", c);

	fmt.Printf("About to create cursor\n");
	cc, e := c.Cursor();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("cursor: %s\n", cc);

	fmt.Printf("About to execute query\n");
	e = cc.Execute("SELECT * FROM users");
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("cursor: %s\n", cc);

	for {
		fmt.Printf("About to fetch one row\n");
		d, e := cc.FetchOne();
		if e != nil {
			fmt.Printf("error: %s\n", e.String());
			break;
		}
		fmt.Printf("cursor: %s\n", cc);
		fmt.Printf("data: %s\n", d);
	}


	fmt.Printf("About to close cursor\n");
	e = cc.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}

	fmt.Printf("About to close connection\n");
	e = c.Close();
	if e != nil {
		fmt.Printf("error: %s\n", e.String());
	}
}
