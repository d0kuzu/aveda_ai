package models

type Campuslogin struct {
	UserId    string `gorm:"primaryKey;column:user_id"`
	ContactID              int    `gorm:"column:contact_id"`
	ProgramID              int    `gorm:"column:program_id"`
	IsGrade11OrLower       bool   `gorm:"column:is_grade11_or_lower;default:false"`
	IsInternationalStudent bool   `gorm:"column:is_international_student;default:false"`
	FirstName              string `gorm:"column:first_name"`
	Email                  string `gorm:"column:email"`
}
