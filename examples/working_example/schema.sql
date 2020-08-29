use test;

create table Companies (
	id INTEGER NOT NULL AUTO_INCREMENT,
	name VARCHAR(20), 
	country VARCHAR(3),
	PRIMARY KEY (ID)
);
insert into Companies (name, country) values ('Company1', 'USA');
insert into Companies (name, country) values ('Company1', 'SWE');
insert into Companies (name, country) values ('Company1', 'IRL');
insert into Companies (name, country) values ('Company2', 'IRL');


create table Requests (
	id INTEGER NOT NULL AUTO_INCREMENT,
	response_time INTEGER NOT NULL,
	PRIMARY KEY (ID)
);
insert into Requests (response_time) values (23);
insert into Requests (response_time) values (3);
insert into Requests (response_time) values (7);
insert into Requests (response_time) values (45);
