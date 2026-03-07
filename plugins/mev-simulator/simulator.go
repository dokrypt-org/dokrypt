package mevsimulator

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

type Strategy string

const (
	StrategyFrontrun Strategy = "frontrun"
	StrategySandwich Strategy = "sandwich"
	StrategyBackrun  Strategy = "backrun"
)

const (
	selSwapExactTokensForTokens = "38ed1739"
	selSwapTokensForExactTokens = "8803dbee"
	selSwapExactETHForTokens    = "7ff36ab5"
	selSwapExactTokensForETH    = "18cbafe5"
	selSwapETHForExactTokens    = "fb3bdb41"

	selTransfer = "a9059cbb"
	selApprove  = "095ea7b3"

	selExactInputSingle  = "414bf389"
	selExactOutputSingle = "db3e2198"

	selLiquidationCall = "00f714ce"

	selTransmit = "c9807539"
)

func selectorName(sel string) string {
	switch sel {
	case selSwapExactTokensForTokens:
		return "swapExactTokensForTokens"
	case selSwapTokensForExactTokens:
		return "swapTokensForExactTokens"
	case selSwapExactETHForTokens:
		return "swapExactETHForTokens"
	case selSwapExactTokensForETH:
		return "swapExactTokensForETH"
	case selSwapETHForExactTokens:
		return "swapETHForExactTokens"
	case selExactInputSingle:
		return "exactInputSingle"
	case selExactOutputSingle:
		return "exactOutputSingle"
	case selTransfer:
		return "transfer"
	case selApprove:
		return "approve"
	case selLiquidationCall:
		return "liquidationCall"
	case selTransmit:
		return "transmit"
	default:
		return "unknown"
	}
}

const (
	gasSimpleTransfer  uint64 = 21_000
	gasERC20Transfer   uint64 = 65_000
	gasUniV2Swap       uint64 = 150_000
	gasUniV3Swap       uint64 = 185_000
	gasSandwichPair    uint64 = 300_000 // front tx + back tx combined
	gasBackrunArb      uint64 = 250_000
	gasLiquidation     uint64 = 350_000
	gasOracleBackrun   uint64 = 200_000
)

var (
	defaultPoolReserve = ethToWei(10_000) // 10 000 ETH equivalent
)

type Transaction struct {
	Hash     string   `json:"hash"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Value    *big.Int `json:"value"`
	GasPrice *big.Int `json:"gas_price"`
	Data     []byte   `json:"data"`
}

type Opportunity struct {
	Strategy    Strategy `json:"strategy"`
	TargetTx    string   `json:"target_tx"`
	Profit      *big.Int `json:"profit"`
	GasCost     *big.Int `json:"gas_cost"`
	NetProfit   *big.Int `json:"net_profit"`
	Description string   `json:"description"`
}

type Simulator struct {
	mu            sync.Mutex
	strategies    []Strategy
	opportunities []Opportunity

	poolReserve *big.Int

	recentTxs []*Transaction
}

func New(strategies []Strategy) *Simulator {
	if len(strategies) == 0 {
		strategies = []Strategy{StrategyFrontrun, StrategySandwich, StrategyBackrun}
	}
	return &Simulator{
		strategies:  strategies,
		poolReserve: new(big.Int).Set(defaultPoolReserve),
	}
}

func (s *Simulator) SetPoolReserve(r *big.Int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.poolReserve = new(big.Int).Set(r)
}

func (s *Simulator) Analyze(ctx context.Context, tx *Transaction) []Opportunity {
	s.mu.Lock()
	defer s.mu.Unlock()

	var opps []Opportunity
	for _, strategy := range s.strategies {
		if ctx.Err() != nil {
			break
		}
		found := s.analyzeStrategy(tx, strategy)
		for i := range found {
			opps = append(opps, found[i])
			s.opportunities = append(s.opportunities, found[i])
		}
	}

	s.recentTxs = append(s.recentTxs, tx)
	if len(s.recentTxs) > 256 {
		s.recentTxs = s.recentTxs[len(s.recentTxs)-256:]
	}

	return opps
}

func (s *Simulator) Report() []Opportunity {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Opportunity, len(s.opportunities))
	copy(result, s.opportunities)
	return result
}

func (s *Simulator) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opportunities = nil
	s.recentTxs = nil
}

func (s *Simulator) analyzeStrategy(tx *Transaction, strategy Strategy) []Opportunity {
	switch strategy {
	case StrategyFrontrun:
		return s.analyzeFrontrun(tx)
	case StrategySandwich:
		return s.analyzeSandwich(tx)
	case StrategyBackrun:
		return s.analyzeBackrun(tx)
	default:
		return nil
	}
}

func (s *Simulator) analyzeFrontrun(tx *Transaction) []Opportunity {
	sel := extractSelector(tx.Data)
	if sel == "" {
		return nil
	}

	switch sel {
	case selSwapExactTokensForTokens,
		selSwapTokensForExactTokens,
		selSwapExactETHForTokens,
		selSwapExactTokensForETH,
		selSwapETHForExactTokens,
		selExactInputSingle,
		selExactOutputSingle:
		return s.frontrunSwap(tx, sel)

	case selTransfer:
		return s.frontrunLargeTransfer(tx, sel)

	default:
		return nil
	}
}

func (s *Simulator) frontrunSwap(tx *Transaction, sel string) []Opportunity {
	tradeSize := s.effectiveTradeSize(tx)
	if tradeSize.Sign() <= 0 {
		return nil
	}

	priceImpactBps := s.priceImpactBps(tradeSize)

	if priceImpactBps < 5 {
		return nil
	}

	frontSize := new(big.Int).Div(s.poolReserve, big.NewInt(10))
	if tradeSize.Cmp(frontSize) < 0 {
		frontSize = new(big.Int).Set(tradeSize)
	}

	gross := new(big.Int).Mul(frontSize, big.NewInt(int64(priceImpactBps)))
	gross.Div(gross, big.NewInt(10_000))

	gasUsed := gasForSelector(sel)
	gasCost := s.gasCostWei(tx, gasUsed)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	return []Opportunity{{
		Strategy:  StrategyFrontrun,
		TargetTx:  tx.Hash,
		Profit:    gross,
		GasCost:   gasCost,
		NetProfit: net,
		Description: fmt.Sprintf(
			"frontrun %s: victim trade %s wei, impact %d bps, est. gross %s wei",
			selectorName(sel), tradeSize.String(), priceImpactBps, gross.String(),
		),
	}}
}

func (s *Simulator) frontrunLargeTransfer(tx *Transaction, sel string) []Opportunity {
	amount := decodeTransferAmount(tx.Data)
	if amount == nil || amount.Sign() <= 0 {
		amount = tx.Value
	}
	if amount == nil || amount.Sign() <= 0 {
		return nil
	}

	tenEth := ethToWei(10)
	if amount.Cmp(tenEth) < 0 {
		return nil
	}

	priceImpactBps := s.priceImpactBps(amount)
	if priceImpactBps < 10 {
		return nil
	}

	gross := new(big.Int).Mul(amount, big.NewInt(int64(priceImpactBps)))
	gross.Div(gross, big.NewInt(10_000))
	gross.Mul(gross, big.NewInt(30))
	gross.Div(gross, big.NewInt(100))

	gasCost := s.gasCostWei(tx, gasERC20Transfer+gasUniV2Swap)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	return []Opportunity{{
		Strategy:  StrategyFrontrun,
		TargetTx:  tx.Hash,
		Profit:    gross,
		GasCost:   gasCost,
		NetProfit: net,
		Description: fmt.Sprintf(
			"frontrun large transfer: amount %s wei, impact %d bps",
			amount.String(), priceImpactBps,
		),
	}}
}

func (s *Simulator) analyzeSandwich(tx *Transaction) []Opportunity {
	sel := extractSelector(tx.Data)
	if !isSwapSelector(sel) {
		return nil
	}

	tradeSize := s.effectiveTradeSize(tx)
	if tradeSize.Sign() <= 0 {
		return nil
	}

	priceImpactBps := s.priceImpactBps(tradeSize)

	if priceImpactBps < 10 {
		return nil
	}

	attackSize := new(big.Int).Div(s.poolReserve, big.NewInt(20))
	if tradeSize.Cmp(attackSize) < 0 {
		attackSize = new(big.Int).Set(tradeSize)
	}

	gross := new(big.Int).Mul(attackSize, big.NewInt(int64(priceImpactBps)))
	gross.Div(gross, big.NewInt(10_000))

	gasCost := s.gasCostWei(tx, gasSandwichPair)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	desc := fmt.Sprintf(
		"sandwich %s: victim trade %s wei, attacker size %s wei, impact %d bps",
		selectorName(sel), tradeSize.String(), attackSize.String(), priceImpactBps,
	)

	if pairTx := s.findRecentSwapTo(tx.To, tx.Hash); pairTx != nil {
		desc += fmt.Sprintf("; correlated with recent tx %s to same pool", pairTx.Hash)
	}

	return []Opportunity{{
		Strategy:    StrategySandwich,
		TargetTx:    tx.Hash,
		Profit:      gross,
		GasCost:     gasCost,
		NetProfit:   net,
		Description: desc,
	}}
}

func (s *Simulator) analyzeBackrun(tx *Transaction) []Opportunity {
	sel := extractSelector(tx.Data)

	var opps []Opportunity

	switch {
	case isSwapSelector(sel):
		if opp := s.backrunArbitrage(tx, sel); opp != nil {
			opps = append(opps, *opp)
		}
	case sel == selLiquidationCall:
		if opp := s.backrunLiquidation(tx); opp != nil {
			opps = append(opps, *opp)
		}
	case sel == selTransmit:
		if opp := s.backrunOracleUpdate(tx); opp != nil {
			opps = append(opps, *opp)
		}
	}

	return opps
}

func (s *Simulator) backrunArbitrage(tx *Transaction, sel string) *Opportunity {
	tradeSize := s.effectiveTradeSize(tx)
	if tradeSize.Sign() <= 0 {
		return nil
	}

	impactBps := s.priceImpactBps(tradeSize)
	if impactBps < 5 {
		return nil
	}

	gross := new(big.Int).Mul(tradeSize, big.NewInt(int64(impactBps)))
	gross.Div(gross, big.NewInt(10_000))
	gross.Mul(gross, big.NewInt(50))
	gross.Div(gross, big.NewInt(100))

	gasCost := s.gasCostWei(tx, gasBackrunArb)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	return &Opportunity{
		Strategy:  StrategyBackrun,
		TargetTx:  tx.Hash,
		Profit:    gross,
		GasCost:   gasCost,
		NetProfit: net,
		Description: fmt.Sprintf(
			"backrun arb after %s: trade %s wei displaced price %d bps",
			selectorName(sel), tradeSize.String(), impactBps,
		),
	}
}

func (s *Simulator) backrunLiquidation(tx *Transaction) *Opportunity {
	value := tx.Value
	if value == nil || value.Sign() <= 0 {
		value = decodeFirstUint256(tx.Data)
	}
	if value == nil || value.Sign() <= 0 {
		return nil
	}

	oneEth := ethToWei(1)
	if value.Cmp(oneEth) < 0 {
		return nil
	}

	gross := new(big.Int).Mul(value, big.NewInt(5))
	gross.Div(gross, big.NewInt(100))

	gasCost := s.gasCostWei(tx, gasLiquidation)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	return &Opportunity{
		Strategy:  StrategyBackrun,
		TargetTx:  tx.Hash,
		Profit:    gross,
		GasCost:   gasCost,
		NetProfit: net,
		Description: fmt.Sprintf(
			"backrun liquidation: debt repaid %s wei, est. bonus %s wei",
			value.String(), gross.String(),
		),
	}
}

func (s *Simulator) backrunOracleUpdate(tx *Transaction) *Opportunity {
	arbSize := ethToWei(100)
	displacementBps := int64(20)

	gross := new(big.Int).Mul(arbSize, big.NewInt(displacementBps))
	gross.Div(gross, big.NewInt(10_000))

	gasCost := s.gasCostWei(tx, gasOracleBackrun)

	net := new(big.Int).Sub(gross, gasCost)
	if net.Sign() <= 0 {
		return nil
	}

	return &Opportunity{
		Strategy:  StrategyBackrun,
		TargetTx:  tx.Hash,
		Profit:    gross,
		GasCost:   gasCost,
		NetProfit: net,
		Description: fmt.Sprintf(
			"backrun oracle update: est. 20 bps displacement on 100 ETH arb trade, gross %s wei",
			gross.String(),
		),
	}
}

func extractSelector(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	return strings.ToLower(hex.EncodeToString(data[:4]))
}

func isSwapSelector(sel string) bool {
	switch sel {
	case selSwapExactTokensForTokens,
		selSwapTokensForExactTokens,
		selSwapExactETHForTokens,
		selSwapExactTokensForETH,
		selSwapETHForExactTokens,
		selExactInputSingle,
		selExactOutputSingle:
		return true
	default:
		return false
	}
}

func decodeTransferAmount(data []byte) *big.Int {
	if len(data) < 68 {
		return nil
	}
	return new(big.Int).SetBytes(data[36:68])
}

func decodeFirstUint256(data []byte) *big.Int {
	if len(data) < 36 {
		return nil
	}
	return new(big.Int).SetBytes(data[4:36])
}

func decodeSwapAmountIn(data []byte) *big.Int {
	return decodeFirstUint256(data)
}

func (s *Simulator) effectiveTradeSize(tx *Transaction) *big.Int {
	sel := extractSelector(tx.Data)

	switch sel {
	case selSwapExactETHForTokens, selSwapETHForExactTokens:
		if tx.Value != nil && tx.Value.Sign() > 0 {
			return tx.Value
		}
	}

	if amt := decodeSwapAmountIn(tx.Data); amt != nil && amt.Sign() > 0 {
		return amt
	}

	if tx.Value != nil && tx.Value.Sign() > 0 {
		return tx.Value
	}

	return new(big.Int)
}

func (s *Simulator) priceImpactBps(tradeSize *big.Int) int64 {
	if s.poolReserve.Sign() <= 0 {
		return 0
	}
	denom := new(big.Int).Add(s.poolReserve, tradeSize)
	if denom.Sign() <= 0 {
		return 0
	}
	num := new(big.Int).Mul(tradeSize, big.NewInt(10_000))
	bps := new(big.Int).Div(num, denom)
	return bps.Int64()
}

func (s *Simulator) gasCostWei(tx *Transaction, gasUnits uint64) *big.Int {
	gasPrice := tx.GasPrice
	if gasPrice == nil || gasPrice.Sign() <= 0 {
		gasPrice = big.NewInt(30_000_000_000) // 30 gwei default
	}
	return new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasUnits))
}

func gasForSelector(sel string) uint64 {
	switch sel {
	case selExactInputSingle, selExactOutputSingle:
		return gasUniV3Swap
	case selSwapExactTokensForTokens,
		selSwapTokensForExactTokens,
		selSwapExactETHForTokens,
		selSwapExactTokensForETH,
		selSwapETHForExactTokens:
		return gasUniV2Swap
	case selTransfer:
		return gasERC20Transfer
	default:
		return gasUniV2Swap
	}
}

func (s *Simulator) findRecentSwapTo(to string, excludeHash string) *Transaction {
	if to == "" {
		return nil
	}
	toNorm := strings.ToLower(to)
	for i := len(s.recentTxs) - 1; i >= 0; i-- {
		rtx := s.recentTxs[i]
		if rtx.Hash == excludeHash {
			continue
		}
		if strings.ToLower(rtx.To) == toNorm && isSwapSelector(extractSelector(rtx.Data)) {
			return rtx
		}
	}
	return nil
}

func ethToWei(eth int64) *big.Int {
	return new(big.Int).Mul(
		big.NewInt(eth),
		new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
	)
}
