package model

import (
	"strconv"
	"time"
)

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code"`
}

// ContactInfo struct represents the user's contact information
type ContactInfo struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	LinkedIn string `json:"linkedin"`
	GitHub   string `json:"github"`
	Twitter  string `json:"twitter"`
}

// PaymentInfo struct represents the user's payment details
type PaymentInfo struct {
	CardNumber     string  `json:"card_number"`
	CardType       string  `json:"card_type"`
	ExpiryDate     string  `json:"expiry_date"`
	CVV            string  `json:"cvv"`
	BillingAddress Address `json:"billing_address"`
}

// UserProfile struct contains additional profile-related fields
type UserProfile struct {
	Bio         string    `json:"bio"`
	ProfilePic  string    `json:"profile_pic"`
	Interests   []string  `json:"interests"`
	MemberSince time.Time `json:"member_since"`
}

// User struct with many fields and nested structs
type User struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Age         int               `json:"age"`
	Gender      string            `json:"gender"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	LastLogin   time.Time         `json:"last_login"`
	Address     Address           `json:"address"`
	Contact     ContactInfo       `json:"contact_info"`
	Profile     UserProfile       `json:"profile"`
	Payments    []PaymentInfo     `json:"payments"`
	Preferences map[string]string `json:"preferences"`
}

func MockUser(id int) *User {
	return &User{
		ID:        id,
		Name:      "JohnDoe-" + strconv.FormatInt(int64(id), 10),
		Age:       30,
		Gender:    "Male",
		IsActive:  true,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
		Address: Address{
			Street:  "123 Main St",
			City:    "New York",
			State:   "NY",
			Country: "USA",
			ZipCode: "10001",
		},
		Contact: ContactInfo{
			Email:    "john.doe@example.com",
			Phone:    "+1234567890",
			LinkedIn: "linkedin.com/in/johndoe",
			GitHub:   "github.com/johndoe",
			Twitter:  "@johndoe",
		},
		Profile: UserProfile{
			Bio:         "Software engineer with 10 years of experience.",
			ProfilePic:  "https://example.com/johndoe.jpg",
			Interests:   []string{"coding", "traveling", "reading"},
			MemberSince: time.Now().AddDate(-5, 0, 0), // 5 years ago
		},
		Payments: []PaymentInfo{
			{
				CardNumber: "4111111111111111",
				CardType:   "Visa",
				ExpiryDate: "12/25",
				CVV:        "123",
				BillingAddress: Address{
					Street:  "123 Main St",
					City:    "New York",
					State:   "NY",
					Country: "USA",
					ZipCode: "10001",
				},
			},
		},
		Preferences: map[string]string{
			"theme":         "dark",
			"notifications": "enabled",
			"language":      "en",
		},
	}
}
