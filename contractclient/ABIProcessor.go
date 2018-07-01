package contractclient

import (
	"strings"
	"github.com/ethereum/go-ethereum/crypto"
	"encoding/hex"
	"github.com/synechron-finlabs/quorum-maker-nodemanager/contracthandler"
	"fmt"
	"regexp"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type ParamTableRow struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var SupportedDatatypes = []*regexp.Regexp{
	regexp.MustCompile(`^(u?int[0-9]{0,3}|address)$`),
	regexp.MustCompile(`^bool$`),
	regexp.MustCompile(`^(u?int[0-9]{0,3}|address)\[[0-9]+\]$`),
	regexp.MustCompile(`^bytes$`),
	regexp.MustCompile(`^(u?int[0-9]{0,3}|address)\[\]$`),
	regexp.MustCompile(`^string$`),
	regexp.MustCompile(`^bytes32\[\]$`),
	regexp.MustCompile(`^bytes32\[[0-9]+\]$`)}

var abiMap = map[string]string{}
var funcSigMap = map[string]string{}
var funcParamNameMap = map[string]string{}
var funcNameMap = map[string]string{}

func ABIParser(contractAdd string, abiContent string, payload string) ([]ParamTableRow, string) {
	var abi abi.ABI
	if abiMap[contractAdd] == "" {
		abiMap[contractAdd] = abiContent
		abi.UnmarshalJSON([]byte(abiMap[contractAdd]))
		methodMap := abi.Methods
		size := 0
		for key := range methodMap {
			if methodMap[key].Const == false {
				size++
			}
		}
		functionSigs := make([]string, size)
		paramNames := make([]string, size)
		keccakHashes := make([]string, size)
		i := 1

		for key := range methodMap {
			if methodMap[key].Const == false {
				var funcSig string
				var params string
				for _, elem := range methodMap[key].Inputs {
					i := 0
					for _, v := range SupportedDatatypes {
						if v.MatchString(elem.Type.String()) != true {
							i++
						}
					}
					if (i == len(SupportedDatatypes)) {
						abiMap[contractAdd] = "Unsupported"
						decodeUnsupported := make([]ParamTableRow, 1)
						decodeUnsupported[0].Key = "decodeFailed"
						decodeUnsupported[0].Value = "Unsupported Datatype"
						return decodeUnsupported, ""
					}
					funcSig = funcSig + elem.Type.String() + ","
					params = params + elem.Name + ","
				}
				paramNames[i-1] = strings.TrimSuffix(params, ",")
				functionSigs[i-1] = key + "(" + strings.TrimSuffix(funcSig, ",") + ")"
				keccakHashes[i-1] = hex.EncodeToString(crypto.Keccak256([]byte(functionSigs[i-1]))[:4])
				funcParamNameMap[contractAdd+":"+keccakHashes[i-1]] = paramNames[i-1]
				functionSigs[i-1] = strings.TrimSuffix(funcSig, ",")
				funcSigMap[contractAdd+":"+keccakHashes[i-1]] = functionSigs[i-1]
				funcNameMap[contractAdd+":"+keccakHashes[i-1]] = key + "(" + strings.TrimSuffix(funcSig, ",") + ")"
				i++
			}
		}

	} else if abiMap[contractAdd] == "Unsupported" {
		decodeUnsupported := make([]ParamTableRow, 1)
		decodeUnsupported[0].Key = "decodeFailed"
		decodeUnsupported[0].Value = "Unsupported Datatype"
		return decodeUnsupported, ""
	}

	return Decode(payload, contractAdd)
}

func Decode(r string, contractAdd string) ([]ParamTableRow, string) {
	keccakHash := r[2:10]
	if funcSigMap[contractAdd+":"+keccakHash] == "" {
		abiMismatch := make([]ParamTableRow, 1)
		abiMismatch[0].Key = "decodeFailed"
		abiMismatch[0].Value = "ABI Mismatch"
		abiMap[contractAdd] = ""
		return abiMismatch, ""
	}
	encodedParams := r[10:]
	params := strings.Split(funcSigMap[contractAdd+":"+keccakHash], ",")
	paramTable := make([]ParamTableRow, len(params))
	if r == "" || len(r) < 1 {
		return paramTable, ""
	}
	paramNamesArr := strings.Split(funcParamNameMap[contractAdd+":"+keccakHash], ",")
	resultArray := contracthandler.FunctionProcessor{funcSigMap[contractAdd+":"+keccakHash], nil, encodedParams}.GetResults()
	for i := 0; i < len(params); i++ {
		paramTable[i].Key = paramNamesArr[i]
		paramTable[i].Value = fmt.Sprint(resultArray[i])
	}

	return paramTable, funcNameMap[contractAdd+":"+keccakHash]
}
