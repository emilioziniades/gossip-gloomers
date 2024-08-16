package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestDeserialize(t *testing.T) {
	msg := `{
  "type": "txn",
  "msg_id": 3,
  "txn": [
    ["r", 1, null],
    ["r", 2, 42],
    ["w", 1, 6],
    ["w", 2, 9]
  ]
}`
	var body txnRequest
	if err := json.Unmarshal([]byte(msg), &body); err != nil {
		t.Errorf("could not unmarshal json: %v", err)
	}

	if body.Type != "txn" {
		t.Errorf("expected txt, got %v", body.Type)
	}

	expectedTxn := []operation{
		{operationType: "r", key: 1, value: nil},
		{operationType: "r", key: 2, value: intptr(42)},
		{operationType: "w", key: 1, value: intptr(6)},
		{operationType: "w", key: 2, value: intptr(9)},
	}

	if !reflect.DeepEqual(body.Transaction, expectedTxn) {
		t.Errorf("expected %v, got %v", expectedTxn, body.Transaction)
	}
}

func TestSerialize(t *testing.T) {
	msg := txnResponse{
		Type: "txn_ok",
		Transaction: []operation{
			{operationType: "r", key: 1, value: intptr(3)},
			{operationType: "w", key: 1, value: intptr(6)},
			{operationType: "w", key: 2, value: intptr(9)},
		},
	}

	rawMsg, err := json.Marshal(msg)

	if err != nil {
		t.Errorf("could not marshal json: %v", err)
	}

	expectedMsg := `{"type":"txn_ok","txn":[["r",1,3],["w",1,6],["w",2,9]]}`

	if string(rawMsg) != expectedMsg {
		t.Errorf(fmt.Sprintf("expected:\n%v\n\ngot:\n%v", expectedMsg, string(rawMsg)))
	}
}
