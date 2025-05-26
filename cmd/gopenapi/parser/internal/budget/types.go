package budget

import "time"

// Amount is a type alias for monetary amounts
type Amount float64

// Currency represents a currency code
type Currency string

// Budget represents a budget
type Budget struct {
	ID          string     `json:"id"`
	Total       Amount     `json:"total"`
	Allocated   Amount     `json:"allocated"`
	Spent       Amount     `json:"spent"`
	Currency    Currency   `json:"currency"`
	Categories  []Category `json:"categories"`
	Period      Period     `json:"period"`
	LastUpdated time.Time  `json:"last_updated"`
}

// Category represents a budget category
type Category struct {
	Name          string        `json:"name"`
	Allocated     Amount        `json:"allocated"`
	Spent         Amount        `json:"spent"`
	Remaining     Amount        `json:"remaining"`
	Subcategories []Subcategory `json:"subcategories"`
}

// Subcategory represents a budget subcategory
type Subcategory struct {
	Name        string `json:"name"`
	Amount      Amount `json:"amount"`
	Description string `json:"description"`
}

// Period represents a budget period
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  string    `json:"type"` // monthly, quarterly, yearly
}

// Resources represents allocated resources
type Resources struct {
	Budget    Budget      `json:"budget"`
	Personnel int         `json:"personnel"`
	Equipment []Equipment `json:"equipment"`
}

// Equipment represents equipment allocation
type Equipment struct {
	Type        string `json:"type"`
	Count       int    `json:"count"`
	CostPerUnit Amount `json:"cost_per_unit"`
}

// BudgetHealth represents the health status of a budget
type BudgetHealth struct {
	Status          string      `json:"status"` // healthy, warning, critical
	UtilizationRate float64     `json:"utilization_rate"`
	Projections     Projections `json:"projections"`
}

// Projections contains budget projections
type Projections struct {
	EndOfPeriod   Amount    `json:"end_of_period"`
	Overrun       bool      `json:"overrun"`
	EstimatedDate time.Time `json:"estimated_date"`
}
