CREATE TABLE user (
    id INTEGER UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    hash CHAR(60) NOT NULL,
    name VARCHAR(255) NOT NULL
);

CREATE INDEX user_email ON user(email);

CREATE TABLE model (
    id INTEGER UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    manufacturer VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL,
    UNIQUE (manufacturer, model)
);
CREATE INDEX model_manufacturer ON model(manufacturer);
CREATE INDEX model_model ON model(model);

CREATE TABLE status (
    status VARCHAR(50) PRIMARY KEY
);

CREATE TABLE device (
    id INTEGER UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    serial_number VARCHAR(255) UNIQUE NOT NULL,
    model_id INTEGER UNSIGNED NOT NULL,
    status VARCHAR(50) NOT NULL,
    location VARCHAR(255) NOT NULL,
    FOREIGN KEY(model_id) REFERENCES model(id) ON DELETE CASCADE,
    FOREIGN KEY(status) REFERENCES status(status) ON DELETE CASCADE,
);

CREATE INDEX device_serial_number ON device(serial_number);
CREATE INDEX device_model_id ON device(model_id);
CREATE INDEX device_status ON device(status);
CREATE INDEX device_location ON device(location);

CREATE TABLE device_log (
    id INTEGER UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    device_id INTEGER UNSIGNED NOT NULL,
    user_id INTEGER UNSIGNED NOT NULL,
    date DATETIME NOT NULL,
    type ENUM ('created', 'modified', 'note') NOT NULL,
    content TEXT,
    FOREIGN KEY(device_id) REFERENCES device(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX device_log_device_id ON device_log(device_id);
CREATE INDEX device_log_user_id ON device_log(user_id);
CREATE INDEX device_log_date ON device_log(date);
CREATE INDEX device_log_type ON device_log(type);

CREATE TABLE model_log (
    id INTEGER UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    model_id INTEGER UNSIGNED NOT NULL,
    user_id INTEGER UNSIGNED NOT NULL,
    date DATETIME NOT NULL,
    type ENUM ('created', 'modified', 'note') NOT NULL,
    content TEXT,
    FOREIGN KEY(model_id) REFERENCES model(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX model_log_device_id ON model_log(model_id);
CREATE INDEX model_log_user_id ON model_log(user_id);
CREATE INDEX model_log_date ON model_log(date);
CREATE INDEX model_log_type ON model_log(type);
