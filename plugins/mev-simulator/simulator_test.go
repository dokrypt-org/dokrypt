package mevsimulator

import (
	"context"
	"encoding/hex"
	"math/big"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func selectorBytes(sel string) []byte {
	b, _ := hex.DecodeString(sel)
	return b
}

func swapCalldata(sel string, amountIn *big.Int) []byte {
	selB := selectorBytes(sel)
	amountBytes := amountIn.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(amountBytes):], amountBytes)
	return append(selB, padded...)
}

func transferCalldata(amount *big.Int) []byte {
	selB := selectorBytes(selTransfer)
	addr := make([]byte, 32)
	amountBytes := amount.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(amountBytes):], amountBytes)
	data := append(selB, addr...)
	data = append(data, padded...)
	return data
}

func liquidationCalldata(value *big.Int) []byte {
	selB := selectorBytes(selLiquidationCall)
	valBytes := value.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(valBytes):], valBytes)
	return append(selB, padded...)
}

func oracleCalldata() []byte {
	return selectorBytes(selTransmit)
}

func lowGasPrice() *big.Int {
	return big.NewInt(1_000_000_000) // 1 gwei
}

func TestNew_DefaultStrategies(t *testing.T) {
	sim := New(nil)
	require.NotNil(t, sim)
	assert.Len(t, sim.strategies, 3)
	assert.Contains(t, sim.strategies, StrategyFrontrun)
	assert.Contains(t, sim.strategies, StrategySandwich)
	assert.Contains(t, sim.strategies, StrategyBackrun)
}

func TestNew_CustomStrategies(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	assert.Len(t, sim.strategies, 1)
	assert.Equal(t, StrategyFrontrun, sim.strategies[0])
}

func TestAnalyze_EmptyData(t *testing.T) {
	sim := New(nil)
	tx := &Transaction{
		Hash:     "0xabc",
		From:     "0x1",
		To:       "0x2",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     nil,
	}
	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_ShortData(t *testing.T) {
	sim := New(nil)
	tx := &Transaction{
		Hash:     "0xabc",
		From:     "0x1",
		To:       "0x2",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     []byte{0x38, 0xed}, // only 2 bytes
	}
	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_FrontrunSwap_LargeUniV2(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100)) // 100 ETH pool

	tradeSize := ethToWei(10) // 10 ETH trade -> ~909 bps impact
	tx := &Transaction{
		Hash:     "0xswap1",
		From:     "0xtrader",
		To:       "0xrouter",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyFrontrun, opps[0].Strategy)
	assert.Equal(t, "0xswap1", opps[0].TargetTx)
	assert.True(t, opps[0].Profit.Sign() > 0)
	assert.True(t, opps[0].GasCost.Sign() > 0)
	assert.True(t, opps[0].NetProfit.Sign() > 0)
	assert.Contains(t, opps[0].Description, "frontrun")
	assert.Contains(t, opps[0].Description, "swapExactTokensForTokens")
}

func TestAnalyze_FrontrunSwap_AllV2Selectors(t *testing.T) {
	v2Selectors := []string{
		selSwapExactTokensForTokens,
		selSwapTokensForExactTokens,
		selSwapExactETHForTokens,
		selSwapExactTokensForETH,
		selSwapETHForExactTokens,
	}

	for _, sel := range v2Selectors {
		t.Run(sel, func(t *testing.T) {
			sim := New([]Strategy{StrategyFrontrun})
			sim.SetPoolReserve(ethToWei(100))

			tradeSize := ethToWei(10)
			tx := &Transaction{
				Hash:     "0x" + sel,
				From:     "0xtrader",
				To:       "0xrouter",
				Value:    tradeSize, // for ETH-payable selectors
				GasPrice: lowGasPrice(),
				Data:     swapCalldata(sel, tradeSize),
			}

			opps := sim.Analyze(context.Background(), tx)
			require.NotEmpty(t, opps, "expected frontrun opportunity for selector %s", sel)
			assert.Equal(t, StrategyFrontrun, opps[0].Strategy)
		})
	}
}

func TestAnalyze_FrontrunSwap_UniV3Selectors(t *testing.T) {
	v3Selectors := []string{selExactInputSingle, selExactOutputSingle}

	for _, sel := range v3Selectors {
		t.Run(sel, func(t *testing.T) {
			sim := New([]Strategy{StrategyFrontrun})
			sim.SetPoolReserve(ethToWei(100))

			tradeSize := ethToWei(10)
			tx := &Transaction{
				Hash:     "0x" + sel,
				From:     "0xtrader",
				To:       "0xrouter",
				Value:    big.NewInt(0),
				GasPrice: lowGasPrice(),
				Data:     swapCalldata(sel, tradeSize),
			}

			opps := sim.Analyze(context.Background(), tx)
			require.NotEmpty(t, opps, "expected frontrun opportunity for V3 selector %s", sel)
			assert.Equal(t, StrategyFrontrun, opps[0].Strategy)
		})
	}
}

func TestAnalyze_FrontrunSwap_TooSmallImpact(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(1_000_000))

	tradeSize := ethToWei(1) // ~0.1 bps on a 1M ETH pool
	tx := &Transaction{
		Hash:     "0xsmall",
		From:     "0xtrader",
		To:       "0xrouter",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_FrontrunSwap_ETHPayable_UsesValue(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100))

	selBytes := selectorBytes(selSwapExactETHForTokens)
	tx := &Transaction{
		Hash:     "0xethswap",
		From:     "0xtrader",
		To:       "0xrouter",
		Value:    ethToWei(10),
		GasPrice: lowGasPrice(),
		Data:     selBytes, // no amountIn encoded
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyFrontrun, opps[0].Strategy)
}

func TestAnalyze_FrontrunLargeTransfer(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100))

	amount := ethToWei(50)
	tx := &Transaction{
		Hash:     "0xtransfer",
		From:     "0xwhale",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     transferCalldata(amount),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyFrontrun, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "large transfer")
}

func TestAnalyze_FrontrunTransfer_TooSmall(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	amount := ethToWei(1)
	tx := &Transaction{
		Hash:     "0xsmall",
		From:     "0xuser",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     transferCalldata(amount),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_Sandwich_LargeSwap(t *testing.T) {
	sim := New([]Strategy{StrategySandwich})
	sim.SetPoolReserve(ethToWei(100))

	tradeSize := ethToWei(15) // ~1304 bps impact on 100 ETH pool
	tx := &Transaction{
		Hash:     "0xvictim",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategySandwich, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "sandwich")
}

func TestAnalyze_Sandwich_NonSwapSelector(t *testing.T) {
	sim := New([]Strategy{StrategySandwich})
	tx := &Transaction{
		Hash:     "0xtransfer",
		From:     "0x1",
		To:       "0x2",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     transferCalldata(ethToWei(100)),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_Sandwich_ImpactTooLow(t *testing.T) {
	sim := New([]Strategy{StrategySandwich})
	sim.SetPoolReserve(ethToWei(1_000_000))

	tradeSize := ethToWei(1) // tiny impact
	tx := &Transaction{
		Hash:     "0xsmall",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_Sandwich_CorrelatedRecentTx(t *testing.T) {
	sim := New([]Strategy{StrategySandwich})
	sim.SetPoolReserve(ethToWei(100))

	tradeSize := ethToWei(15)
	pool := "0xSamePool"

	tx1 := &Transaction{
		Hash:     "0xfirst",
		From:     "0xalice",
		To:       pool,
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}
	sim.Analyze(context.Background(), tx1)

	tx2 := &Transaction{
		Hash:     "0xsecond",
		From:     "0xbob",
		To:       pool,
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}
	opps := sim.Analyze(context.Background(), tx2)

	var found bool
	for _, o := range opps {
		if o.Strategy == StrategySandwich {
			assert.Contains(t, o.Description, "correlated with recent tx 0xfirst")
			found = true
		}
	}
	assert.True(t, found, "expected correlated sandwich detection")
}

func TestAnalyze_Backrun_Arbitrage(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})
	sim.SetPoolReserve(ethToWei(100))

	tradeSize := ethToWei(10)
	tx := &Transaction{
		Hash:     "0xswap",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyBackrun, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "backrun arb")
}

func TestAnalyze_Backrun_Liquidation(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})

	value := ethToWei(100)
	tx := &Transaction{
		Hash:     "0xliquidation",
		From:     "0xliquidator",
		To:       "0xlending",
		Value:    value,
		GasPrice: lowGasPrice(),
		Data:     liquidationCalldata(value),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyBackrun, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "backrun liquidation")
}

func TestAnalyze_Backrun_Liquidation_TooSmall(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})

	smallValue := big.NewInt(1_000_000) // way less than 1 ETH
	tx := &Transaction{
		Hash:     "0xsmall",
		From:     "0xliquidator",
		To:       "0xlending",
		Value:    smallValue,
		GasPrice: lowGasPrice(),
		Data:     liquidationCalldata(smallValue),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestAnalyze_Backrun_Liquidation_ValueFromCalldata(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})

	value := ethToWei(50)
	tx := &Transaction{
		Hash:     "0xliq2",
		From:     "0xliquidator",
		To:       "0xlending",
		Value:    big.NewInt(0), // no tx.Value, amount in calldata
		GasPrice: lowGasPrice(),
		Data:     liquidationCalldata(value),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyBackrun, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "backrun liquidation")
}

func TestAnalyze_Backrun_OracleUpdate(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})

	tx := &Transaction{
		Hash:     "0xoracle",
		From:     "0xnode",
		To:       "0xoracle_contract",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     oracleCalldata(),
	}

	opps := sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, opps)
	assert.Equal(t, StrategyBackrun, opps[0].Strategy)
	assert.Contains(t, opps[0].Description, "backrun oracle update")
}

func TestAnalyze_Backrun_OracleUpdate_HighGasPrice(t *testing.T) {
	sim := New([]Strategy{StrategyBackrun})

	tx := &Transaction{
		Hash:     "0xexpensive",
		From:     "0xnode",
		To:       "0xoracle_contract",
		Value:    big.NewInt(0),
		GasPrice: big.NewInt(10_000_000_000_000), // 10000 gwei
		Data:     oracleCalldata(),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps, "oracle backrun should be unprofitable at very high gas prices")
}

func TestAnalyze_AllStrategies(t *testing.T) {
	sim := New(nil) // all strategies
	sim.SetPoolReserve(ethToWei(100))

	tradeSize := ethToWei(15)
	tx := &Transaction{
		Hash:     "0xmulti",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(context.Background(), tx)
	strategies := make(map[Strategy]bool)
	for _, o := range opps {
		strategies[o.Strategy] = true
	}
	assert.True(t, strategies[StrategyFrontrun], "expected frontrun")
	assert.True(t, strategies[StrategyBackrun], "expected backrun")
}

func TestAnalyze_CancelledContext(t *testing.T) {
	sim := New(nil)
	sim.SetPoolReserve(ethToWei(100))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tradeSize := ethToWei(10)
	tx := &Transaction{
		Hash:     "0xcancelled",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
	}

	opps := sim.Analyze(ctx, tx)
	assert.True(t, len(opps) <= 1, "expected at most 1 opportunity with cancelled context, got %d", len(opps))
}

func TestReport_AccumulatesAcrossAnalyze(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100))

	tradeSize := ethToWei(10)
	for i := 0; i < 3; i++ {
		tx := &Transaction{
			Hash:     "0x" + string(rune('a'+i)),
			From:     "0xtrader",
			To:       "0xpool",
			Value:    big.NewInt(0),
			GasPrice: lowGasPrice(),
			Data:     swapCalldata(selSwapExactTokensForTokens, tradeSize),
		}
		sim.Analyze(context.Background(), tx)
	}

	report := sim.Report()
	assert.Len(t, report, 3)
}

func TestReset_ClearsOpportunitiesAndRecentTxs(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100))

	tx := &Transaction{
		Hash:     "0x1",
		From:     "0xtrader",
		To:       "0xpool",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     swapCalldata(selSwapExactTokensForTokens, ethToWei(10)),
	}
	sim.Analyze(context.Background(), tx)
	require.NotEmpty(t, sim.Report())

	sim.Reset()
	assert.Empty(t, sim.Report())
	assert.Nil(t, sim.recentTxs)
}

func TestAnalyze_SlidingWindowCap(t *testing.T) {
	sim := New([]Strategy{StrategyFrontrun})
	sim.SetPoolReserve(ethToWei(100))

	for i := 0; i < 300; i++ {
		tx := &Transaction{
			Hash:     "0x" + big.NewInt(int64(i)).Text(16),
			From:     "0xtrader",
			To:       "0xpool",
			Value:    big.NewInt(0),
			GasPrice: lowGasPrice(),
			Data:     swapCalldata(selSwapExactTokensForTokens, ethToWei(10)),
		}
		sim.Analyze(context.Background(), tx)
	}

	sim.mu.Lock()
	windowLen := len(sim.recentTxs)
	sim.mu.Unlock()
	assert.LessOrEqual(t, windowLen, 256)
}

func TestAnalyze_UnknownSelector(t *testing.T) {
	sim := New(nil)
	tx := &Transaction{
		Hash:     "0xunknown",
		From:     "0x1",
		To:       "0x2",
		Value:    big.NewInt(0),
		GasPrice: lowGasPrice(),
		Data:     selectorBytes("deadbeef"),
	}

	opps := sim.Analyze(context.Background(), tx)
	assert.Empty(t, opps)
}

func TestExtractSelector(t *testing.T) {
	assert.Equal(t, "", extractSelector(nil))
	assert.Equal(t, "", extractSelector([]byte{0x38}))
	assert.Equal(t, "38ed1739", extractSelector(selectorBytes("38ed1739")))
}

func TestIsSwapSelector(t *testing.T) {
	assert.True(t, isSwapSelector(selSwapExactTokensForTokens))
	assert.True(t, isSwapSelector(selExactInputSingle))
	assert.False(t, isSwapSelector(selTransfer))
	assert.False(t, isSwapSelector(selApprove))
	assert.False(t, isSwapSelector("deadbeef"))
	assert.False(t, isSwapSelector(""))
}

func TestSelectorName(t *testing.T) {
	assert.Equal(t, "swapExactTokensForTokens", selectorName(selSwapExactTokensForTokens))
	assert.Equal(t, "exactInputSingle", selectorName(selExactInputSingle))
	assert.Equal(t, "transfer", selectorName(selTransfer))
	assert.Equal(t, "approve", selectorName(selApprove))
	assert.Equal(t, "liquidationCall", selectorName(selLiquidationCall))
	assert.Equal(t, "transmit", selectorName(selTransmit))
	assert.Equal(t, "unknown", selectorName("deadbeef"))
}

func TestDecodeTransferAmount(t *testing.T) {
	amount := ethToWei(42)
	data := transferCalldata(amount)
	decoded := decodeTransferAmount(data)
	require.NotNil(t, decoded)
	assert.Equal(t, 0, amount.Cmp(decoded))

	assert.Nil(t, decodeTransferAmount([]byte{0x01, 0x02}))
}

func TestDecodeFirstUint256(t *testing.T) {
	val := big.NewInt(12345)
	data := swapCalldata(selSwapExactTokensForTokens, val)
	decoded := decodeFirstUint256(data)
	require.NotNil(t, decoded)
	assert.Equal(t, 0, val.Cmp(decoded))

	assert.Nil(t, decodeFirstUint256([]byte{0x01}))
}

func TestEthToWei(t *testing.T) {
	oneEth := ethToWei(1)
	expected := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	assert.Equal(t, 0, oneEth.Cmp(expected))

	tenEth := ethToWei(10)
	assert.Equal(t, 0, tenEth.Cmp(new(big.Int).Mul(expected, big.NewInt(10))))
}

func TestGasForSelector(t *testing.T) {
	assert.Equal(t, uint64(150_000), gasForSelector(selSwapExactTokensForTokens))
	assert.Equal(t, uint64(185_000), gasForSelector(selExactInputSingle))
	assert.Equal(t, uint64(65_000), gasForSelector(selTransfer))
	assert.Equal(t, uint64(150_000), gasForSelector("deadbeef")) // default
}

func TestSetPoolReserve(t *testing.T) {
	sim := New(nil)
	newReserve := ethToWei(500)
	sim.SetPoolReserve(newReserve)

	sim.mu.Lock()
	assert.Equal(t, 0, sim.poolReserve.Cmp(newReserve))
	sim.mu.Unlock()
}

func TestGasCostWei_DefaultGasPrice(t *testing.T) {
	sim := New(nil)
	tx := &Transaction{
		Hash:     "0x1",
		GasPrice: nil, // no gas price
	}

	sim.mu.Lock()
	cost := sim.gasCostWei(tx, 21_000)
	sim.mu.Unlock()

	expected := new(big.Int).Mul(big.NewInt(21_000), big.NewInt(30_000_000_000))
	assert.Equal(t, 0, cost.Cmp(expected))
}

func TestConcurrentAnalyze(t *testing.T) {
	sim := New(nil)
	sim.SetPoolReserve(ethToWei(100))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tx := &Transaction{
				Hash:     "0x" + big.NewInt(int64(id)).Text(16),
				From:     "0xtrader",
				To:       "0xpool",
				Value:    big.NewInt(0),
				GasPrice: lowGasPrice(),
				Data:     swapCalldata(selSwapExactTokensForTokens, ethToWei(10)),
			}
			sim.Analyze(context.Background(), tx)
		}(i)
	}
	wg.Wait()

	report := sim.Report()
	assert.GreaterOrEqual(t, len(report), 50)
}
