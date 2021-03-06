package repo

import (
	"errors"
	"fmt"
	"log"

	"github.com/maximthomas/gortas/pkg/models"
	"gopkg.in/ldap.v3"
)

type UserLdapRepository struct {
	Address        string
	BindDN         string
	Password       string
	BaseDN         string
	ObjectClasses  []string
	UserAttributes []string
}

func (ur *UserLdapRepository) getConnection() (*ldap.Conn, error) {
	conn, err := ldap.Dial("tcp", ur.Address)
	if err != nil {
		return nil, err
	}
	err = conn.Bind(ur.BindDN, ur.Password)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (ur *UserLdapRepository) getLdapEntry(id string, conn *ldap.Conn) (*ldap.Entry, error) {
	fields := append([]string{"dn", "uid"}, ur.UserAttributes...)
	result, err := conn.Search(ldap.NewSearchRequest(
		ur.BaseDN,
		ldap.ScopeSingleLevel,
		ldap.NeverDerefAliases,
		0,
		100,
		false,
		fmt.Sprintf("(uid=%v)", id),
		fields,
		nil,
	))
	if err != nil {
		log.Print(err)
		return nil, err
	}

	if len(result.Entries) != 1 {
		return nil, fmt.Errorf("found multiple entries %v", result.Entries)
	}

	return result.Entries[0], nil
}

func (ur *UserLdapRepository) GetUser(id string) (user models.User, exists bool) {
	conn, err := ur.getConnection()
	if err != nil {
		log.Print(err)
		return user, exists
	}
	defer conn.Close()

	entry, err := ur.getLdapEntry(id, conn)
	if err != nil {
		log.Print(err)
		return user, exists
	}

	properties := make(map[string]string)
	for _, attr := range ur.UserAttributes {
		properties[attr] = entry.GetAttributeValue(attr)
	}

	user = models.User{
		ID:         entry.GetAttributeValue("uid"),
		Properties: properties,
	}
	exists = true

	return user, exists
}

func (ur *UserLdapRepository) ValidatePassword(id, password string) bool {
	conn, err := ur.getConnection()
	if err != nil {
		log.Print(err)
		return false
	}
	defer conn.Close()
	entry, err := ur.getLdapEntry(id, conn)
	if err != nil {
		log.Print(err)
		return false
	}

	if err := conn.Bind(entry.DN, password); err != nil {
		return false
	} else {
		return true
	}

}
func (ur *UserLdapRepository) CreateUser(user models.User) (models.User, error) {
	conn, err := ur.getConnection()
	if err != nil {
		log.Print(err)
		return user, err
	}
	defer conn.Close()
	dn := fmt.Sprintf("uid=%v,"+ur.BaseDN, user.ID)
	addRequest := ldap.NewAddRequest(dn, nil)
	addRequest.Attribute("objectClass", ur.ObjectClasses)
	addRequest.Attribute("sn", []string{user.ID})
	addRequest.Attribute("cn", []string{user.ID})
	err = conn.Add(addRequest)
	if err != nil {
		log.Print(err)
		return user, err
	}
	return user, err

}
func (ur *UserLdapRepository) UpdateUser(user models.User) error {
	return errors.New("not implemented")
}

func (ur *UserLdapRepository) SetPassword(id, password string) error {
	conn, err := ur.getConnection()
	if err != nil {
		log.Print(err)
		return err
	}
	defer conn.Close()
	entry, err := ur.getLdapEntry(id, conn)
	if err != nil {
		log.Print(err)
		return err
	}

	passwordModifyRequest := ldap.NewPasswordModifyRequest(entry.DN, "", password)
	_, err = conn.PasswordModify(passwordModifyRequest)

	if err != nil {
		log.Fatalf("Password could not be changed: %s", err.Error())
	}
	return nil
}
