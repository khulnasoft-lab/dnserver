package rewrite

import (
	"context"
	"reflect"
	"testing"

	"github.com/khulnasoft-lab/dnserver/plugin"
	"github.com/khulnasoft-lab/dnserver/plugin/pkg/dnstest"
	"github.com/khulnasoft-lab/dnserver/plugin/test"
	"github.com/khulnasoft-lab/dnserver/request"

	"github.com/miekg/dns"
)

type MockedUpstream struct{}

func (u *MockedUpstream) Lookup(ctx context.Context, state request.Request, name string, typ uint16) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetReply(state.Req)
	m.Authoritative = true
	switch state.Req.Question[0].Name {
	case "xyz.example.com.":
		m.Answer = []dns.RR{
			test.A("xyz.example.com. 3600 IN A 3.4.5.6"),
		}
		return m, nil
	case "bard.google.com.cdn.cloudflare.net.":
		m.Answer = []dns.RR{
			test.A("bard.google.com.cdn.cloudflare.net.  1800  IN  A  9.7.2.1"),
		}
		return m, nil
	case "www.hosting.xyz.":
		m.Answer = []dns.RR{
			test.A("www.hosting.xyz.  500  IN  A  20.30.40.50"),
		}
		return m, nil
	case "abcd.zzzz.www.pqrst.":
		m.Answer = []dns.RR{
			test.A("abcd.zzzz.www.pqrst.   120  IN  A   101.20.5.1"),
			test.A("abcd.zzzz.www.pqrst.   120  IN  A   101.20.5.2"),
		}
		return m, nil
	case "orders.webapp.eu.org.":
		m.Answer = []dns.RR{
			test.A("orders.webapp.eu.org.   120  IN  A   20.0.0.9"),
		}
		return m, nil
	}
	return &dns.Msg{}, nil
}

func TestCNameTargetRewrite(t *testing.T) {
	rules := []Rule{}
	ruleset := []struct {
		args         []string
		expectedType reflect.Type
	}{
		{[]string{"continue", "cname", "exact", "def.example.com.", "xyz.example.com."}, reflect.TypeOf(&cnameTargetRule{})},
		{[]string{"continue", "cname", "prefix", "chat.openai.com", "bard.google.com"}, reflect.TypeOf(&cnameTargetRule{})},
		{[]string{"continue", "cname", "suffix", "uvw.", "xyz."}, reflect.TypeOf(&cnameTargetRule{})},
		{[]string{"continue", "cname", "substring", "efgh", "zzzz.www"}, reflect.TypeOf(&cnameTargetRule{})},
		{[]string{"continue", "cname", "regex", `(.*)\.web\.(.*)\.site\.`, `{1}.webapp.{2}.org.`}, reflect.TypeOf(&cnameTargetRule{})},
	}
	for i, r := range ruleset {
		rule, err := newRule(r.args...)
		if err != nil {
			t.Fatalf("Rule %d: FAIL, %s: %s", i, r.args, err)
		}
		if reflect.TypeOf(rule) != r.expectedType {
			t.Fatalf("Rule %d: FAIL, %s: rule type mismatch, expected %q, but got %q", i, r.args, r.expectedType, rule)
		}
		cnameTargetRule := rule.(*cnameTargetRule)
		cnameTargetRule.Upstream = &MockedUpstream{}
		rules = append(rules, rule)
	}
	doTestCNameTargetTests(rules, t)
}

func doTestCNameTargetTests(rules []Rule, t *testing.T) {
	tests := []struct {
		from           string
		fromType       uint16
		answer         []dns.RR
		expectedAnswer []dns.RR
	}{
		{"abc.example.com", dns.TypeA,
			[]dns.RR{
				test.CNAME("abc.example.com.  5   IN  CNAME  def.example.com."),
				test.A("def.example.com.   5  IN  A   1.2.3.4"),
			},
			[]dns.RR{
				test.CNAME("abc.example.com.  5   IN  CNAME  xyz.example.com."),
				test.A("xyz.example.com.  3600  IN  A  3.4.5.6"),
			},
		},
		{"chat.openai.com", dns.TypeA,
			[]dns.RR{
				test.CNAME("chat.openai.com.  20   IN  CNAME  chat.openai.com.cdn.cloudflare.net."),
				test.A("chat.openai.com.cdn.cloudflare.net.   30  IN  A   23.2.1.2"),
				test.A("chat.openai.com.cdn.cloudflare.net.   30  IN  A   24.6.0.8"),
			},
			[]dns.RR{
				test.CNAME("chat.openai.com.  20   IN  CNAME  bard.google.com.cdn.cloudflare.net."),
				test.A("bard.google.com.cdn.cloudflare.net.  1800  IN  A  9.7.2.1"),
			},
		},
		{"dnsserver.khulnasoft.com", dns.TypeA,
			[]dns.RR{
				test.CNAME("dnsserver.khulnasoft.com.  100   IN  CNAME  www.hosting.uvw."),
				test.A("www.hosting.uvw.   200  IN  A   7.2.3.4"),
			},
			[]dns.RR{
				test.CNAME("dnsserver.khulnasoft.com.  100   IN  CNAME  www.hosting.xyz."),
				test.A("www.hosting.xyz.  500  IN  A  20.30.40.50"),
			},
		},
		{"core.dns.rocks", dns.TypeA,
			[]dns.RR{
				test.CNAME("core.dns.rocks.  200   IN  CNAME  abcd.efgh.pqrst."),
				test.A("abcd.efgh.pqrst.   100  IN  A   200.30.45.67"),
			},
			[]dns.RR{
				test.CNAME("core.dns.rocks.  200   IN  CNAME  abcd.zzzz.www.pqrst."),
				test.A("abcd.zzzz.www.pqrst.   120  IN  A   101.20.5.1"),
				test.A("abcd.zzzz.www.pqrst.   120  IN  A   101.20.5.2"),
			},
		},
		{"order.service.eu", dns.TypeA,
			[]dns.RR{
				test.CNAME("order.service.eu.  200   IN  CNAME  orders.web.eu.site."),
				test.A("orders.web.eu.site.   50  IN  A   10.10.15.1"),
			},
			[]dns.RR{
				test.CNAME("order.service.eu.  200   IN  CNAME  orders.webapp.eu.org."),
				test.A("orders.webapp.eu.org.   120  IN  A   20.0.0.9"),
			},
		},
	}
	ctx := context.TODO()
	for i, tc := range tests {
		m := new(dns.Msg)
		m.SetQuestion(tc.from, tc.fromType)
		m.Question[0].Qclass = dns.ClassINET
		m.Answer = tc.answer
		rw := Rewrite{
			Next:  plugin.HandlerFunc(msgPrinter),
			Rules: rules,
		}
		rec := dnstest.NewRecorder(&test.ResponseWriter{})
		rw.ServeDNS(ctx, rec, m)
		resp := rec.Msg
		if len(resp.Answer) == 0 {
			t.Errorf("Test %d: FAIL %s (%d) Expected valid response but received %q", i, tc.from, tc.fromType, resp)
			continue
		}
		if !reflect.DeepEqual(resp.Answer, tc.expectedAnswer) {
			t.Errorf("Test %d: FAIL %s (%d) Actual are expected answer does not match, actual: %v, expected: %v",
				i, tc.from, tc.fromType, resp.Answer, tc.expectedAnswer)
			continue
		}
	}
}
