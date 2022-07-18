package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// PasswordCanBeModifiedBy checks whether "other" can change the password of "user".
func (user User) PasswordCanBeModifiedBy(other User) bool {
	if other.Username == "determined" {
		return false
	}

	if other.Admin {
		return true
	}

	if other.ID == user.ID {
		return true
	}

	return false
}

// CanCreateUser checks whether the calling user
// has the authority to create other users.
func (user User) CanCreateUser() bool {
	return user.Admin
}

// AdminCanBeModifiedBy checks whether "other" can enable or disable the admin status of "user".
func (user User) AdminCanBeModifiedBy(other User) bool {
	return other.Admin
}

// ActiveCanBeModifiedBy checks whether "other" can enable or disable the active status of "user".
func (user User) ActiveCanBeModifiedBy(other User) bool {
	return other.Admin
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

func (user *User) Proto() *userv1.User {
	return &userv1.User{
		Id:         int32(user.ID),
		Username:   user.Username,
		Admin:      user.Admin,
		Active:     user.Active,
		ModifiedAt: timestamppb.New(user.ModifiedAt),
	}
}

type Users []User

func (users Users) Proto() []*userv1.User {
	out := make([]*userv1.User, len(users))
	for i, u := range users {
		out[i] = u.Proto()
	}
	return out
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
