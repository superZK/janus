package proxy

import (
	"strconv"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/hellofresh/janus/pkg/proxy/balancer"
	"github.com/hellofresh/janus/pkg/router"
)

// Definition defines proxy rules for a route
type Definition struct {
	PreserveHost       bool               `bson:"preserve_host" json:"preserve_host" mapstructure:"preserve_host"`
	ListenPath         string             `bson:"listen_path" json:"listen_path" mapstructure:"listen_path" valid:"required~proxy.listen_path is required,urlpath"`
	Upstreams          *Upstreams         `bson:"upstreams" json:"upstreams" mapstructure:"upstreams"`
	InsecureSkipVerify bool               `bson:"insecure_skip_verify" json:"insecure_skip_verify" mapstructure:"insecure_skip_verify"`
	StripPath          bool               `bson:"strip_path" json:"strip_path" mapstructure:"strip_path"`
	AppendPath         bool               `bson:"append_path" json:"append_path" mapstructure:"append_path"`
	Methods            []string           `bson:"methods" json:"methods"`
	Hosts              []string           `bson:"hosts" json:"hosts"`
	ForwardingTimeouts ForwardingTimeouts `bson:"forwarding_timeouts" json:"forwarding_timeouts" mapstructure:"forwarding_timeouts"`
}

// RouterDefinition represents an API that you want to proxy with internal router routines
type RouterDefinition struct {
	*Definition
	middleware []router.Constructor
}

// Upstreams represents a collection of targets where the requests will go to
type Upstreams struct {
	Balancing string  `bson:"balancing" json:"balancing"`
	Targets   Targets `bson:"targets" json:"targets"`
}

// Target is an ip address/hostname with a port that identifies an instance of a backend service
type Target struct {
	Target string `bson:"target" json:"target" valid:"url,required"`
	Weight int    `bson:"weight" json:"weight"`
}

// Targets is a set of target
type Targets []*Target

// Duration is the time.Duration that can be unmarshalled from JSON
type Duration time.Duration

// MarshalJSON implements marshalling from JSON
func (d *Duration) MarshalJSON() ([]byte, error) {
	s := (*time.Duration)(d).String()
	s = strconv.Quote(s)

	return []byte(s), nil
}

// UnmarshalJSON implements unmarshalling from JSON
func (d *Duration) UnmarshalJSON(data []byte) (err error) {
	s := string(data)
	if s == "null" {
		return
	}

	s, err = strconv.Unquote(s)
	if err != nil {
		return
	}

	t, err := time.ParseDuration(s)
	if err != nil {
		return
	}

	*d = Duration(t)
	return
}

// ForwardingTimeouts contains timeout configurations for forwarding requests to the backend servers.
type ForwardingTimeouts struct {
	DialTimeout           Duration `bson:"dial_timeout" json:"dial_timeout"`
	ResponseHeaderTimeout Duration `bson:"response_header_timeout" json:"response_header_timeout"`
}

// NewDefinition creates a new Proxy Definition with default values
func NewDefinition() *Definition {
	return &Definition{
		Methods: []string{"GET"},
		Hosts:   make([]string, 0),
		Upstreams: &Upstreams{
			Targets: make([]*Target, 0),
		},
	}
}

// NewRouterDefinition creates a new Proxy RouterDefinition from Proxy Definition
func NewRouterDefinition(def *Definition) *RouterDefinition {
	return &RouterDefinition{Definition: def}
}

// Middleware returns s.middleware (useful for tests).
func (d *RouterDefinition) Middleware() []router.Constructor {
	return d.middleware
}

// AddMiddleware adds a middleware to a site's middleware stack.
func (d *RouterDefinition) AddMiddleware(m router.Constructor) {
	d.middleware = append(d.middleware, m)
}

// Validate validates proxy data
func (d *Definition) Validate() (bool, error) {
	return govalidator.ValidateStruct(d)
}

// IsBalancerDefined checks if load balancer is defined
func (d *Definition) IsBalancerDefined() bool {
	return d.Upstreams != nil && d.Upstreams.Targets != nil && len(d.Upstreams.Targets) > 0
}

// ToBalancerTargets returns the balancer expected type
func (t Targets) ToBalancerTargets() []*balancer.Target {
	var balancerTargets []*balancer.Target
	for _, t := range t {
		balancerTargets = append(balancerTargets, &balancer.Target{
			Target: t.Target,
			Weight: t.Weight,
		})
	}

	return balancerTargets
}
func init() {
	// initializes custom validators
	govalidator.CustomTypeTagMap.Set("urlpath", func(i interface{}, o interface{}) bool {
		s, ok := i.(string)
		if !ok {
			return false
		}

		return strings.Index(s, "/") == 0
	})
}
