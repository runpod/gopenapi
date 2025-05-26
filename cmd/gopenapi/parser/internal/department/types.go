package department

import (
	"time"

	"github.com/runpod/gopenapi/cmd/gopenapi/parser/internal/budget"
)

// DepartmentID is a type alias for department identifiers
type DepartmentID string

// Department represents a department within a company
type Department struct {
	ID        DepartmentID  `json:"id"`
	Name      string        `json:"name"`
	Manager   ManagerInfo   `json:"manager"` // Local nested type
	Teams     []Team        `json:"teams"`   // Slice of local type
	Budget    budget.Budget `json:"budget"`  // From budget package
	CreatedAt time.Time     `json:"created_at"`
}

// ManagerInfo contains information about a department manager
type ManagerInfo struct {
	EmployeeID string    `json:"employee_id"`
	Name       string    `json:"name"`
	Since      time.Time `json:"since"`
	Level      int       `json:"level"`
}

// Team represents a team within a department
type Team struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Size        int              `json:"size"`
	Projects    []Project        `json:"projects"`    // Nested array
	Performance TeamPerformance  `json:"performance"` // Nested struct
	Resources   budget.Resources `json:"resources"`   // From budget package
}

// Project represents a project
type Project struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Deadline   time.Time   `json:"deadline"`
	Milestones []Milestone `json:"milestones"`
}

// Milestone represents a project milestone
type Milestone struct {
	Name        string    `json:"name"`
	Completed   bool      `json:"completed"`
	CompletedAt time.Time `json:"completed_at"`
}

// TeamPerformance contains performance metrics
type TeamPerformance struct {
	Score       float64            `json:"score"`
	Metrics     map[string]float64 `json:"metrics"`
	LastUpdated time.Time          `json:"last_updated"`
}

// Performance represents overall department performance
type Performance struct {
	DepartmentID DepartmentID        `json:"department_id"`
	Quarter      string              `json:"quarter"`
	Metrics      map[string]float64  `json:"metrics"`
	TeamScores   map[string]float64  `json:"team_scores"`
	BudgetHealth budget.BudgetHealth `json:"budget_health"` // From budget package
}
