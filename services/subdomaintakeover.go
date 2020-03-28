package services

import (
	"bytes"
	"github.com/miekg/dns"
	"net/url"
)

//https://github.com/EdOverflow/can-i-take-over-xyz
var fingerprints = map[string]string{
	"AWS/S3":           "The specified bucket does not exist",
	"Bitbucket":        "Repository not found",
	"Campaign Monitor": "'Trying to access your account?'",
	"Cargo Collective": "404 Not Found",
	"Cloudfront":       "ViewerCertificateException",
	"Desk":             "Please try again or try Desk.com free for 14 days.",
	"Digital Ocean":    "Domain uses DO name serves with no records in DO.",
	"Fastly":           "Fastly error: unknown domain:",
	"Feedpress":        "The feed has not been found.",
	"Fly.io":           "404 Not Found",
	"Ghost":            "The thing you were looking for is no longer here, or never was",
	"Github":           "There isn't a Github Pages site here.",
	"HatenaBlog":       "404 Blog is not found",
	"Help Juice":       "We could not find what you're looking for.",
	"Help Scout":       "No settings were found for this company:",
	"Heroku":           "No such app",
	"Intercom":         "Uh oh. That page doesn't exist.",
	"JetBrains":        "is not a registered InCloud YouTrack",
	"Kinsta":           "No Site For Domain",
	"LaunchRock":       "It looks like you may have taken a wrong turn somewhere. Don't worry...it happens to all of us.",
	"Mashery":          "Unrecognized domain",
	"Pantheon":         "404 error unknown site!",
	"Readme.io":        "Project doesnt exist... yet!",
	"Shopify":          "Sorry, this shop is currently unavailable.",
	"Statuspage":       "Visiting the subdomain will redirect users to https://www.statuspage.io.",
	"Strikingly":       "page not found",
	"Surge.sh":         "project not found",
	"Tumblr":           "Whatever you were looking for doesn't currently exist at this address",
	"Tilda":            "Please renew your subscription",
	"Unbounce":         "The requested URL was not found on this server.",
	"Uptimerobot":      "page not found",
	"UserVoice":        "This UserVoice subdomain is currently available!",
	"Wordpress":        "Do you want to register",
	"Zendesk":          "Help Center Closed",
	"Example.com":      "This domain is for use in illustrative examples in documents.", //for tests
}

func SubCheck(body []byte) string {
	var s string
	for _, val := range fingerprints {
		if bytes.Contains(body, []byte(val)) {
			s = "Possible vulnerable"
			break
		} else {
			s = "Not vulnerable"
		}
	}
	return s
}

func getCNAME(u string) string {
	var cname string
	s, _ := url.Parse(u)
	d := new(dns.Msg)
	d.SetQuestion(s.Hostname()+".", dns.TypeCNAME)
	ret, err := dns.Exchange(d, "8.8.8.8:53")
	if err != nil {
		return cname
	}
	for _, a := range ret.Answer {
		if t, ok := a.(*dns.CNAME); ok {
			cname = t.Target
		}
	}
	return cname
}
