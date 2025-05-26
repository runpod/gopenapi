package skills

// SkillLevel represents the proficiency level of a skill
type SkillLevel int

const (
	Beginner SkillLevel = iota + 1
	Intermediate
	Advanced
	Expert
)

// Skill represents a skill
type Skill struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Category   string     `json:"category"`
	Level      SkillLevel `json:"level"`
	Certified  bool       `json:"certified"`
	YearsOfExp int        `json:"years_of_exp"`
}

// Competency represents competency information
type Competency struct {
	Technical []Skill    `json:"technical"`
	Soft      []Skill    `json:"soft"`
	Languages []Language `json:"languages"`
	Overall   float64    `json:"overall"`
}

// Language represents a language skill
type Language struct {
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"` // native, fluent, conversational, basic
	Written     bool   `json:"written"`
	Spoken      bool   `json:"spoken"`
}

// SkillsBreakdown provides a breakdown of skills
type SkillsBreakdown struct {
	ByCategory map[string]int     `json:"by_category"`
	ByLevel    map[SkillLevel]int `json:"by_level"`
	TopSkills  []Skill            `json:"top_skills"`
	SkillGaps  []string           `json:"skill_gaps"`
}
