package tier

// Plan represents a subscription tier.
type Plan string

const (
	Free     Plan = "free"
	Pro      Plan = "pro"
	Business Plan = "business"
)

// Config holds the limits and features for a tier.
type Config struct {
	Name             Plan
	PriceCents       int32
	MaxAccounts      int
	PlanGenerations  int // per month, -1 = unlimited
	RateLimitPerHour int // only for unlimited tiers
	AutoPosting      bool
	Export           bool
	Sharing          bool
}

var configs = map[Plan]Config{
	Free: {
		Name:            Free,
		PriceCents:      0,
		MaxAccounts:     1,
		PlanGenerations: 3,
		AutoPosting:     false,
		Export:          false,
		Sharing:         false,
	},
	Pro: {
		Name:            Pro,
		PriceCents:      999, // $9.99
		MaxAccounts:     1,
		PlanGenerations: 15,
		AutoPosting:     true,
		Export:          true,
		Sharing:         false,
	},
	Business: {
		Name:             Business,
		PriceCents:       2899, // $28.99
		MaxAccounts:      5,
		PlanGenerations:  -1, // unlimited
		RateLimitPerHour: 30,
		AutoPosting:      true,
		Export:           true,
		Sharing:          true,
	},
}

// Get returns the config for the given plan. Defaults to Free if unknown.
func Get(plan string) Config {
	if c, ok := configs[Plan(plan)]; ok {
		return c
	}
	return configs[Free]
}

// IsValid reports whether the plan name is a known tier.
func IsValid(plan string) bool {
	_, ok := configs[Plan(plan)]
	return ok
}
