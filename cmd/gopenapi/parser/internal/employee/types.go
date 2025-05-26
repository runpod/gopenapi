package employee

import (
	"time"

	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/skills"
)

// EmployeeID is a type alias for employee identifiers
type EmployeeID string

// Employee represents an employee
type Employee struct {
	ID         EmployeeID     `json:"id"`
	Name       string         `json:"name"`
	Email      string         `json:"email"`
	Department string         `json:"department"`
	Position   Position       `json:"position"` // Local nested type
	Skills     []skills.Skill `json:"skills"`   // From skills package
	Contact    ContactInfo    `json:"contact"`  // Local nested type
	JoinedAt   time.Time      `json:"joined_at"`
	IsActive   bool           `json:"is_active"`
}

// Position represents an employee's position
type Position struct {
	Title      string            `json:"title"`
	Level      string            `json:"level"`
	Seniority  int               `json:"seniority"`
	Competency skills.Competency `json:"competency"` // From skills package
}

// ContactInfo contains contact information
type ContactInfo struct {
	Phone     string           `json:"phone"`
	Address   Address          `json:"address"`   // Deeply nested
	Emergency EmergencyContact `json:"emergency"` // Another level of nesting
}

// Address represents a physical address
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// EmergencyContact represents emergency contact info
type EmergencyContact struct {
	Name     string `json:"name"`
	Relation string `json:"relation"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

// EmployeeAnalytics contains analytics about employees
type EmployeeAnalytics struct {
	TotalEmployees  int                    `json:"total_employees"`
	ByDepartment    map[string]int         `json:"by_department"`
	ByLevel         map[string]int         `json:"by_level"`
	SkillsBreakdown skills.SkillsBreakdown `json:"skills_breakdown"` // From skills package
	Timestamp       time.Time              `json:"timestamp"`
}
