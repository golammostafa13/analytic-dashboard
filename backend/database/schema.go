// Define table schema structures
package database

type TableSchema struct {
	Name        string         `json:"name"`
	Columns     []ColumnSchema `json:"columns"`
	Description string         `json:"description"`
}

type ColumnSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable"`
	Description string `json:"description"`
}

var DatabaseSchema = []TableSchema{
	{
		Name:        "departments",
		Description: "Stores department information",
		Columns: []ColumnSchema{
			{Name: "department_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Department name"},
			{Name: "location", Type: "varchar(100)", Nullable: false, Description: "Location of the department"},
		},
	},
	{
		Name:        "employees",
		Description: "Stores employee information",
		Columns: []ColumnSchema{
			{Name: "employee_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Employee name"},
			{Name: "department_id", Type: "int", Nullable: false, Description: "Reference to departments table"},
			{Name: "hire_date", Type: "date", Nullable: false, Description: "Hire date of the employee"},
			{Name: "salary", Type: "numeric(10,2)", Nullable: false, Description: "Salary of the employee"},
		},
	},
	{
		Name:        "sales",
		Description: "Stores sales records",
		Columns: []ColumnSchema{
			{Name: "sale_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "employee_id", Type: "int", Nullable: false, Description: "Reference to employees table"},
			{Name: "sale_date", Type: "date", Nullable: false, Description: "Date of the sale"},
			{Name: "amount", Type: "numeric(10,2)", Nullable: false, Description: "Amount of the sale"},
		},
	},
	{
		Name:        "projects",
		Description: "Stores project information",
		Columns: []ColumnSchema{
			{Name: "project_id", Type: "serial", Nullable: false, Description: "Primary key"},
			{Name: "name", Type: "varchar(100)", Nullable: false, Description: "Project name"},
			{Name: "start_date", Type: "date", Nullable: false, Description: "Start date of the project"},
			{Name: "end_date", Type: "date", Nullable: false, Description: "End date of the project"},
			{Name: "budget", Type: "numeric(10,2)", Nullable: false, Description: "Budget for the project"},
		},
	},
}
