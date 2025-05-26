package company

import (
	"time"

	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/department"
	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/employee"
)

// CompanyID is a type alias for company identifiers
type CompanyID string

// Company represents a company with departments and employees
type Company struct {
	ID           CompanyID               `json:"id"`
	Name         string                  `json:"name"`
	Founded      time.Time               `json:"founded"`
	Departments  []department.Department `json:"departments"`  // References department package
	CEO          employee.Employee       `json:"ceo"`          // References employee package
	Employees    []employee.Employee     `json:"employees"`    // Slice of employees
	Headquarters Location                `json:"headquarters"` // Local type
	Subsidiaries []Company               `json:"subsidiaries"` // Recursive reference
	Metadata     map[string]interface{}  `json:"metadata"`
}

// Location represents a physical location
type Location struct {
	Address string     `json:"address"`
	City    string     `json:"city"`
	Country string     `json:"country"`
	Coords  Coordinate `json:"coords"`
}

// Coordinate represents GPS coordinates
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CompanyReport contains analytics about a company
type CompanyReport struct {
	Company     Company                    `json:"company"`
	Performance department.Performance     `json:"performance"` // From department package
	TopTeams    []department.Team          `json:"top_teams"`   // From department package
	Analytics   employee.EmployeeAnalytics `json:"analytics"`   // From employee package
	Generated   time.Time                  `json:"generated"`
}
