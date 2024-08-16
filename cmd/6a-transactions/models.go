package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

type operationType string

const (
	read  operationType = "r"
	write operationType = "w"
)

type txnRequest struct {
	Type        string      `json:"type"`
	Transaction []operation `json:"txn"`
}

type txnResponse struct {
	Type        string      `json:"type"`
	Transaction []operation `json:"txn"`
}

type operation struct {
	operationType operationType
	key           int
	value         *int
}

// unmarshal array of the form ['r', 1, nil] into an operation struct
func (o *operation) UnmarshalJSON(p []byte) error {
	tmp := []interface{}{}
	if err := json.Unmarshal(p, &tmp); err != nil {
		return err
	}

	op, ok := tmp[0].(string)
	if !ok {
		return errors.New(fmt.Sprintf("could not deserialize as an operation: %v", tmp[0]))
	}

	key, ok := tmp[1].(float64)
	if !ok {
		return errors.New(fmt.Sprintf("could not deserialize as a number: %V", tmp[1]))
	}

	var value *int
	switch v := tmp[2].(type) {
	case nil:
		value = nil
	case float64:
		value = intptr(int(v))
	default:
		return errors.New(fmt.Sprintf("could not deserialize as a nullable number: %v", tmp[2]))
	}

	o.operationType = operationType(op)
	o.key = int(key)
	o.value = value

	return nil
}

// marshal transaction struct into an array of the form ['r', 1, nil]
func (o *operation) MarshalJSON() ([]byte, error) {
	tmp := []interface{}{}
	tmp = append(tmp, o.operationType, o.key, o.value)

	raw, err := json.Marshal(tmp)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

func intptr(i int) *int {
	return &i
}
