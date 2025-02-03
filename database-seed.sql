CREATE TABLE departments (
    department_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(100) NOT NULL
);
CREATE TABLE employees (
    employee_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    department_id INT REFERENCES departments(department_id),
    hire_date DATE NOT NULL,
    salary NUMERIC(10, 2) NOT NULL
);
CREATE TABLE sales (
    sale_id SERIAL PRIMARY KEY,
    employee_id INT REFERENCES employees(employee_id),
    sale_date DATE NOT NULL,
    amount NUMERIC(10, 2) NOT NULL
);

CREATE TABLE projects (
    project_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    budget NUMERIC(10, 2) NOT NULL
);
INSERT INTO departments (name, location) VALUES
('Sales', 'New York'),
('Engineering', 'San Francisco'),
('Marketing', 'Chicago'),
('HR', 'Los Angeles'),
('Finance', 'Boston');

INSERT INTO employees (name, department_id, hire_date, salary) VALUES
('John Doe', 1, '2020-01-15', 60000.00),
('Jane Smith', 2, '2019-05-20', 80000.00),
('Alice Johnson', 1, '2021-03-10', 55000.00),
('Bob Brown', 3, '2018-11-01', 70000.00),
('Charlie Davis', 2, '2022-07-05', 90000.00),
('Eva Green', 4, '2020-09-12', 65000.00),
('Frank White', 5, '2021-12-01', 75000.00),
('Grace Lee', 3, '2022-03-22', 72000.00),
('Henry Carter', 1, '2023-01-10', 58000.00),
('Ivy Adams', 2, '2023-02-15', 85000.00);

INSERT INTO sales (employee_id, sale_date, amount) VALUES
(1, '2023-01-10', 1500.00),
(1, '2023-02-15', 2000.00),
(2, '2023-01-20', 3000.00),
(3, '2023-03-05', 1200.00),
(4, '2023-02-01', 2500.00),
(5, '2023-03-10', 1800.00),
(6, '2023-04-01', 2200.00),
(7, '2023-05-15', 3000.00),
(8, '2023-06-20', 1500.00),
(9, '2023-07-05', 1700.00),
(10, '2023-08-10', 2100.00),
(1, '2023-09-15', 1900.00),
(2, '2023-10-20', 2300.00),
(3, '2023-11-05', 1400.00),
(4, '2023-12-01', 2600.00);

INSERT INTO projects (name, start_date, end_date, budget) VALUES
('Project Alpha', '2023-01-01', '2023-06-30', 100000.00),
('Project Beta', '2023-02-01', '2023-08-31', 150000.00),
('Project Gamma', '2023-03-01', '2023-09-30', 200000.00),
('Project Delta', '2023-04-01', '2023-10-31', 120000.00),
('Project Epsilon', '2023-05-01', '2023-11-30', 180000.00);

