CREATE TABLE file (
    screenid varchar(36) NOT NULL,
    lineid varchar(36) NOT NULL,
    filename varchar(200) NOT NULL,
    filetype varchar(20) NOT NULL,
    diskfilename varchar(250) NOT NULL,
    contents blob NOT NULL,
    PRIMARY KEY (screenid, lineid, filename)
);