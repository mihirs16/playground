# The edge: the single CloudFront distribution the whole playground fronts, its
# public TLS certificate, and the DNS that makes custodian reachable same-origin
# over HTTPS at its domain (ADR-0001). This ticket wires custodian as the API
# origin; persona's private origin and the path routing land in ticket 06.

locals {
  origin_fqdn = "origin.${var.custodian_domain_name}"
}

# The apex zone covering the whole domain. Registration stays at Squarespace; this
# zone is the delegation target. Its name servers (route53_name_servers output)
# must be set at the registrar before the ACM validation below can resolve — until
# delegation is live, an apply blocks on aws_acm_certificate_validation. The apex
# is persona's (ticket 06); custodian is the custodian_domain_name subdomain here.
resource "aws_route53_zone" "playground" {
  name = var.zone_name
}

# A stable address for the origin. The box's auto-assigned public IP changes on
# every stop/start, but CloudFront needs a durable origin hostname — so the box
# gets an Elastic IP and an A record that never moves.
resource "aws_eip" "box" {
  instance = aws_instance.box.id
  domain   = "vpc"

  tags = {
    Name = "custodian-origin"
  }
}

resource "aws_route53_record" "origin" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = local.origin_fqdn
  type    = "A"
  ttl     = 300
  records = [aws_eip.box.public_ip]
}

# AWS holds the private key — no certificate material ever lands on disk. Issued in
# us-east-1 because that is the only region CloudFront reads viewer certificates
# from, and DNS-validated through the zone above.
resource "aws_acm_certificate" "edge" {
  provider          = aws.us_east_1
  domain_name       = var.custodian_domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.edge.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id         = aws_route53_zone.playground.zone_id
  name            = each.value.name
  type            = each.value.type
  ttl             = 60
  records         = [each.value.record]
  allow_overwrite = true
}

resource "aws_acm_certificate_validation" "edge" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.edge.arn
  validation_record_fqdns = [for r in aws_route53_record.cert_validation : r.fqdn]
}

# The box terminates plain HTTP behind nginx and holds no origin certificate, so
# the edge<->origin hop is HTTP. Only CloudFront can make that hop: the box's
# security group pins ingress to this managed prefix list (main.tf).
data "aws_ec2_managed_prefix_list" "cloudfront_origin_facing" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

# The API default behavior does not cache and forwards the whole viewer request —
# cookies, Authorization, query string — except the Host header, so custodian's
# auth and same-origin cookies survive the edge. persona's cacheable /logs/
# behavior is added in ticket 06.
data "aws_cloudfront_cache_policy" "caching_disabled" {
  name = "Managed-CachingDisabled"
}

data "aws_cloudfront_origin_request_policy" "all_viewer_except_host" {
  name = "Managed-AllViewerExceptHostHeader"
}

resource "aws_cloudfront_distribution" "edge" {
  enabled         = true
  is_ipv6_enabled = true
  aliases         = [var.custodian_domain_name]
  price_class     = "PriceClass_100"
  comment         = "playground edge — custodian API origin"

  origin {
    origin_id   = "custodian"
    domain_name = local.origin_fqdn

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "http-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = "custodian"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods         = ["GET", "HEAD"]

    cache_policy_id          = data.aws_cloudfront_cache_policy.caching_disabled.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.all_viewer_except_host.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.edge.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}

# Alias records (not a CNAME) point the custodian domain at the distribution: they
# resolve to both A and AAAA and cost nothing per query.
resource "aws_route53_record" "edge_a" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.custodian_domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.edge.domain_name
    zone_id                = aws_cloudfront_distribution.edge.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "edge_aaaa" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.custodian_domain_name
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.edge.domain_name
    zone_id                = aws_cloudfront_distribution.edge.hosted_zone_id
    evaluate_target_health = false
  }
}
