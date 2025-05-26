package mock

import "time"

// Basic type aliases
type UserID string
type ProductID int64
type Price float64
type IsActive bool

// Slice aliases
type Tags []string
type Scores []float64
type UserIDs []UserID

// Struct with various field types
type User struct {
	ID        UserID    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	IsActive  IsActive  `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	Tags      Tags      `json:"tags"`
}

// Product struct with different types
type Product struct {
	ID          ProductID     `json:"id"`
	Name        string        `json:"name"`
	Price       Price         `json:"price"`
	InStock     bool          `json:"in_stock"`
	LastUpdated time.Time     `json:"last_updated"`
	Duration    time.Duration `json:"duration"`
	Scores      Scores        `json:"scores"`
}

// Nested struct with aliases
type Order struct {
	ID       UserID    `json:"id"`
	UserID   UserID    `json:"user_id"`
	Products []Product `json:"products"`
	Total    Price     `json:"total"`
	Status   Status    `json:"status"`
}

// String-based enum alias
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
)

// Complex nested structure
type Analytics struct {
	UserMetrics    UserMetrics    `json:"user_metrics"`
	ProductMetrics ProductMetrics `json:"product_metrics"`
	TimeRange      TimeRange      `json:"time_range"`
}

type UserMetrics struct {
	TotalUsers  int64     `json:"total_users"`
	ActiveUsers int64     `json:"active_users"`
	NewUsers    int64     `json:"new_users"`
	UserIDs     UserIDs   `json:"user_ids"`
	LastUpdated time.Time `json:"last_updated"`
}

type ProductMetrics struct {
	TotalProducts int64  `json:"total_products"`
	AveragePrice  Price  `json:"average_price"`
	TopScores     Scores `json:"top_scores"`
}

type TimeRange struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

// Pointer types
type OptionalUser struct {
	User     *User     `json:"user,omitempty"`
	UserID   *UserID   `json:"user_id,omitempty"`
	IsActive *IsActive `json:"is_active,omitempty"`
}

// Map types with aliases
type UserMap map[UserID]User
type PriceMap map[ProductID]Price

// Interface for testing
type Identifiable interface {
	GetID() UserID
}

// Struct implementing interface
type IdentifiableUser struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
}

func (u IdentifiableUser) GetID() UserID {
	return u.ID
}
