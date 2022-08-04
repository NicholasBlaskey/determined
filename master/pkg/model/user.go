package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/proto/pkg/userv1"
)

// BCryptCost is a stopgap until we implement sane master-configuration.
const BCryptCost = 15

// UserID is the type for user IDs.
type UserID int

// SessionID is the type for user session IDs.
type SessionID int

// User corresponds to a row in the "users" DB table.
type User struct {
	ID           UserID      `db:"id" json:"id"`
	Username     string      `db:"username" json:"username"`
	PasswordHash null.String `db:"password_hash" json:"-"`
	DisplayName  null.String `db:"display_name" json:"display_name"`
	Admin        bool        `db:"admin" json:"admin"`
	Active       bool        `db:"active" json:"active"`
	ModifiedAt   time.Time   `db:"modified_at" json:"modified_at"`
}

// UserSession corresponds to a row in the "user_sessions" DB table.
type UserSession struct {
	ID     SessionID `db:"id" json:"id"`
	UserID UserID    `db:"user_id" json:"user_id"`
	Expiry time.Time `db:"expiry" json:"expiry"`
}

// A FullUser is a User joined with any other user relations.
type FullUser struct {
	ID          UserID      `db:"id" json:"id"`
	DisplayName null.String `db:"display_name" json:"display_name"`
	Username    string      `db:"username" json:"username"`
	Admin       bool        `db:"admin" json:"admin"`
	Active      bool        `db:"active" json:"active"`
	ModifiedAt  time.Time   `db:"modified_at" json:"modified_at"`

	AgentUID   null.Int    `db:"agent_uid" json:"agent_uid"`
	AgentGID   null.Int    `db:"agent_gid" json:"agent_gid"`
	AgentUser  null.String `db:"agent_user" json:"agent_user"`
	AgentGroup null.String `db:"agent_group" json:"agent_group"`
}

// ToUser converts a FullUser model to just a User model.
func (u FullUser) ToUser() User {
	return User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: null.String{},
		DisplayName:  u.DisplayName,
		Admin:        u.Admin,
		Active:       u.Active,
		ModifiedAt:   u.ModifiedAt,
	}
}

// UserFromProto converts a proto user to a model user.
func UserFromProto(u *userv1.User) User {
	return User{
		ID:          UserID(u.Id),
		Username:    u.Username,
		DisplayName: null.StringFrom(u.DisplayName),
		Admin:       u.Admin,
		Active:      u.Active,
		ModifiedAt:  u.ModifiedAt.AsTime(),
	}
}

// ValidatePassword checks that the supplied password is correct.
func (user User) ValidatePassword(password string) bool {
	// If an empty password was posted, we need to check that the
	// user is a password-less user.
	if password == "" {
		return !user.PasswordHash.Valid
	}

	// If the model's password is empty, then
	// supplied password must be incorrect
	if !user.PasswordHash.Valid {
		return false
	}
	err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash.ValueOrZero()),
		[]byte(password))

	return err == nil
}

// UpdatePasswordHash updates the model's password hash employing necessary cryptographic
// techniques.
func (user *User) UpdatePasswordHash(password string) error {
	if password == "" {
		user.PasswordHash = null.NewString("", false)
	} else {
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(password),
			BCryptCost,
		)
		if err != nil {
			return err
		}
		user.PasswordHash = null.StringFrom(string(passwordHash))
	}
	return nil
}

// ExternalSessions provides an integration point for an external service to issue JWTs to control
// access to the cluster.
type ExternalSessions struct {
	LoginURI  string `json:"login_uri"`
	LogoutURI string `json:"logout_uri"`
	JwtKey    string `json:"jwt_key"`
}

// UserWebSetting is a record of user web setting.
type UserWebSetting struct {
	UserID      UserID
	Key         string
	Value       string
	StoragePath string
}
