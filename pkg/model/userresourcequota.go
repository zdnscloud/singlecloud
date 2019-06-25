package model

import (
	"fmt"
	"time"

	"github.com/zdnscloud/cement/rstore"
)

type UserResourceQuota struct {
	Id                string
	ClusterName       string `sql:"uk"`
	Namespace         string `sql:"uk"`
	UserName          string `sql:"uk"`
	CPU               string `sql:"uk"`
	Memory            string `sql:"uk"`
	Storage           string `sql:"uk"`
	RequestType       string `sql:"uk"`
	Status            string `sql:"uk"`
	Purpose           string
	RejectionReason   string
	CreationTimestamp time.Time
	ResponseTimestamp time.Time
	Requestor         string
	Telephone         string
}

func (u *UserResourceQuota) Validate() error {
	return nil
}

func SaveUserResourceQuotaToDB(quota *UserResourceQuota) (string, error) {
	tx, err := Begin()
	if err != nil {
		return "", err
	}
	u, err := tx.Insert(quota)
	if err != nil {
		tx.RollBack()
		return "", err
	}

	return u.(*UserResourceQuota).Id, tx.Commit()
}

func UpdateUserResourceQuotaToDB(quota *UserResourceQuota) error {
	tx, err := Begin()
	if err != nil {
		return err
	}

	defer tx.RollBack()
	if err := deleteUserResourceQuota(tx, quota.Id); err != nil {
		return err
	}

	if _, err := tx.Insert(quota); err != nil {
		return err
	}

	return tx.Commit()
}

func GetUserResourceQuotasFromDB() ([]UserResourceQuota, error) {
	tx, err := Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()
	return getUserResourceQuotas(tx, nil)
}

func GetUserResourceQuotaByIDFromDB(id string) (*UserResourceQuota, error) {
	tx, err := Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Commit()
	userResourceQuotas, err := getUserResourceQuotas(tx, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}

	if len(userResourceQuotas) == 0 {
		return nil, fmt.Errorf("user quota %s info is non-exists", id)
	}

	return &userResourceQuotas[0], nil
}

func DeleteUserResourceQuotaFromDB(id string) error {
	tx, err := Begin()
	if err != nil {
		return err
	}
	defer tx.RollBack()

	if err := deleteUserResourceQuota(tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

func deleteUserResourceQuota(tx rstore.Transaction, id string) error {
	if rows, err := tx.Delete("user_resource_quota", map[string]interface{}{"id": id}); err != nil {
		return err
	} else if rows == 0 {
		return fmt.Errorf("user quota %s info is non-exists", id)
	} else {
		return nil
	}
}

func getUserResourceQuotas(tx rstore.Transaction, conditions map[string]interface{}) ([]UserResourceQuota, error) {
	slice, err := tx.Get("user_resource_quota", conditions)
	if err != nil {
		return nil, err
	}

	quotas, ok := slice.([]UserResourceQuota)
	if ok == false {
		return nil, fmt.Errorf("db file corrupt")
	}

	return quotas, nil
}

func IsExistsNamespaceInDB(namespace string) (bool, error) {
	tx, err := Begin()
	if err != nil {
		return false, err
	}
	defer tx.Commit()

	quotas, err := getUserResourceQuotas(tx, map[string]interface{}{
		"namespace": namespace,
	})
	if err != nil {
		return false, err
	}

	return len(quotas) != 0, nil
}
