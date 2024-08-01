CREATE TABLE tags (
    question_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    name VARCHAR(20) NOT NULL,
    PRIMARY KEY(question_id, tag_id)
);