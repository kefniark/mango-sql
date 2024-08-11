CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy');

CREATE TABLE person (
   name text,
   current_mood mood
);