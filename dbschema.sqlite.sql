CREATE TABLE sessions (
	session_id INTEGER PRIMARY KEY NOT NULL,
	polar_id INTEGER NOT NULL,
	auth_token TEXT NOT NULL
);
CREATE TABLE measurements (
	session_id INTEGER NOT NULL REFERENCES sessions(session_id),
	timestamp INTEGER NOT NULL,
	heartbeat REAL NOT NULL,
	PRIMARY KEY(session_id, timestamp)
);
