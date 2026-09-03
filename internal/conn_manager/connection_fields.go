package conn_manager

import (
	"github.com/charmbracelet/bubbles/textinput"
)

func createTextInput(value string) textinput.Model {
	ti := textinput.New()
	ti.CharLimit = 256
	ti.SetValue(value)
	return ti
}

func createNameInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Name"
	return ti
}

func createDriverInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Driver"
	return ti
}

// driverOption pairs the internal driver value (what adapters.database.go
// actually accepts) with a friendly display label for the connection form,
// plus that driver's conventional default port - prefilled so you don't
// have to go look up "what port does postgres use again" every time you
// add a new connection.
type driverOption struct {
	Value       string
	Label       string
	DefaultPort string
}

// DriverOptions is the closed set of drivers the app actually supports
// (see internal/adapters/database.go InitConnection) - the driver field on
// the connection form cycles through exactly these instead of accepting
// arbitrary free text, since typing anything else (e.g. "PostgreSQL" instead
// of "pgx") used to silently save a connection that could never connect
// ("unsupported driver: PostgreSQL").
var DriverOptions = []driverOption{
	{Value: "pgx", Label: "PostgreSQL", DefaultPort: "5432"},
	{Value: "mysql", Label: "MySQL", DefaultPort: "3306"},
}

// isDefaultPort reports whether port matches any driver's own default -
// used to decide whether switching drivers is safe to auto-update the Port
// field (it was never manually customized to something else) or should
// leave a real custom port alone.
func isDefaultPort(port string) bool {
	if port == "" {
		return true
	}
	for _, opt := range DriverOptions {
		if opt.DefaultPort == port {
			return true
		}
	}
	return false
}

// driverIndexForValue returns the DriverOptions index matching the given
// internal driver value, defaulting to 0 if not found (e.g. empty/new form).
func driverIndexForValue(value string) int {
	for i, opt := range DriverOptions {
		if opt.Value == value {
			return i
		}
	}
	return 0
}

func createHostInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Host"
	return ti
}

func createPortInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Port"
	return ti
}

func createUserInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "User"
	return ti
}

func createUrlInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Connection URL"
	return ti
}

func createCommandInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Custom Command"
	return ti
}

func createPasswordInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	return ti
}

func createDatabaseInput(value string) textinput.Model {
	ti := createTextInput(value)
	ti.Placeholder = "Database (optional, e.g. pmo_db)"
	return ti
}
