# Records the zone must carry that are nothing to do with the playground, but which
# the delegated zone becomes authoritative for. Losing any of them is an
# outward-facing regression, so they are declared here rather than left behind at
# the old nameservers:
#   - Mailgun email (MX + SPF + DKIM): inbound routing and forwarding live in
#     Mailgun's panel, not DNS; these three keep receiving and sending auth working.
#   - the apex and www: the deprecated Netlify site, kept reachable through the cutover.
#   - monteapi and projects: other live personal services on this domain.
#   - the Google verification CNAME: proves domain ownership to Google (Workspace/
#     Search Console); kept so that verification survives the move off Google's DNS.

resource "aws_route53_record" "mail_mx" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.zone_name
  type    = "MX"
  ttl     = 3600
  records = [
    "10 mxa.mailgun.org",
    "10 mxb.mailgun.org",
  ]
}

resource "aws_route53_record" "mail_spf" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.zone_name
  type    = "TXT"
  ttl     = 3600
  records = ["v=spf1 include:mailgun.org ~all"]
}

# One TXT record whose value exceeds 255 characters. The embedded "" splits it into
# two DNS character-strings that resolvers concatenate back into the single key —
# the AWS provider's required form for a long TXT value.
resource "aws_route53_record" "mail_dkim" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "krs._domainkey.${var.zone_name}"
  type    = "TXT"
  ttl     = 3600
  records = ["k=rsa; p=MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDAYWtKhjL5nBBqvXFLUC3TGLknDr9H42Y7UT3vXE7x8lKx75qne8UsNnJcoVLLCPmk8Y8U0P3sVh6eu1EZzBZ+gQROiGUZYl8+Wzq6FCtXrl/2FJlVdpKWpfMemRyCLcsudZOj3/H6D6bc/23NCSV8Tgb0\"\"pZT52PSGiEK9WypHzQIDAQAB"]
}

resource "aws_route53_record" "www" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "www.${var.zone_name}"
  type    = "CNAME"
  ttl     = 3600
  records = ["mihirsingh.netlify.app"]
}

# The apex still serves the deprecated Netlify site (its apex load-balancer IP).
# When persona takes the apex (ticket 06) this record becomes the distribution
# alias instead.
resource "aws_route53_record" "apex_a" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.zone_name
  type    = "A"
  ttl     = 3600
  records = ["75.2.60.5"]
}

resource "aws_route53_record" "monteapi_a" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "monteapi.${var.zone_name}"
  type    = "A"
  ttl     = 3600
  records = ["176.58.98.219"]
}

resource "aws_route53_record" "monteapi_aaaa" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "monteapi.${var.zone_name}"
  type    = "AAAA"
  ttl     = 3600
  records = ["2a01:7e00::f03c:93ff:fecc:e2f"]
}

resource "aws_route53_record" "projects_a" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "projects.${var.zone_name}"
  type    = "A"
  ttl     = 3600
  records = ["20.193.154.112"]
}

resource "aws_route53_record" "google_verification" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = "g2vwt3sn5lde.${var.zone_name}"
  type    = "CNAME"
  ttl     = 3600
  records = ["gv-ary5fopz4lgpye.dv.googlehosted.com"]
}
