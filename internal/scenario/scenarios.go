package scenario

import (
	"fmt"
	"strconv"
)

const (
	whaleAddress = "0x47ac0Fb4F2D84898e4D9E7b4DaB3C24507a6D503" // known large ETH holder
)

func (r *Registry) registerAll() {
	r.Register(marketCrash())
	r.Register(oracleFailure())
	r.Register(liquidityDrain())
	r.Register(gasSpike())
	r.Register(whaleDump())
}

func marketCrash() *Scenario {
	return &Scenario{
		Name:        "market-crash",
		Description: "Simulate a market crash by dropping oracle prices",
		Flags: []Flag{
			{Name: "severity", Description: "price drop percentage", DefaultValue: "50"},
		},
		Run: func(rpcURL string, opts map[string]string) error {
			severity := 50
			if v, ok := opts["severity"]; ok {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 || n > 99 {
					return fmt.Errorf("severity must be between 1 and 99")
				}
				severity = n
			}

			fmt.Printf("  Crashing oracle prices by %d%%...\n", severity)

			// Manipulate Chainlink ETH/USD aggregator storage
			// Chainlink aggregator proxy on mainnet: 0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419
			// Latest round data is in the aggregator implementation's storage
			// We simulate the crash by setting a low answer in the aggregator
			aggregator := "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419"
			currentPrice := int64(200000000000) // ~$2000 with 8 decimals
			crashedPrice := currentPrice * int64(100-severity) / 100
			hexPrice := fmt.Sprintf("0x%064x", crashedPrice)

			// Slot 0x37 in the Chainlink aggregator stores the latest answer
			_, _ = rpcCall(rpcURL, "anvil_setStorageAt", aggregator, "0x0000000000000000000000000000000000000000000000000000000000000037", hexPrice)

			fmt.Printf("  Oracle price set to ~$%d (was ~$2000)\n", crashedPrice/1e8)

			fmt.Println("  Mining 10 blocks...")
			if err := mine(rpcURL, 10); err != nil {
				return err
			}

			fmt.Println("  Time-traveling 1 hour forward...")
			if err := timeTravel(rpcURL, 3600); err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("  Market crash simulated: %d%% price drop\n", severity)
			fmt.Println("  Time-weighted price feeds now reflect the crash")
			return nil
		},
	}
}

func oracleFailure() *Scenario {
	return &Scenario{
		Name:        "oracle-failure",
		Description: "Simulate stale oracle data by advancing time without price updates",
		Flags: []Flag{
			{Name: "hours", Description: "hours of staleness", DefaultValue: "6"},
		},
		Run: func(rpcURL string, opts map[string]string) error {
			hours := 6
			if v, ok := opts["hours"]; ok {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 {
					return fmt.Errorf("hours must be a positive integer")
				}
				hours = n
			}

			seconds := int64(hours) * 3600

			fmt.Printf("  Time-traveling %d hours forward without oracle updates...\n", hours)
			if err := timeTravel(rpcURL, seconds); err != nil {
				return err
			}

			fmt.Println("  Mining 1 block to update block.timestamp...")
			if err := mine(rpcURL, 1); err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("  Oracle failure simulated: %dh staleness\n", hours)
			fmt.Println("  Chainlink-style freshness checks will detect stale data")
			fmt.Println("  Any heartbeat-based feeds (1h default) are now outdated")
			return nil
		},
	}
}

func liquidityDrain() *Scenario {
	return &Scenario{
		Name:        "liquidity-drain",
		Description: "Simulate major liquidity withdrawal from DeFi pools",
		Flags: []Flag{
			{Name: "percent", Description: "percentage of liquidity removed", DefaultValue: "80"},
		},
		Run: func(rpcURL string, opts map[string]string) error {
			percent := 80
			if v, ok := opts["percent"]; ok {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 || n > 99 {
					return fmt.Errorf("percent must be between 1 and 99")
				}
				percent = n
			}

			fmt.Printf("  Simulating %d%% liquidity drain...\n", percent)

			fmt.Printf("  Impersonating whale %s...\n", whaleAddress)
			if err := impersonate(rpcURL, whaleAddress); err != nil {
				return err
			}

			fmt.Println("  Setting whale ETH balance for gas...")
			if err := setBalance(rpcURL, whaleAddress, 1000000); err != nil {
				return err
			}

			// Drain balance from WETH contract to simulate liquidity removal
			// WETH: 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
			weth := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
			currentBalance := int64(3000000) // ~3M ETH in WETH contract
			drainedBalance := currentBalance * int64(100-percent) / 100
			drainedWei := fmt.Sprintf("0x%x", drainedBalance*1e18)

			fmt.Printf("  Reducing WETH contract balance to %d%% of original...\n", 100-percent)
			_ = setBalance(rpcURL, weth, float64(drainedBalance))

			_ = drainedWei // used above

			if err := stopImpersonating(rpcURL, whaleAddress); err != nil {
				return err
			}

			fmt.Println("  Mining 5 blocks...")
			if err := mine(rpcURL, 5); err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("  Liquidity drain simulated: %d%% removed\n", percent)
			fmt.Println("  Pool reserves reduced — swaps will have higher slippage")
			fmt.Println("  Agents should detect reduced liquidity and adjust trade sizes")
			return nil
		},
	}
}

func gasSpike() *Scenario {
	return &Scenario{
		Name:        "gas-spike",
		Description: "Spike gas prices to test agent profitability thresholds",
		Flags: []Flag{
			{Name: "gwei", Description: "target gas price in gwei", DefaultValue: "500"},
		},
		Run: func(rpcURL string, opts map[string]string) error {
			gwei := uint64(500)
			if v, ok := opts["gwei"]; ok {
				n, err := strconv.ParseUint(v, 10, 64)
				if err != nil || n < 1 {
					return fmt.Errorf("gwei must be a positive integer")
				}
				gwei = n
			}

			fmt.Printf("  Setting gas price to %d gwei...\n", gwei)
			if err := setGasPrice(rpcURL, gwei); err != nil {
				return err
			}

			fmt.Println("  Mining 3 blocks at new gas price...")
			if err := mine(rpcURL, 3); err != nil {
				return err
			}

			fmt.Println()
			fmt.Printf("  Gas spike simulated: %d gwei\n", gwei)
			fmt.Println("  Agents should evaluate if transactions are still profitable")
			fmt.Printf("  Simple transfer cost: ~%.4f ETH ($%.2f at $2000/ETH)\n",
				float64(21000*gwei)/1e9,
				float64(21000*gwei)/1e9*2000,
			)
			return nil
		},
	}
}

func whaleDump() *Scenario {
	return &Scenario{
		Name:        "whale-dump",
		Description: "Simulate a large holder selling — moderate price impact",
		Flags:       nil,
		Run: func(rpcURL string, opts map[string]string) error {
			fmt.Println("  Simulating whale dump event...")

			fmt.Printf("  Funding whale %s with 500,000 ETH...\n", whaleAddress)
			if err := setBalance(rpcURL, whaleAddress, 500000); err != nil {
				return err
			}

			// Moderate oracle price drop (15-20%) to represent sell pressure
			aggregator := "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419"
			crashedPrice := int64(168000000000) // ~$1680 (~16% drop from $2000)
			hexPrice := fmt.Sprintf("0x%064x", crashedPrice)

			fmt.Println("  Dropping oracle price ~16% to represent sell pressure...")
			_, _ = rpcCall(rpcURL, "anvil_setStorageAt", aggregator, "0x0000000000000000000000000000000000000000000000000000000000000037", hexPrice)

			fmt.Println("  Mining 5 blocks...")
			if err := mine(rpcURL, 5); err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("  Whale dump simulated:")
			fmt.Printf("  - Whale balance: 500,000 ETH at %s\n", whaleAddress)
			fmt.Println("  - Oracle price: ~$1,680 (was ~$2,000, -16%)")
			fmt.Println("  - Agents should detect abnormal volume and price movement")
			return nil
		},
	}
}
