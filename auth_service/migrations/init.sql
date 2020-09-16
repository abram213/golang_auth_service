CREATE TABLE IF NOT EXISTS users  (
    id          serial PRIMARY KEY,
    login       varchar(255) UNIQUE,
    password    varchar(255),
    admin       boolean DEFAULT '0',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);