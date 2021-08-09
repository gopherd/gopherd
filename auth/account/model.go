package account

import (
	"time"
)

const tableName = "account"

// account implements auth.Account
type account struct {
	ID           int64
	DeviceID     string
	Banned       bool
	BannedReason string
	RegisterAt   time.Time
	RegisterIp   string
	LastLoginAt  time.Time
	LastLoginIp  string
	Name         string
	Avatar       string
	Gender       int
	Location     string
	Providers    map[string]string
}

func newAccount() *account {
	return &account{
		Providers: make(map[string]string),
	}
}

func (a *account) GetID() int64                       { return a.ID }
func (a *account) SetID(x int64)                      { a.ID = x }
func (a *account) GetDeviceID() string                { return a.DeviceID }
func (a *account) SetDeviceID(x string)               { a.DeviceID = x }
func (a *account) GetBanned() (bool, string)          { return a.Banned, a.BannedReason }
func (a *account) SetBanned(x bool, y string)         { a.Banned, a.BannedReason = x, y }
func (a *account) GetRegister() (time.Time, string)   { return a.RegisterAt, a.RegisterIp }
func (a *account) SetRegister(x time.Time, y string)  { a.RegisterAt, a.RegisterIp = x, y }
func (a *account) GetLastLogin() (time.Time, string)  { return a.LastLoginAt, a.LastLoginIp }
func (a *account) SetLastLogin(x time.Time, y string) { a.LastLoginAt, a.LastLoginIp = x, y }
func (a *account) GetName() string                    { return a.Name }
func (a *account) SetName(x string)                   { a.Name = x }
func (a *account) GetAvatar() string                  { return a.Avatar }
func (a *account) SetAvatar(x string)                 { a.Avatar = x }
func (a *account) GetGender() int                     { return a.Gender }
func (a *account) SetGender(x int)                    { a.Gender = x }
func (a *account) GetLocation() string                { return a.Location }
func (a *account) SetLocation(x string)               { a.Location = x }
func (a *account) GetProvider(x string) string        { return a.Providers[x] }
func (a *account) SetProvider(x, y string)            { a.Providers[x] = y }

func (a *account) GetProviders() []string {
	var providers = make([]string, 0, len(a.Providers))
	for k := range a.Providers {
		providers = append(providers, k)
	}
	return providers
}
