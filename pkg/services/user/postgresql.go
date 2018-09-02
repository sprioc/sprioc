package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/fatih/structs"
	"github.com/fokal/fokal-core/pkg/log"
	"github.com/fokal/fokal-core/pkg/request"
	"github.com/fokal/fokal-core/pkg/services/image"
	"github.com/fokal/fokal-core/pkg/services/permission"
	"github.com/jmoiron/sqlx"
)

type UserStore struct {
	db          *sqlx.DB
	permissions permission.Service
	images      image.Service
}

func New(db *sqlx.DB, permissions permission.Service, images image.Service) *UserStore {
	return &UserStore{
		db:          db,
		permissions: permissions,
		images:      images,
	}
}

func (store *UserStore) CreateUser(ctx context.Context, user *User) error {
	var userID uint64
	tx, err := store.db.Beginx()
	if err != nil {
		log.WithContext(ctx).Println(err)
		return err
	}

	exists, err := store.ExistsByEmail(ctx, user.Email)
	if err != nil {
		return errors.New("unable to reach user service")
	}

	if exists {
		return fmt.Errorf("user with email already exists")
	}

	exists, err = store.ExistsByUsername(ctx, user.Username)
	if err != nil {
		return errors.New("unable to reach user service")
	}

	if exists {
		return fmt.Errorf("user with username already exists")
	}

	rows, err := tx.QueryContext(ctx, `
	INSERT INTO content.users(username, email, name)
	VALUES($1, $2, $3) RETURNING id;`,
		user.Username, user.Email, user.Name)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&userID)
		if err != nil {
			log.WithContext(ctx).Error(err)
			return err
		}
	}

	err = store.permissions.AddScope(ctx, tx, userID, userID, permission.UserClass, permission.CanEdit)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	err = store.permissions.Public(ctx, tx, userID, permission.UserClass)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	err = store.permissions.AddScope(ctx, tx, userID, userID, permission.UserClass, permission.CanDelete)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.WithContext(ctx).Error(err)
		if err := tx.Rollback(); err != nil {
			log.WithContext(ctx).Error(err)
			return err
		}
		return err
	}
	return nil
}

func (store UserStore) UserByID(ctx context.Context, id uint64) (*User, error) {
	user := new(User)
	err := store.db.GetContext(ctx, user, "SELECT * FROM content.users WHERE id = $1", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return user, nil
}

func (store UserStore) UserByUsername(ctx context.Context, username string) (*User, error) {
	user := new(User)
	err := store.db.GetContext(ctx, user, "SELECT * FROM content.users WHERE username = $1", username)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return user, nil
}
func (store UserStore) UserByEmail(ctx context.Context, email string) (*User, error) {
	user := new(User)
	err := store.db.GetContext(ctx, user, "SELECT * FROM content.users WHERE email = $1", email)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return user, nil
}

func (store UserStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := store.db.GetContext(ctx, &exists, "SELECT count(1) FROM content.users WHERE email = $1", email)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return false, err
	}
	return exists, nil
}

func (store UserStore) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := store.db.GetContext(ctx, &exists, "SELECT count(1) FROM content.users WHERE username = $1", username)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return false, err
	}
	return exists, nil
}

func (store UserStore) Users(ctx context.Context, limit int) (*[]User, error) {
	var users *[]User

	err := store.db.SelectContext(ctx, users, "SELECT * FROM content.users ORDER BY last_modified LIMIT $1", limit)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return users, nil
}

func (store UserStore) Admins(ctx context.Context) (*[]User, error) {
	var users *[]User

	err := store.db.SelectContext(ctx, users, "SELECT * FROM content.users WHERE admin = true ORDER BY last_modified")
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return users, nil
}

func (store UserStore) IsAdmin(ctx context.Context, id uint64) (bool, error) {
	var exists bool
	err := store.db.GetContext(ctx, &exists, "SELECT count(1) FROM content.users WHERE id = $1 AND admin = true", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return false, err
	}
	return exists, nil
}

func (store UserStore) Featured(ctx context.Context) (*[]User, error) {
	var users *[]User

	err := store.db.SelectContext(ctx, users, "SELECT * FROM content.users WHERE featured = true ORDER BY last_modified")
	if err != nil {
		log.WithContext(ctx).Error(err)
		return nil, err
	}
	return users, nil
}

func (store UserStore) Feature(ctx context.Context, id uint64) error {
	log.WithContext(ctx).WithField("user-id", id).Debug("featuring user")
	_, err := store.db.ExecContext(ctx, "UPDATE content.users SET featured = TRUE WHERE id = $1", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	return nil
}

func (store UserStore) UnFeature(ctx context.Context, id uint64) error {
	log.WithContext(ctx).WithField("user-id", id).Debug("unfeaturing user")

	_, err := store.db.ExecContext(ctx, "UPDATE content.users SET featured = FALSE WHERE id = $1", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	return nil
}

func (store UserStore) IsFeatured(ctx context.Context, id uint64) (bool, error) {
	var exists bool
	err := store.db.GetContext(ctx, &exists, "SELECT count(1) FROM content.users WHERE id = $1 AND featured = true", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return false, err
	}
	return exists, nil
}

func (store UserStore) SetAvatarID(ctx context.Context, id uint64, avatarID string) error {
	_, err := store.db.ExecContext(ctx, "UPDATE content.users SET avatar_id = $1 WHERE id = $2", avatarID, id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	return nil
}

func (store UserStore) DeleteUser(ctx context.Context, id uint64) error {
	log.WithContext(ctx).WithField("user-id", id).Warn("performing distructive action: deleting user")
	_, err := store.db.ExecContext(ctx, "DELETE FROM content.users WHERE id = $1", id)
	if err != nil {
		log.WithContext(ctx).Error(err)
		return err
	}
	return nil
}

func (store UserStore) PatchUser(ctx context.Context, id uint64, changes request.PatchUser) error {
	tx, err := store.db.Beginx()
	if err != nil {
		log.WithContext(ctx).Println(err)
		return err
	}

	for key, val := range structs.Map(changes) {
		log.WithContext(ctx).Debugf("UPDATE content.users SET %s = '%s' WHERE id = %d", key, val, id)
		stmt := fmt.Sprintf("UPDATE content.users SET %s = $1 WHERE id = $2", key)
		_, err = tx.Exec(stmt, val, id)
		if err != nil {
			log.WithContext(ctx).Println(err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.WithContext(ctx).Println(err)
		return err
	}

	return nil
}