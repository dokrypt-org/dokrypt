package types

type Account struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key,omitempty"`
	Label      string `json:"label,omitempty"`
	Balance    string `json:"balance,omitempty"`
}

type ChainStatus struct {
	Name    string `json:"name"`
	Engine  string `json:"engine"`
	ChainID uint64 `json:"chain_id"`
	RPCURL  string `json:"rpc_url"`
	WSURL   string `json:"ws_url"`
	Running bool   `json:"running"`
	Healthy bool   `json:"healthy"`
}

type TransactionReceipt struct {
	Hash        string `json:"hash"`
	Status      uint64 `json:"status"`
	GasUsed     uint64 `json:"gas_used"`
	BlockNumber uint64 `json:"block_number"`
}
