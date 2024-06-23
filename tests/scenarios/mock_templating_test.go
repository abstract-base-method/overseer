package scenarios

import (
	"text/template"

	"github.com/stretchr/testify/mock"
)

type MockTemplatingClient struct {
	mock.Mock
}

func (m *MockTemplatingClient) InternalLoreTemplate() *template.Template {
	args := m.Called()
	return args.Get(0).(*template.Template)
}

func (m *MockTemplatingClient) PublicLoreTemplate() *template.Template {
	args := m.Called()
	return args.Get(0).(*template.Template)
}

func (m *MockTemplatingClient) CoordinateLoreTemplate() *template.Template {
	args := m.Called()
	return args.Get(0).(*template.Template)
}
