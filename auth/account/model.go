package account

import (
	"time"
)

const tableName = "account"

// Account implements auth.Account
type Account struct {
	ID           int64                `gorm:"primaryKey;column:id"`
	DeviceID     string               `gorm:"uniqueIndex;column:device_id;not null"`
	Banned       bool                 `gorm:"column:banned"`
	BannedReason string               `gorm:"column:banned_reason"`
	RegisterAt   time.Time            `gorm:"column:register_at"`
	RegisterIp   string               `gorm:"column:register_ip"`
	LastLoginAt  time.Time            `gorm:"column:last_login_at"`
	LastLoginIp  string               `gorm:"column:last_login_ip"`
	Name         string               `gorm:"column:name"`
	Avatar       string               `gorm:"column:avatar"`
	Gender       int                  `gorm:"column:gender"`
	Location     string               `gorm:"location"`
	Providers    map[string]*provider `gorm:"-"`
}

func newAccount() *Account {
	return &Account{
		Providers: make(map[string]*provider),
	}
}

func (*Account) TableName() string { return tableName }

func (a *Account) GetID() int64                       { return a.ID }
func (a *Account) SetID(x int64)                      { a.ID = x }
func (a *Account) GetDeviceID() string                { return a.DeviceID }
func (a *Account) SetDeviceID(x string)               { a.DeviceID = x }
func (a *Account) GetBanned() (bool, string)          { return a.Banned, a.BannedReason }
func (a *Account) SetBanned(x bool, y string)         { a.Banned, a.BannedReason = x, y }
func (a *Account) GetRegister() (time.Time, string)   { return a.RegisterAt, a.RegisterIp }
func (a *Account) SetRegister(x time.Time, y string)  { a.RegisterAt, a.RegisterIp = x, y }
func (a *Account) GetLastLogin() (time.Time, string)  { return a.LastLoginAt, a.LastLoginIp }
func (a *Account) SetLastLogin(x time.Time, y string) { a.LastLoginAt, a.LastLoginIp = x, y }
func (a *Account) GetName() string                    { return a.Name }
func (a *Account) SetName(x string)                   { a.Name = x }
func (a *Account) GetAvatar() string                  { return a.Avatar }
func (a *Account) SetAvatar(x string)                 { a.Avatar = x }
func (a *Account) GetGender() int                     { return a.Gender }
func (a *Account) SetGender(x int)                    { a.Gender = x }
func (a *Account) GetLocation() string                { return a.Location }
func (a *Account) SetLocation(x string)               { a.Location = x }
func (a *Account) GetProvider(x string) string        { return lookupProvider(a.Providers, x) }
func (a *Account) SetProvider(x, y string)            { setProvider(a.Providers, x, y) }
func (a *Account) GetProviders() map[string]string {
	var m = make(map[string]string)
	for k, p := range a.Providers {
		m[k] = p.Token
	}
	return m
}

type provider struct {
	ID       int64  `gorm:"primaryKey;column:id"`
	Uid      int64  `gorm:"column:uid;not null"`
	Provider string `gorm:"uniqueIndex:provider_token;column:provider;not null"`
	Token    string `gorm:"uniqueIndex:provider_token;column:token;not null"`
	OpenId   string `gorm:"column:openid"`
}

func (p *provider) TableName() string {
	return "provider"
}

func lookupProvider(providers map[string]*provider, typ string) string {
	if p, ok := providers[typ]; ok {
		return p.Token
	}
	return ""
}

func setProvider(providers map[string]*provider, typ, token string) {
	if p, ok := providers[typ]; ok {
		p.Token = token
	}
}
