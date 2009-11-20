CREATE TABLE Users (
  login VARCHAR NOT NULL UNIQUE,
  password VARCHAR NOT NULL,
  name VARCHAR NOT NULL UNIQUE,
  email VARCHAR NOT NULL UNIQUE,
  active BOOLEAN NOT NULL DEFAULT 0,
  last TIMESTAMP,
  PRIMARY KEY (login)
);

INSERT INTO Users (login, password, name, email)
  VALUES ("phf", "somepassword", "Peter Froehlich", "phf@phf.phf");

INSERT INTO Users (login, password, name, email)
  VALUES ("adt", "somepassword", "Abstract Data Type", "adt@adt.adt");
