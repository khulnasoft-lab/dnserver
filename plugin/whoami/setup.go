package whoami

import (
	"github.com/coredns/caddy"
	"github.com/khulnasoft-lab/dnserver/core/dnsserver"
	"github.com/khulnasoft-lab/dnserver/plugin"
)

func init() { plugin.Register("whoami", setup) }

func setup(c *caddy.Controller) error {
	c.Next() // 'whoami'
	if c.NextArg() {
		return plugin.Error("whoami", c.ArgErr())
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Whoami{}
	})

	return nil
}
