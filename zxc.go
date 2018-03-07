package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/zhixinchain/wallet"
)

const initialSupply = 500000000000000000
const tokenUnit      = "gas"
const owner = "1Ge2Q89tkZZLrWvNtLBaERBRdTDNXWQwhF"

type Transaction struct{
	Address      string  `json:"address"` //交易对方账户地址
	Type         string  `json:"type"`    //in--标示转入,out-标示转出
	Amount       int64   `json:"amount"`  //交易的总金额
	Time         string  `json:"time"`    //交易时间
}

type Token struct{
	IsFrozen bool       `json:"isFrozen"`   //账号是否被冻结
	Amount   int64      `json:"amount"`     //账户金额
	Unit     string     `json:"unit"`       //单位
	TxInfo Transaction  `json:"txInfo"`     //本次交易信息
}

type TokenChaincode struct {
}


func getDefaultToken() Token{
	token := Token{}
	token.IsFrozen = false
	token.Amount = 0
	token.Unit = tokenUnit
	token.TxInfo = Transaction{}
	return token
}

func getTime () string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func (t *TokenChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("zhixintoke Init")
	var err error
	var initialTokenBytes []byte
	var initialToken Token

	fmt.Printf("zhixin toke init,owner:%s,initial supply:%f", owner, initialSupply)

	//wirte initial token to owner
	initialToken = getDefaultToken()
	initialToken.Amount = initialSupply 
	initialToken.TxInfo.Address = "coin chaincode init"
	initialToken.TxInfo.Type = "in"
	initialToken.TxInfo.Amount = initialSupply
	initialToken.TxInfo.Time = getTime()

 	initialTokenBytes, err = json.Marshal(initialToken)
	if err != nil {
		return shim.Error("init initial token json marshal err")
	}

	if initialTokenBytes == nil {
		return shim.Error("init initial token json marshal err")
	}

	err = stub.PutState(owner, initialTokenBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *TokenChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("zhixintoken Invoke")
	function, args := stub.GetFunctionAndParameters()
	if function == "invoke" {
		// Make payment of X units from A to B
		return t.invoke(stub, args)
	} else if function == "query" {
		// the old "Query" is now implemtned in invoke
		return t.query(stub, args)
	} else if function == "getHistory" {
		return t.getHistory(stub, args)
	}

	return shim.Error("Invalid invoke function name. Expecting \"invoke\" \"delete\" \"query\"")
}

// Transaction makes payment of X units from A to B
func (t *TokenChaincode) invoke(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var X int64          // Transaction value
	var err error
	var Avalbytes, Bvalbytes []byte

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	APrivateHash := args[0] //todo private to address
	AWallet := wallet.SetWallet(APrivateHash)
	AAddressHash :=AWallet.GetAddress()

	BAddressHash := args[1]


	// Get the state from the ledger
	// TODO: will be nice to have a GetAllState call to ledger
	Avalbytes, err = stub.GetState(AAddressHash)
	if err != nil {
		return shim.Error("Failed to get account state")
	}
	if Avalbytes == nil {
		return shim.Error("Acount entity not found")
	}

	Atoken := Token{}
	Btoken := Token{}

	json.Unmarshal(Avalbytes, &Atoken)

	Bvalbytes, err = stub.GetState(BAddressHash)
	if err != nil {
		return shim.Error("Failed to get account state")
	}
	//返回nill说明账户不存在
	if Bvalbytes == nil {
		Btoken = getDefaultToken()
	} else {
		json.Unmarshal(Bvalbytes, &Btoken)
	}


	// Perform the execution
	X, err = strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return shim.Error("Invalid transaction amount, expecting a integer value")
	}

	if Atoken.Amount <=0 {
		return shim.Error("The balance is empty")
	}

	if Atoken.Amount < X {
		return shim.Error("Lack of balance")
	}

	if X < 0 {
		return shim.Error("Transfer amount cannot be negative")
	}

	if Atoken.Amount - X > Atoken.Amount {
		return shim.Error("Transfer amount err")
	}

	if Btoken.Amount + X < Btoken.Amount {
		return shim.Error("Transfer amount err")
	}

	nowTime := getTime()

	Atoken.Amount -= X
	Atoken.TxInfo.Address = BAddressHash
	Atoken.TxInfo.Amount = X
	Atoken.TxInfo.Type = "out"
	Atoken.TxInfo.Time = nowTime


	Btoken.Amount += X
	Btoken.TxInfo.Address = AAddressHash
	Btoken.TxInfo.Amount = X
	Btoken.TxInfo.Type = "in"
	Btoken.TxInfo.Time = nowTime

	fmt.Printf("Atoken.amount = %f, Btoken.amount = %f\n", Atoken.Amount, Btoken.Amount)

	// Write the state back to the ledger
	AtokenBytes, _ := json.Marshal(Atoken)
	err = stub.PutState(AAddressHash, AtokenBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	BtokenBytes, _ := json.Marshal(Btoken)
	err = stub.PutState(BAddressHash, BtokenBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// query callback representing the query of a chaincode
func (t *TokenChaincode) query(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var A string // Entities
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	A = args[0]

	// Get the state from the ledger
	Avalbytes, err := stub.GetState(A)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + A + "\"}"
		return shim.Error(jsonResp)
	}


	if Avalbytes == nil {
		Atoken :=getDefaultToken()
		Avalbytes, _= json.Marshal(Atoken)
	}

	//jsonResp := "{\"publicKey\":\"" + args[0] + "\",\"Amount\":\"" + strconv.FormatFloat(Atoken.Amount, 'f', 16, 64) + "\"}"
	//fmt.Printf("Query Response:%s\n", jsonResp)

	return shim.Success(Avalbytes)
}

func (t *TokenChaincode) getHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	type TokenHistory struct {
		TxId    string   `json:"txId"`
		Value   Token    `json:"value"`
	}
	var history []TokenHistory;
	var token Token

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	tokenAddress := args[0]
	fmt.Printf("- start getHistoryForToken: %s\n", tokenAddress)

	// Get History
	resultsIterator, err := stub.GetHistoryForKey(tokenAddress)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var tx TokenHistory
		tx.TxId = historyData.TxId                     //copy transaction id over
		json.Unmarshal(historyData.Value, &token)     //un stringify it aka JSON.parse()
		if historyData.Value == nil {                  //token has been deleted
			emptyToken := getDefaultToken()
			tx.Value = emptyToken                 //copy nil token
		} else {
			json.Unmarshal(historyData.Value, &token) //un stringify it aka JSON.parse()
			tx.Value = token                      //copy token over
		}
		history = append(history, tx)              //add this tx to the list
	}
	fmt.Printf("- getHistoryForMarble returning:\n%s", history)

	//change to array of bytes
	historyAsBytes, _ := json.Marshal(history)     //convert to array of bytes
	return shim.Success(historyAsBytes)
}

func main() {
	err := shim.Start(new(TokenChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
