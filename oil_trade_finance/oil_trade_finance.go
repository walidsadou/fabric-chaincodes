/*******************************************************************************
Copyright (c) 2016 IBM Corporation and other Contributors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and limitations under the License.
Contributors:
Sumabala Nair - Initial Contribution
Kim Letkeman - Initial Contribution
Sumabala Nair - Updated for hyperledger May 2016
Sumabala Nair - Partial updates added May 2016
******************************************************************************/
//SN: March 2016

// IoT Blockchain Simple Smart Contract v 1.0

// This is a simple contract that creates a CRUD interface to
// create, read, update and delete an asset

//go:generate go run scripts/generate_go_schema.go

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

// CONTRACTSTATEKEY is used to store contract state into world state
const CONTRACTSTATEKEY string = "ContractStateKey"

// MYVERSION and DEFAULTSTATUS must be used to deploy the contract
const MYVERSION string = "1.0"
const DEFAULTSTATUS uint8 = 0

// TRADESTATEKEY is used to store trade state into world state
const TRADESTATEKEY string = "TradeStateKey"

// TRADEID is the id associated to the trade
const TRADEID string = "0476219"

// ************************************
// asset and contract state
// ************************************

// ContractState holds the contract version
type ContractState struct {
	Version string `json:"version"`
	Status  uint8  `json:"status"`
}

// Geolocation stores lat and long
type Geolocation struct {
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// AssetState stores current state for any assset
type AssetState struct {
	AssetID        *string      `json:"assetID,omitempty"`        // all assets must have an ID, primary key of contract
	Location       *Geolocation `json:"location,omitempty"`       // current asset location
	MaxTemperature *float64     `json:"maxTemperature,omitempty"` // asset temp
	MaxHumidity    *float64     `json:"maxHumidity,omitempty"`    // asset humidity
	Carrier        *string      `json:"carrier,omitempty"`        // the name of the carrier
	//Event          *Event       `json:"event,omitempty"`
}

type Event struct {
	Name *string `json:"name,omitempty"` // name of the Watson IoT event received
	Date *string `json:"date,omitempty"` // date of reception
}

type TradeState struct {
	TradeID string `json:"tradeID"`
}

// ************************************
// deploy callback mode
// ************************************

// Init is called during deploy
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var contractStateArg ContractState
	var tradeStateArg TradeState
	var err error

	if len(args) != 2 {
		return nil, errors.New("init expects 2 arguments, a JSON string with tagged version string and the id of the trade")
	}

	// handle contract state
	err = json.Unmarshal([]byte(args[0]), &contractStateArg)
	if err != nil {
		return nil, errors.New("Version argument unmarshal failed: " + fmt.Sprint(err))
	}
	if contractStateArg.Version != MYVERSION {
		return nil, errors.New("Contract version " + MYVERSION + " must match version argument: " + contractStateArg.Version)
	}
	// set status to default (0)
	contractStateArg.Status = DEFAULTSTATUS

	contractStateJSON, err := json.Marshal(contractStateArg)
	if err != nil {
		return nil, errors.New("Marshal failed for contract state" + fmt.Sprint(err))
	}
	err = stub.PutState(CONTRACTSTATEKEY, contractStateJSON)
	if err != nil {
		return nil, errors.New("Contract state failed PUT to ledger: " + fmt.Sprint(err))
	}

	// handle trade state
	err = json.Unmarshal([]byte(args[1]), &tradeStateArg)
	if err != nil {
		return nil, errors.New("Trade id argument unmarshal failed: " + fmt.Sprint(err))
	}
	if tradeStateArg.TradeID != TRADEID {
		return nil, errors.New("Trade id " + TRADEID + " must match trade id: " + tradeStateArg.TradeID)
	}
	tradeStateJSON, err := json.Marshal(tradeStateArg)
	if err != nil {
		return nil, errors.New("Marshal failed for trade state" + fmt.Sprint(err))
	}
	err = stub.PutState(TRADESTATEKEY, tradeStateJSON)
	if err != nil {
		return nil, errors.New("Trade state failed PUT to ledger: " + fmt.Sprint(err))
	}

	return nil, nil
}

// ************************************
// deploy and invoke callback mode
// ************************************

// Invoke is called when an invoke message is received
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// Handle different functions
	if function == "createAsset" {
		// create assetID
		return t.createAsset(stub, args)
	} else if function == "updateAsset" {
		// create assetID
		return t.updateAsset(stub, args)
	} else if function == "deleteAsset" {
		// Deletes an asset by ID from the ledger
		return t.deleteAsset(stub, args)
	}
	return nil, errors.New("Received unknown invocation: " + function)
}

// ************************************
// query callback mode
// ************************************

// Query is called when a query message is received
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	// Handle different functions
	if function == "readAsset" {
		// gets the state for an assetID as a JSON struct
		return t.readAsset(stub, args)
	} else if function == "readTradeState" {
		// get trade state as a JSON struct
		return t.readTradeState(stub, args)
	} else if function == "readAssetSamples" {
		// returns selected sample objects
		return t.readAssetSamples(stub, args)
	} else if function == "readAssetSchemas" {
		// returns selected sample objects
		return t.readAssetSchemas(stub, args)
	} else if function == "readContractState" {
		return t.readContractState(stub, args)
	}

	return nil, errors.New("Received unknown invocation: " + function)
}

/**********main implementation *************/

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple Chaincode: %s", err)
	}
}

/*****************ASSET CRUD INTERFACE starts here************/

/****************** 'deploy' methods *****************/

/******************** createAsset ********************/

func (t *SimpleChaincode) createAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	_, erval := t.createOrUpdateAsset(stub, args)
	return nil, erval
}

//******************** updateAsset ********************/

func (t *SimpleChaincode) updateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	_, erval := t.createOrUpdateAsset(stub, args)
	return nil, erval
}

//******************** deleteAsset ********************/

func (t *SimpleChaincode) deleteAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string // asset ID
	var err error
	var stateIn AssetState

	// validate input data for number of args, Unmarshaling to asset state and obtain asset id
	stateIn, err = t.validateInput(args)
	if err != nil {
		return nil, err
	}
	assetID = *stateIn.AssetID
	// Delete the key / asset from the ledger
	err = stub.DelState(assetID)
	if err != nil {
		err = errors.New("DELSTATE failed! : " + fmt.Sprint(err))
		return nil, err
	}
	return nil, nil
}

/******************* Query Methods ***************/

//********************readAsset********************/

func (t *SimpleChaincode) readAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string // asset ID
	var err error
	var state AssetState

	// validate input data for number of args, Unmarshaling to asset state and obtain asset id
	stateIn, err := t.validateInput(args)
	if err != nil {
		return nil, errors.New("Asset does not exist!")
	}
	assetID = *stateIn.AssetID
	// Get the state from the ledger
	assetBytes, err := stub.GetState(assetID)
	if err != nil || len(assetBytes) == 0 {
		err = errors.New("Unable to get asset state from ledger")
		return nil, err
	}
	err = json.Unmarshal(assetBytes, &state)
	if err != nil {
		err = errors.New("Unable to unmarshal state data obtained from ledger")
		return nil, err
	}
	return assetBytes, nil
}

//********************readTradeState********************/

func (t *SimpleChaincode) readTradeState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	var state TradeState

	if len(args) != 0 {
		err = errors.New("Too many arguments. Expecting none.")
		return nil, err
	}

	// Get the state from the ledger
	tradeBytes, err := stub.GetState(TRADESTATEKEY)
	if err != nil || len(tradeBytes) == 0 {
		err = errors.New("Unable to get trade state from ledger")
		return nil, err
	}
	err = json.Unmarshal(tradeBytes, &state)
	if err != nil {
		err = errors.New("Unable to unmarshal state data obtained from ledger")
		return nil, err
	}
	return tradeBytes, nil
}

//********************readContractState********************/

func (t *SimpleChaincode) readContractState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error
	var state ContractState

	if len(args) != 0 {
		err = errors.New("Too many arguments. Expecting none.")
		return nil, err
	}

	// Get the state from the ledger
	contractStateBytes, err := stub.GetState(CONTRACTSTATEKEY)
	if err != nil || len(contractStateBytes) == 0 {
		err = errors.New("Unable to get contract state from ledger")
		return nil, err
	}
	err = json.Unmarshal(contractStateBytes, &state)
	if err != nil {
		err = errors.New("Unable to unmarshal state data obtained from ledger")
		return nil, err
	}
	return contractStateBytes, nil
}

//*************readContractObjectModel*****************/

// func (t *SimpleChaincode) readContractObjectModel(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
// 	var state = ContractState{}

// 	// Marshal and return
// 	stateJSON, err := json.Marshal(state)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return stateJSON, nil
// }

//*************readAssetObjectModel*****************/

// func (t *SimpleChaincode) readAssetObjectModel(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
// 	var state AssetState = AssetState{}

// 	// Marshal and return
// 	stateJSON, err := json.Marshal(state)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return stateJSON, nil
// }

//*************readAssetSamples*******************/

func (t *SimpleChaincode) readAssetSamples(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	return []byte(samples), nil
}

//*************readAssetSchemas*******************/

func (t *SimpleChaincode) readAssetSchemas(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	return []byte(schemas), nil
}

// ************************************
// validate input data : common method called by the CRUD functions
// ************************************
func (t *SimpleChaincode) validateInput(args []string) (stateIn AssetState, err error) {
	var assetID string       // asset ID
	var state = AssetState{} // The calling function is expecting an object of type AssetState

	if len(args) != 1 {
		err = errors.New("Incorrect number of arguments. Expecting a JSON strings with mandatory assetID")
		return state, err
	}
	jsonData := args[0]
	assetID = ""
	stateJSON := []byte(jsonData)
	err = json.Unmarshal(stateJSON, &stateIn)
	if err != nil {
		err = errors.New("Unable to unmarshal input JSON data")
		return state, err
		// state is an empty instance of asset state
	}
	// was assetID present?
	// The nil check is required because the asset id is a pointer.
	// If no value comes in from the json input string, the values are set to nil

	if stateIn.AssetID != nil {
		assetID = strings.TrimSpace(*stateIn.AssetID)
		if assetID == "" {
			err = errors.New("AssetID not passed")
			return state, err
		}
	} else {
		err = errors.New("Asset id is mandatory in the input JSON data")
		return state, err
	}

	stateIn.AssetID = &assetID
	return stateIn, nil
}

//******************** createOrUpdateAsset ********************/

func (t *SimpleChaincode) createOrUpdateAsset(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var assetID string // asset ID                    // used when looking in map
	var err error
	var stateIn AssetState
	var stateStub AssetState

	// validate input data for number of args, Unmarshaling to asset state and obtain asset id

	stateIn, err = t.validateInput(args)
	if err != nil {
		return nil, err
	}
	assetID = *stateIn.AssetID
	// Partial updates introduced here
	// Check if asset record existed in stub
	assetBytes, err := stub.GetState(assetID)
	if err != nil || len(assetBytes) == 0 {
		// This implies that this is a 'create' scenario
		stateStub = stateIn // The record that goes into the stub is the one that cme in
	} else {
		// This is an update scenario
		err = json.Unmarshal(assetBytes, &stateStub)
		if err != nil {
			err = errors.New("Unable to unmarshal JSON data from stub")
			return nil, err
			// state is an empty instance of asset state
		}
		// Merge partial state updates
		stateStub, err = t.mergePartialState(stateStub, stateIn)
		if err != nil {
			err = errors.New("Unable to merge state")
			return nil, err
		}
	}
	stateJSON, err := json.Marshal(stateStub)
	if err != nil {
		return nil, errors.New("Marshal failed for contract state" + fmt.Sprint(err))
	}
	// Get existing state from the stub

	// Write the new state to the ledger
	err = stub.PutState(assetID, stateJSON)
	if err != nil {
		err = errors.New("PUT ledger state failed: " + fmt.Sprint(err))
		return nil, err
	}
	return nil, nil
}

/*********************************  internal: mergePartialState ****************************/
func (t *SimpleChaincode) mergePartialState(oldState AssetState, newState AssetState) (AssetState, error) {

	old := reflect.ValueOf(&oldState).Elem()
	new := reflect.ValueOf(&newState).Elem()

	for i := 0; i < old.NumField(); i++ {
		oldOne := old.Field(i)
		newOne := new.Field(i)
		if !reflect.ValueOf(newOne.Interface()).IsNil() {
			oldOne.Set(reflect.Value(newOne))
		}
	}

	return oldState, nil
}
// --------------------------------ALERTS-----------------------------------------


var AlertsName = map[int]string{
	0: "OVERTEMP",
	1: "overhum",
}

var AlertsValue = map[string]int32{
	"OVERTEMP": 0,
	"overhum": 1,
}


func (x Alerts) String() string {
	return AlertsName[int(x)]
}


type AlertNameArray []string

type Alerts int32

const (
    // AlertsOVERTEMP the over temperature alert 
 	AlertsOVERTEMP Alerts = 0
    // AlertsSIZE is to be maintained always as 1 greater than the last alert, giving a size  
	AlertsOVERHUM        Alerts = 1

	AlertsSIZE.  Alerts = 2
)

type AlertArrayInternal [AlertsSIZE]bool

var NOALERTSACTIVEINTERNAL = AlertArrayInternal{}

var NOALERTSACTIVE = AlertNameArray{}

type AlertStatus struct {
    Active  AlertNameArray  `json:"active"`
    Raised  AlertNameArray  `json:"raised"`
    Cleared AlertNameArray  `json:"cleared"`
}
type AlertStatusInternal struct {
    Active  AlertArrayInternal  
    Raised  AlertArrayInternal  
    Cleared AlertArrayInternal  
}

func (a *AlertStatus) asAlertStatusInternal() (AlertStatusInternal) {
    var aOut = AlertStatusInternal{}
    for i := range a.Active {
        aOut.Active[AlertsValue[a.Active[i]]] = true
    }
    for i := range a.Raised {
        aOut.Raised[AlertsValue[a.Raised[i]]] = true
    }
    for i := range a.Cleared {
        aOut.Cleared[AlertsValue[a.Cleared[i]]] = true
    }
    return aOut
}

func (a *AlertStatusInternal) asAlertStatus() (AlertStatus) {
    var aOut = newAlertStatus()
    for i := range a.Active {
        if a.Active[i] {
            aOut.Active = append(aOut.Active, AlertsName[i])
        }
    }
    for i := range a.Raised {
        if a.Raised[i] {
            aOut.Raised = append(aOut.Raised, AlertsName[i])
        }
    }
    for i := range a.Cleared {
        if a.Cleared[i] {
            aOut.Cleared = append(aOut.Cleared, AlertsName[i])
        }
    }
    return aOut
}

func (a *AlertStatusInternal) clearRaisedAndClearedStatus () {
    for i := range a.Active {
        if a.Active[i] {
            a.Raised[i] = false
        } else {
            a.Cleared[i] = false
        }
    }
}
func (a *AlertStatusInternal) raiseAlert (alert Alerts) {
    if a.Active[alert] {
        // already raised
        // this is tricky, should not say this event raised an
        // active alarm, as it makes it much more difficult to track
        // the exact moments of transition
        a.Active[alert] = true
        a.Raised[alert] = false
        a.Cleared[alert] = false
    } else {
        // raising it
        a.Active[alert] = true
        a.Raised[alert] = true
        a.Cleared[alert] = false
    }
}

func (a *AlertStatusInternal) clearAlert (alert Alerts) {
    if a.Active[alert] {
        // clearing alert
        a.Active[alert] = false
        a.Raised[alert] = false
        a.Cleared[alert] = true
    } else {
        // was not active
        a.Active[alert] = false
        a.Raised[alert] = false
        // this is tricky, should not say this event cleared an
        // inactive alarm, as it makes it much more difficult to track
        //  the exact moments of transition
        a.Cleared[alert] = false
    }
}

func newAlertStatus() (AlertStatus) {
    var a AlertStatus
    a.Active = make([]string, 0, AlertsSIZE)
    a.Raised = make([]string, 0, AlertsSIZE)
    a.Cleared = make([]string, 0, AlertsSIZE)
    return a
}

func (arr *AlertNameArray) copyFrom (s []interface{}) {
    // a conversion like this must assert type at every level
    for i := 0; i < len(s); i++ {
        *arr = append(*arr, s[i].(string))
    }
}

// NoAlertsActive returns true when no alerts are active in the asset's status at this time
func (arr *AlertStatusInternal) NoAlertsActive() (bool) {
    return (arr.Active == NOALERTSACTIVEINTERNAL)
}

// AllClear returns true when no alerts are active, raised or cleared in the asset's status at this time
func (arr *AlertStatusInternal) AllClear() (bool) {
    return  (arr.Active == NOALERTSACTIVEINTERNAL) &&
            (arr.Raised == NOALERTSACTIVEINTERNAL) &&
            (arr.Cleared == NOALERTSACTIVEINTERNAL) 
}

// NoAlertsActive returns true when no alerts are active in the asset's status at this time
func (a *AlertStatus) NoAlertsActive() (bool) {
    return len(a.Active) == 0
}

// AllClear returns true when no alerts are active, raised or cleared in the asset's status at this time
func (a *AlertStatus) AllClear() (bool) {
    return  len(a.Active) == 0 &&
            len(a.Raised) == 0 &&
            len(a.Cleared) == 0 
}

//------------------------------- rules ---------------------------------

func (a *ArgsMap) executeRules(alerts *AlertStatus) (bool, error) {
    log.Debugf("Executing rules input: %+v", *alerts)
    // transform external to internal for easy alert status processing
    var internal = (*alerts).asAlertStatusInternal()

    internal.clearRaisedAndClearedStatus()

    // ------ validation rules
    // rule 1 -- test validation failure
    err := internal.testValidationRule(a)
    // return value is not used, return true, which means noncompliant
    if err != nil {return true, err}
    // rule 2 -- ???

    // ------ alert rules
    // rule 1 -- overtemp
    err = internal.overTempRule(a)
    if err != nil {return true, err}
    // rule 2 -- overhum
    err = internal.overHumRule(a)
    if err != nil {return true, err}

    // transform for external consumption
    *alerts = internal.asAlertStatus()
    log.Debugf("Executing rules output: %+v", *alerts)

    // set compliance true means out of compliance
    compliant, err := internal.calculateContractCompliance(a) 
    if err != nil {return true, err} 
    // returns true if anything at all is active (i.e. NOT compliant)
    return !compliant, nil
}

func (alerts *AlertStatusInternal) testValidationRule (a *ArgsMap) error {
    tbytes, found := getObject(*a, "testValidation")
    if found {
        t, found := tbytes.(bool)
        if found {
            if t {
                err := errors.New("testValidation property found and is true") 
                return err
            }
        }
    }
    return nil
}


func (alerts *AlertStatusInternal) overTempRule (a *ArgsMap) error {
    const temperatureThreshold  float64 = 60 // (inclusive good value)

    tbytes, found := getObject(*a, "MaxTemperature")
    if found {
        t, found := tbytes.(float64)
        if found {
            if t > temperatureThreshold {
                alerts.raiseAlert(AlertsOVERTEMP)
                return nil
            }
        } else {
            log.Warning("overTempRule: temperature not type JSON Number, alert status not changed")
            // do nothing to the alerts status
            return nil 
        }
    }
    alerts.clearAlert(AlertsOVERTEMP)
    return nil
}

func (alerts *AlertStatusInternal) overHumRule (a *ArgsMap) error {
    const HumidityThreshold  float64 = 80 // (inclusive good value)

    tbytes, found := getObject(*a, "MaxHumidity")
    if found {
        t, found := tbytes.(float64)
        if found {
            if t > HumidityThreshold {
                alerts.raiseAlert(AlertsOVERHUM)
                return nil
            }
        } else {
            log.Warning("overHumRule: humidity not type JSON Number, alert status not changed")
            // do nothing to the alerts status
            return nil 
        }
    }
    alerts.clearAlert(AlertsOVERHUM)
    return nil
}

func (alerts *AlertStatusInternal) calculateContractCompliance (a *ArgsMap) (bool, error) {
    // a simplistic calculation for this particular contract, but has access
    // to the entire state object and can thus have at it
    // compliant is no alerts active
    return alerts.NoAlertsActive(), nil
    // NOTE: There could still a "cleared" alert, so don't go
    //       deleting the alerts from the ledger just on this status.
}

   


