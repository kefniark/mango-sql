CREATE TABLE numeric (
    ID  SERIAL PRIMARY KEY,
    smallserial smallserial NOT NULL,
    serial serial NOT NULL,
    bigserial bigserial NOT NULL,
    integer integer NOT NULL,
    smallint smallint NOT NULL,
    bigint bigint NOT NULL,
    numeric numeric(4) NOT NULL,
    float float(4) NOT NULL
);

CREATE TABLE numeric_nullable (
    ID  SERIAL PRIMARY KEY,
    smallserial smallserial,
    serial serial,
    bigserial bigserial,
    integer integer,
    smallint smallint,
    bigint bigint,
    numeric numeric(4),
    float float(4)
);

CREATE TABLE text (
    ID  SERIAL PRIMARY KEY,
    char1 CHARACTER NOT NULL,
    char2 CHAR(10) NOT NULL,
    varchar1 VARCHAR(10) NOT NULL,
    varchar2 CHARACTER VARYING(10) NOT NULL,
    text TEXT NOT NULL,
    text2 BPCHAR NOT NULL
);

CREATE TABLE text_nullable (
    ID  SERIAL PRIMARY KEY,
    char1 CHARACTER,
    char2 CHAR(10),
    varchar1 VARCHAR(10),
    varchar2 CHARACTER VARYING(10),
    text TEXT,
    text2 BPCHAR
);

CREATE TABLE list (
    ID  SERIAL PRIMARY KEY,
    integer1 INTEGER[] NOT NULL,
    integer2 INTEGER[],
    smallint1 SMALLINT[] NOT NULL,
    smallint2 SMALLINT[],
    bigint1 BIGINT[] NOT NULL,
    bigint2 BIGINT[],
    text1 TEXT[] NOT NULL,
    text2 TEXT[]
);

CREATE TABLE json (
    ID  SERIAL PRIMARY KEY,
    json1 JSON NOT NULL,
    json2 JSON,
    jsonb1 JSONB NOT NULL,
    jsonb2 JSONB
);

CREATE TABLE date (
    ID  SERIAL PRIMARY KEY,
    date1 DATE NOT NULL,
    date2 DATE,
    time1 TIME NOT NULL,
    time2 TIME,
    timestamp1 TIMESTAMP NOT NULL,
    timestamp2 TIMESTAMP,
    timestamptz1 TIMESTAMPTZ NOT NULL,
    timestamptz2 TIMESTAMPTZ
);