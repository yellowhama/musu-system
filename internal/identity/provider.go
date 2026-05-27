package identity

import (
	"fmt"
	"math/rand"
	"time"
)

type Profile struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type Provider interface {
	RequestEmail() (string, error)
	RequestPhone(service string) (string, string, error) // number, orderID, error
	CheckSMS(orderID string) (string, error)
}

type MockProvider struct{}

func (p *MockProvider) RequestEmail() (string, error) {
	names := []string{"hacker", "citizen", "digital", "nomad", "musu"}
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s%d@mockmail.com", names[rand.Intn(len(names))], rand.Intn(9999)), nil
}

func (p *MockProvider) RequestPhone(service string) (string, string, error) {
	return "+15550123456", "MOCK-ORDER-123", nil
}

func (p *MockProvider) CheckSMS(orderID string) (string, error) {
	return "123456", nil
}
