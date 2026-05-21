-- Local-dev schema bootstrap.
-- Runs automatically the first time the mysql container is created.

CREATE DATABASE IF NOT EXISTS kris;
USE kris;

-- Placeholder probe table; replace with your real schema.
CREATE TABLE IF NOT EXISTS probe (
  id  INT AUTO_INCREMENT PRIMARY KEY,
  msg VARCHAR(255),
  ts  DATETIME DEFAULT CURRENT_TIMESTAMP
);
