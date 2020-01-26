package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	"golang.org/x/crypto/bcrypt"
)

type KVStoreApplication struct {
	db           *badger.DB
	currentBatch *badger.Txn
}

type Patient struct {
	Name string `json:"name"`
	Age  string `json:"age"`
	Sick string `json:"sick"`
}

//NewKVStoreApplication ...
func NewKVStoreApplication(db *badger.DB) *KVStoreApplication {
	return &KVStoreApplication{
		db: db,
	}
}

var _ abcitypes.Application = (*KVStoreApplication)(nil)

//Info ...
func (KVStoreApplication) Info(req abcitypes.RequestInfo) abcitypes.ResponseInfo {
	return abcitypes.ResponseInfo{}
}

//SetOption ...
func (KVStoreApplication) SetOption(req abcitypes.RequestSetOption) abcitypes.ResponseSetOption {
	return abcitypes.ResponseSetOption{}
}

//DeliverTx ...
func (app *KVStoreApplication) DeliverTx(req abcitypes.RequestDeliverTx) abcitypes.ResponseDeliverTx {
	code := app.isValid(req.Tx)
	if code != 0 {
		return abcitypes.ResponseDeliverTx{Code: code}
	}

	parts := bytes.Split(req.Tx, []byte("="))
	key, value := parts[0], parts[1]

	err := app.currentBatch.Set(key, value)
	if err != nil {
		panic(err)
	}

	return abcitypes.ResponseDeliverTx{Code: 0}
}

func (app *KVStoreApplication) isValid(tx []byte) (code uint32) {
	// check format
	parts := bytes.Split(tx, []byte("="))
	if len(parts) != 2 {
		return 1
	}

	key, value := parts[0], parts[1]

	// check if the same key=value already exists
	err := app.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if err == nil {
			return item.Value(func(val []byte) error {
				if bytes.Equal(val, value) {
					code = 2
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return code
}

//CheckTx ...
func (app *KVStoreApplication) CheckTx(req abcitypes.RequestCheckTx) abcitypes.ResponseCheckTx {
	code := app.isValid(req.Tx)
	return abcitypes.ResponseCheckTx{Code: code, GasWanted: 1}
}

//Commit ...
func (app *KVStoreApplication) Commit() abcitypes.ResponseCommit {
	app.currentBatch.Commit()
	return abcitypes.ResponseCommit{Data: []byte{}}
}

//Bhash ...
func Bhash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

//CheckHash ...
func CheckHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

//Query ...
func (app *KVStoreApplication) Query(reqQuery abcitypes.RequestQuery) (resQuery abcitypes.ResponseQuery) {
	resQuery.Key = reqQuery.Data
	err := app.db.View(func(txn *badger.Txn) error {

		if strings.Contains(string(reqQuery.Data), "patient_search_") {

			id := strings.Split(string(reqQuery.Data), "patient_search_")[1]

			nameb, errA := txn.Get([]byte("patient_name_" + id))
			if errA != nil && errA != badger.ErrKeyNotFound {
				resQuery.Log = "does not exist"
				return errA
			}
			ageb, errB := txn.Get([]byte("patient_age_" + id))
			if errB != nil && errB != badger.ErrKeyNotFound {
				resQuery.Log = "does not exist"
				return errB
			}
			sickb, errC := txn.Get([]byte("patient_sick_" + id))
			if errB != nil && errB != badger.ErrKeyNotFound {
				resQuery.Log = "does not exist"
				return errB
			}

			if errA != nil || errB != nil || errC != nil {
				resQuery.Log = "does not exist"
				return errA
			}

			var name, age, sick string
			fmt.Println("Yeehaw")

			nameb.Value(func(val []byte) error {
				name = string(val)
				return nil
			})
			ageb.Value(func(val []byte) error {
				age = string(val)
				return nil
			})
			sickb.Value(func(val []byte) error {
				sick = string(val)
				return nil
			})

			patient := Patient{name, age, sick}

			bytePatient, err := json.Marshal(patient)
			if err != nil {
				fmt.Println(err)
				return err
			}

			resQuery.Log = "exists"
			resQuery.Value = bytePatient
			return nil

		}

		item, err := txn.Get(reqQuery.Data)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		if err == badger.ErrKeyNotFound {
			resQuery.Log = "does not exist"
		} else {
			return item.Value(func(val []byte) error {
				resQuery.Log = "exists"
				resQuery.Value = val
				return nil
			})
		}
		return nil

	})
	if err != nil {
		fmt.Println(err)
	}
	return
}

//InitChain ...
func (KVStoreApplication) InitChain(req abcitypes.RequestInitChain) abcitypes.ResponseInitChain {
	return abcitypes.ResponseInitChain{}
}

//BeginBlock ...
func (app *KVStoreApplication) BeginBlock(req abcitypes.RequestBeginBlock) abcitypes.ResponseBeginBlock {
	app.currentBatch = app.db.NewTransaction(true)
	return abcitypes.ResponseBeginBlock{}
}

//EndBlock ...
func (KVStoreApplication) EndBlock(req abcitypes.RequestEndBlock) abcitypes.ResponseEndBlock {
	return abcitypes.ResponseEndBlock{}
}
