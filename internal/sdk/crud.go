// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package go_npm_proxy_manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	version "github.com/hashicorp/go-version"
)

// Basic CRUD methods.
type CRUD[O Object, OB ObjectBase, OC ObjectCreate] interface {
	List() ([]O, error)
	Get(id int) (O, error)
	Create(obj OC) (O, error)
	Update(id int, newProperties OB) (O, error)
	Delete(id int) error
	ClientVersion() *version.Version
	GetWriteOnlyFields() []string
}

// CRUD, but no Update function.
// This is mainly just for certificates.
type CRD[O Object, OB ObjectBase, OC ObjectCreate] interface {
	List() ([]O, error)
	Get(id int) (O, error)
	Create(obj OC) (O, error)
	Delete(id int) error
	ClientVersion() *version.Version
	GetWriteOnlyFields() []string
}

// Typical crud object.
type crudObject[O Object, OB ObjectBase, OC ObjectCreate] struct {
	client          *Client
	apiPath         string
	writeOnlyFields []string
}

// Common fields for all CRUD objects post creation.
type ObjectMeta struct {
	ID          int    `json:"id"`
	CreatedOn   string `json:"created_on"`
	ModifiedOn  string `json:"modified_on"`
	OwnerUserID int    `json:"owner_user_id"`
}

// The Object type is the CRUD object will all parameters.
type Object interface {
	User | ProxyHost | RedirectionHost |
		DeadHost | Stream | AccessList |
		Certificate
}

// The ObjectBase contains the mutable parameters, e.g Name or ForwardHost
// but not ID or CreatedOn.
type ObjectBase interface {
	UserBase | ProxyHostBase | RedirectionHostBase |
		DeadHostBase | StreamBase | AccessListBase |
		CertificateBase
}

// Sometimes the required parameters are different for creations, e.g Users
// are created with a password but it's not accepted for updates.
type ObjectCreate interface {
	UserCreate | ProxyHostBase | RedirectionHostBase |
		DeadHostBase | StreamBase | AccessListBase |
		CertificateBase
}

func getID[O Object](object O) int {
	return int(reflect.ValueOf(object).FieldByName("ID").Int())
}

// Return the lowercase name of a struct.
func getName(obj interface{}) string {
	return strings.ToLower(reflect.ValueOf(obj).Type().Name())
}

// List objects.
func _list[O Object](c *Client, apiPath string) (objList []O, err error) {

	var queryParams string
	switch apiPath {
	case "users":
		{
			queryParams = "?expand=permissions"
		}
	case "nginx/access-lists":
		{
			queryParams = "?expand=items,clients"
		}
	default:
		{
			queryParams = ""
		}
	}

	// Make the request to the API.
	response, err := c.NewRequest(
		fmt.Sprintf("%s%s", apiPath, queryParams),
		http.MethodGet,
		nil,
		true,
		nil,
	)
	if err != nil {
		return objList, fmt.Errorf("unable to list %s: %s", getName(objList), err)
	}

	// Unmarshal the response.
	err = json.Unmarshal(response, &objList)
	if err != nil {
		return objList, fmt.Errorf("unable to unmarshal json response: %s", err)
	}

	// Ensure the output is sanitised and any slice values aren't nil.
	for i := 0; i < len(objList); i++ {
		setZeroSlices(&objList[i])
	}

	return objList, nil

}

func (o crudObject[O, OB, OC]) List() ([]O, error) {
	return _list[O](o.client, o.apiPath)
}

// Get an object by ID.
func _get[O Object](c *Client, apiPath string, id int) (obj O, err error) {

	var allObjects []O
	allObjects, err = _list[O](c, apiPath)
	if err != nil {
		return obj, fmt.Errorf("unable to list %s objects: %s", getName(obj), err)
	}

	for _, o := range allObjects {
		if getID(o) == id {
			return o, nil
		}
	}

	return obj, fmt.Errorf("%s with id %d not found", getName(obj), id)
}
func (o crudObject[O, OB, OC]) Get(id int) (O, error) {
	return _get[O](o.client, o.apiPath, id)
}

// Create an object.
func _create[O Object, OC ObjectCreate](c *Client, apiPath string, obj OC) (createdObj O, err error) {

	// Marshal the object into json.
	body, err := json.Marshal(obj)
	if err != nil {
		return createdObj, fmt.Errorf("unable to marshal %s into json: %s", getName(createdObj), err)
	}

	// Make the request to the API.
	response, err := c.NewRequest(
		apiPath,
		http.MethodPost,
		&body,
		true,
		returnPointer("application/json"),
	)
	if err != nil {
		return createdObj, fmt.Errorf("error making request to create %s: %s", getName(createdObj), err)
	}

	// Unmarshal the response.
	err = json.Unmarshal(response, &createdObj)
	if err != nil {
		return createdObj, fmt.Errorf("unable to unmarshal json response: %s", err)
	}

	setZeroSlices(&createdObj)

	return createdObj, nil

}
func (o crudObject[O, OB, OC]) Create(obj OC) (O, error) {
	return _create[O, OC](o.client, o.apiPath, obj)
}

// Update an object.
func _update[O Object, OB ObjectBase](c *Client, apiPath string, id int, newProperties OB) (updatedObject O, err error) {

	// Make sure the object exists.
	_, err = _get[O](c, apiPath, id)
	if err != nil {
		return updatedObject, fmt.Errorf("unable to find %s with ID %d: %s", getName(updatedObject), id, err)
	}

	// Marshal the new properties into json.
	body, err := json.Marshal(newProperties)
	if err != nil {
		return updatedObject, fmt.Errorf("unable to marshal json body: %s", err)
	}

	// Make the request to update the object.
	response, err := c.NewRequest(
		fmt.Sprintf("%s/%d", apiPath, id),
		http.MethodPut,
		&body,
		true,
		returnPointer("application/json"),
	)
	if err != nil {
		return updatedObject, fmt.Errorf("unable to update details of %s ID %d: %s", getName(updatedObject), id, err)
	}

	// Unmarshal the response into an Object.
	err = json.Unmarshal(response, &updatedObject)
	if err != nil {
		return updatedObject, fmt.Errorf("unable to parse response to update %s ID %d: %s", getName(updatedObject), id, err)
	}

	return updatedObject, nil

}

func (o crudObject[O, OB, OC]) Update(id int, newProperties OB) (O, error) {
	return _update[O, OB](o.client, o.apiPath, id, newProperties)
}

// Delete an object.
func _delete[O Object](c *Client, apiPath string, id int) (err error) {

	_, err = _get[O](c, apiPath, id)
	if err != nil {
		return err
	}

	// Delete the object.
	response, err := c.NewRequest(
		fmt.Sprintf("%s/%d", apiPath, id),
		http.MethodDelete,
		nil,
		true,
		returnPointer("application/json"),
	)

	// Check for any errors. A successful response will return "true".
	if err != nil {
		return fmt.Errorf("unable to delete object with ID %d: %s", id, err)
	} else if string(response) != "true" {
		return fmt.Errorf("failed to delete object with ID %d: %s", id, string(response))
	}

	return nil

}
func (o crudObject[O, OB, OC]) Delete(id int) error {
	return _delete[O](o.client, o.apiPath, id)
}

// Enable or disable an object. NPM only regenerates the nginx config when
// this is done through the enable/disable endpoints, not through an update.
func (o crudObject[O, OB, OC]) SetEnabled(id int, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}

	_, err := o.client.NewRequest(
		fmt.Sprintf("%s/%d/%s", o.apiPath, id, action),
		http.MethodPost,
		nil,
		true,
		nil,
	)
	// NPM refuses to enable an enabled host or disable a disabled one, which
	// means there's nothing to do.
	if err != nil && strings.Contains(err.Error(), "already") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to %s object with ID %d: %s", action, id, err)
	}
	return nil
}

func (o crudObject[O, OB, OC]) ClientVersion() *version.Version {
	return o.client.Version
}

func (o crudObject[O, OB, OC]) GetWriteOnlyFields() []string {
	return o.writeOnlyFields
}
