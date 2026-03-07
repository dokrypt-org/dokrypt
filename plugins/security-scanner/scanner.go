package securityscanner

import (
	"context"
	"sync"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Finding struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Severity    Severity `json:"severity"`
	Contract    string   `json:"contract"`
	Location    string   `json:"location,omitempty"`
	Suggestion  string   `json:"suggestion,omitempty"`
}

type ScanResult struct {
	Contract string    `json:"contract"`
	Address  string    `json:"address"`
	Findings []Finding `json:"findings"`
}

type Scanner struct {
	mu      sync.Mutex
	results []ScanResult
	rules   []Rule
}

type Rule struct {
	ID       string
	Title    string
	Severity Severity
	Check    func(bytecode []byte) *Finding
}

func New() *Scanner {
	return &Scanner{
		rules: defaultRules(),
	}
}

func (s *Scanner) Scan(ctx context.Context, contractName string, address string, bytecode []byte) *ScanResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := &ScanResult{
		Contract: contractName,
		Address:  address,
	}

	for _, rule := range s.rules {
		if finding := rule.Check(bytecode); finding != nil {
			finding.Contract = contractName
			result.Findings = append(result.Findings, *finding)
		}
	}

	s.results = append(s.results, *result)
	return result
}

func (s *Scanner) Results() []ScanResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ScanResult, len(s.results))
	copy(result, s.results)
	return result
}

func (s *Scanner) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = nil
}

func scanOpcodes(bytecode []byte, target byte) bool {
	for i := 0; i < len(bytecode); {
		op := bytecode[i]
		if op == target {
			return true
		}
		if op >= 0x60 && op <= 0x7f {
			n := int(op-0x60) + 1
			i += 1 + n
		} else {
			i++
		}
	}
	return false
}

func defaultRules() []Rule {
	return []Rule{
		{
			ID: "SELFDESTRUCT", Title: "SELFDESTRUCT usage detected", Severity: SeverityHigh,
			Check: func(bytecode []byte) *Finding {
				if scanOpcodes(bytecode, 0xFF) {
					return &Finding{
						Title:       "SELFDESTRUCT usage detected",
						Description: "Contract contains SELFDESTRUCT opcode which can destroy the contract",
						Severity:    SeverityHigh,
						Suggestion:  "Consider removing SELFDESTRUCT or adding access controls",
					}
				}
				return nil
			},
		},
		{
			ID: "DELEGATECALL", Title: "DELEGATECALL usage detected", Severity: SeverityMedium,
			Check: func(bytecode []byte) *Finding {
				if scanOpcodes(bytecode, 0xF4) {
					return &Finding{
						Title:       "DELEGATECALL usage detected",
						Description: "Contract uses DELEGATECALL which can execute arbitrary code in contract context",
						Severity:    SeverityMedium,
						Suggestion:  "Ensure DELEGATECALL targets are trusted and immutable",
					}
				}
				return nil
			},
		},
		{
			ID: "TX_ORIGIN", Title: "tx.origin usage detected", Severity: SeverityMedium,
			Check: func(bytecode []byte) *Finding {
				if scanOpcodes(bytecode, 0x32) {
					return &Finding{
						Title:       "tx.origin usage detected",
						Description: "Contract may use tx.origin for authorization which is vulnerable to phishing",
						Severity:    SeverityMedium,
						Suggestion:  "Use msg.sender instead of tx.origin for authorization",
					}
				}
				return nil
			},
		},
		{
			ID: "REENTRANCY_RISK", Title: "Potential reentrancy risk", Severity: SeverityHigh,
			Check: func(bytecode []byte) *Finding {
				callSeen := false
				for i := 0; i < len(bytecode); {
					op := bytecode[i]
					if op >= 0x60 && op <= 0x7f {
						i += 1 + int(op-0x60) + 1
						continue
					}
					if op == 0xF1 { // CALL
						callSeen = true
					}
					if callSeen && op == 0x55 { // SSTORE after CALL
						return &Finding{
							Title:       "Potential reentrancy risk",
							Description: "Contract performs SSTORE after CALL — state update after external call may be vulnerable to reentrancy",
							Severity:    SeverityHigh,
							Suggestion:  "Use checks-effects-interactions pattern or a reentrancy guard",
						}
					}
					i++
				}
				return nil
			},
		},
		{
			ID: "UNCHECKED_CALL", Title: "Unchecked low-level call", Severity: SeverityMedium,
			Check: func(bytecode []byte) *Finding {
				for i := 0; i < len(bytecode); {
					op := bytecode[i]
					if op >= 0x60 && op <= 0x7f {
						i += 1 + int(op-0x60) + 1
						continue
					}
					if op == 0xF1 && i+1 < len(bytecode) && bytecode[i+1] == 0x50 {
						return &Finding{
							Title:       "Unchecked low-level call",
							Description: "Return value of CALL is immediately POPped without checking for success",
							Severity:    SeverityMedium,
							Suggestion:  "Check the return value of low-level calls and handle failures",
						}
					}
					i++
				}
				return nil
			},
		},
	}
}
