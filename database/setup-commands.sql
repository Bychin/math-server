DROP TABLE IF EXISTS users;
CREATE TABLE users (
  login varchar(255) primary key,
  pass varchar(255) NOT NULL
);

INSERT INTO users (login, pass) VALUES
('pavel3', 'lol'),
('pavel4', 'lol'),
('pavel', 'lol'),
('test1', 'pass'),
('test2', 'pass'),
('kseniya', 'shks'),
('kseniya2', 'shks2'),
('Lusie', 'Alira'),
('Dmitry', 'Bogod'),
('kseniya1', 'shks1'),
('tiger', '123');
